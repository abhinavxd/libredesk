// Package media provides functionality for managing files backed by fs or S3.
package media

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/image"
	"github.com/abhinavxd/libredesk/internal/media/models"
	"github.com/gabriel-vasile/mimetype"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/knadh/go-i18n"
	"github.com/lib/pq"
	"github.com/volatiletech/null/v9"
	"github.com/zerodha/logf"
)

// PublicURI is the app route that serves uploaded media.
const PublicURI = "/uploads"

const storageUsageSQL = `SELECT COALESCE(SUM(size) FILTER (
 WHERE model_type IS NULL OR model_type NOT IN ('resource_images', 'resource_avatars')
), 0) FROM media`

const storageQuotaLockSQL = `SELECT pg_advisory_xact_lock(hashtextextended('libredesk:durable-media-quota', 0))`

// StorageUsage is the logical durable-media usage tracked in the media table.
// External resource images and avatars are excluded because they have their
// own independently evictable cache budget.
type StorageUsage struct {
	UsedBytes    int64    `json:"used_bytes"`
	LimitBytes   *int64   `json:"limit_bytes"`
	UsagePercent *float64 `json:"usage_percent"`
	LimitSource  string   `json:"limit_source"`
}

var (
	//go:embed queries.sql
	efs embed.FS

	publicMediaURLRe = regexp.MustCompile(`/uploads/([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12})`)
)

// Store defines the interface for media storage operations.
type Store interface {
	Put(name, contentType string, content io.ReadSeeker) (string, error)
	Delete(name string) error
	GetURL(name, disposition, fileName string) string
	GetBlob(name string) ([]byte, error)
	Open(name string) (io.ReadCloser, error)
	Name() string
	// SignedURLValidator returns a validator function if the store supports signed URLs.
	// Returns nil if the store doesn't use signed URLs (e.g., S3 handles validation itself).
	SignedURLValidator() func(name, sig string, exp int64) bool
}

// SignedURLStore defines the interface for stores that support signed URLs.
// This is optional and only implemented by stores that need signed URL functionality (like fs).
type SignedURLStore interface {
	Store
	GetSignedURL(name string) string
}

type Manager struct {
	db                    *sqlx.DB
	store                 Store
	lo                    *logf.Logger
	i18n                  *i18n.I18n
	rootURL               func() string
	signingKey            string
	urlExpiry             time.Duration
	queries               queries
	maxStorageBytes       int64
	maxStorageMu          sync.RWMutex
	storageFullMu         sync.Mutex
	lastStorageFullNotice time.Time
	storageFullNotifier   func()
}

// Opts provides options for configuring the Manager.
type Opts struct {
	Store           Store
	Lo              *logf.Logger
	DB              *sqlx.DB
	I18n            *i18n.I18n
	RootURL         func() string
	SigningKey      string
	URLExpiry       time.Duration
	MaxStorageBytes int64
}

// New initializes and returns a new Manager instance for handling media operations.
func New(opt Opts) (*Manager, error) {
	var q queries
	if err := dbutil.ScanSQLFile("queries.sql", &q, opt.DB, efs); err != nil {
		return nil, err
	}
	return &Manager{
		db:              opt.DB,
		store:           opt.Store,
		lo:              opt.Lo,
		i18n:            opt.I18n,
		rootURL:         opt.RootURL,
		signingKey:      opt.SigningKey,
		urlExpiry:       opt.URLExpiry,
		queries:         q,
		maxStorageBytes: opt.MaxStorageBytes,
	}, nil
}

// SetStorageFullNotifier registers a throttled callback invoked when a durable
// media write is rejected by the configured quota.
func (m *Manager) SetStorageFullNotifier(notifier func()) {
	m.storageFullMu.Lock()
	defer m.storageFullMu.Unlock()
	m.storageFullNotifier = notifier
}

func (m *Manager) notifyStorageFull() {
	m.storageFullMu.Lock()
	if m.storageFullNotifier == nil || time.Since(m.lastStorageFullNotice) < time.Minute {
		m.storageFullMu.Unlock()
		return
	}
	m.lastStorageFullNotice = time.Now()
	notifier := m.storageFullNotifier
	m.storageFullMu.Unlock()
	notifier()
}

func (m *Manager) storageFullError() error {
	message := "Durable media storage is full. New uploads are temporarily disabled."
	if m.i18n != nil {
		message = m.i18n.T("media.storageFull")
	}
	return envelope.NewError(envelope.StorageFullError, message, nil)
}

// SetMaxStorageBytes changes the durable-media quota for subsequent writes.
// A zero value disables the application-level quota.
func (m *Manager) SetMaxStorageBytes(limit int64) {
	m.maxStorageMu.Lock()
	m.maxStorageBytes = limit
	m.maxStorageMu.Unlock()
}

func (m *Manager) mediaGenericError() error {
	message := "Something went wrong."
	if m.i18n != nil {
		message = m.i18n.T("globals.messages.somethingWentWrong")
	}
	return envelope.NewError(envelope.GeneralError, message, nil)
}

// GetStorageUsage returns durable-media usage and the configured quota.
func (m *Manager) GetStorageUsage() (StorageUsage, error) {
	var used int64
	if err := m.db.Get(&used, storageUsageSQL); err != nil {
		return StorageUsage{}, err
	}
	usage := StorageUsage{UsedBytes: used, LimitSource: "application"}
	m.maxStorageMu.RLock()
	limit := m.maxStorageBytes
	m.maxStorageMu.RUnlock()
	if limit > 0 {
		usage.LimitBytes = &limit
		percentage := float64(used) * 100 / float64(limit)
		usage.UsagePercent = &percentage
	}
	return usage, nil
}

// queries holds the prepared SQL statements.
type queries struct {
	GetUnlinkedResourceAvatars  *sqlx.Stmt `query:"get-unlinked-resource-avatars"`
	GetUnlinkedResourceImages   *sqlx.Stmt `query:"get-unlinked-resource-images"`
	Insert                      *sqlx.Stmt `query:"insert-media"`
	Get                         *sqlx.Stmt `query:"get-media"`
	GetByUUID                   *sqlx.Stmt `query:"get-media-by-uuid"`
	Delete                      *sqlx.Stmt `query:"delete-media"`
	LinkMessageMedia            *sqlx.Stmt `query:"link-message-media"`
	GetByModel                  *sqlx.Stmt `query:"get-model-media"`
	GetUnlinkedMessageMedia     *sqlx.Stmt `query:"get-unlinked-message-media"`
	GetUnlinkedHelpArticleMedia *sqlx.Stmt `query:"get-unlinked-help-article-media"`
	LinkHelpArticleMedia        *sqlx.Stmt `query:"link-help-article-media"`
	UnlinkHelpArticleMedia      *sqlx.Stmt `query:"unlink-help-article-media"`
	ContentIDExists             *sqlx.Stmt `query:"content-id-exists"`
	GetByContentIDs             *sqlx.Stmt `query:"get-media-by-content-ids"`
	GetDraftInlineMedia         *sqlx.Stmt `query:"get-draft-inline-media"`
}

// UploadAndInsert reserves durable capacity in the media table before writing
// any bytes. The short accounting transaction ends before storage I/O; pending
// uploads count against the quota and abandoned rows use the unlinked-media sweep.
func (m *Manager) UploadAndInsert(srcFilename, contentType, contentID string, modelType null.String, modelID null.Int, content io.ReadSeeker, fileSize int, disposition null.String, meta []byte, private bool) (models.Media, error) {
	if modelType.String == models.ModelResourceImages || modelType.String == models.ModelResourceAvatars {
		return models.Media{}, fmt.Errorf("external images must use the image cache reservation path")
	}
	// Determine the actual size rather than trusting a caller-supplied estimate.
	size, err := content.Seek(0, io.SeekEnd)
	if err != nil || size < 0 || size > 1<<31-1 {
		return models.Media{}, fmt.Errorf("unable to determine valid upload size")
	}
	if _, err := content.Seek(0, io.SeekStart); err != nil {
		return models.Media{}, err
	}
	contentType, err = m.detectContentType(contentType, content)
	if err != nil {
		return models.Media{}, err
	}
	if !modelType.Valid || modelType.String == "" {
		modelType = null.StringFrom(models.ModelMessages)
	}
	name := uuid.NewString()
	reserved, err := m.Insert(disposition, srcFilename, contentType, contentID, modelType, name, modelID, int(size), meta, private)
	if err != nil {
		return models.Media{}, err
	}
	if _, _, err := m.Upload(name, contentType, content); err != nil {
		// Delete removes the row only once the blob has been removed. A failed
		// cleanup remains accounted for and is retried by the existing sweep.
		if cleanupErr := m.Delete(name); cleanupErr != nil {
			m.lo.Error("error cleaning up failed durable upload", "uuid", name, "error", cleanupErr)
		}
		return models.Media{}, err
	}
	return reserved, nil
}

// Upload saves the media file to the storage backend - returns the generated filename and content type (after detection).
func (m *Manager) Upload(fileName, contentType string, content io.ReadSeeker) (string, string, error) {
	// On store file is named by UUID to avoid collisions and the actual filename is stored in DB.
	m.lo.Debug("detecting content type for file before upload", "uuid", fileName, "source_content_type", contentType)

	// Detect content type and override if needed.
	contentType, err := m.detectContentType(contentType, content)
	if err != nil {
		m.lo.Error("error detecting content type", "error", err, "file_name", fileName, "content_type", contentType, "store", m.store.Name())
		return "", "", err
	}

	fName, err := m.store.Put(fileName, contentType, content)
	if err != nil {
		// Providers may leave a partial object when a write fails. Keep the
		// original name: Put commonly returns an empty name on error.
		if cleanupErr := m.store.Delete(fileName); cleanupErr != nil && !errors.Is(cleanupErr, os.ErrNotExist) {
			m.lo.Error("error removing partial upload", "uuid", fileName, "error", cleanupErr)
		}
		if errors.Is(err, syscall.ENOSPC) || errors.Is(err, syscall.EDQUOT) {
			m.notifyStorageFull()
			return "", "", m.storageFullError()
		}
		m.lo.Error("error uploading media to store", "error", err, "file_name", fileName, "content_type", contentType, "store", m.store.Name())
		return "", "", envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.errorUploadingFile"), nil)
	}
	return fName, contentType, nil
}

// Insert inserts media details into the database and returns the inserted media record.
func (m *Manager) Insert(disposition null.String, fileName, contentType, contentID string, modelType null.String, uuid string, modelID null.Int, fileSize int, meta []byte, private bool) (models.Media, error) {
	var id int
	m.maxStorageMu.RLock()
	maxStorageBytes := m.maxStorageBytes
	m.maxStorageMu.RUnlock()
	if maxStorageBytes > 0 {
		tx, err := m.db.Beginx()
		if err != nil {
			return models.Media{}, m.mediaGenericError()
		}
		defer tx.Rollback()
		if _, err := tx.Exec(storageQuotaLockSQL); err != nil {
			return models.Media{}, m.mediaGenericError()
		}
		var used int64
		if err := tx.Get(&used, storageUsageSQL); err != nil {
			return models.Media{}, m.mediaGenericError()
		}
		if fileSize < 0 || used+int64(fileSize) > maxStorageBytes {
			m.notifyStorageFull()
			return models.Media{}, m.storageFullError()
		}
		if err := tx.Stmtx(m.queries.Insert).QueryRow(m.store.Name(), fileName, contentType, fileSize, meta, modelID, modelType, disposition, contentID, uuid, private).Scan(&id); err != nil {
			m.lo.Error("error inserting media", "error", err, "file_name", fileName, "content_type", contentType, "store", m.store.Name())
			return models.Media{}, m.mediaGenericError()
		}
		if err := tx.Commit(); err != nil {
			return models.Media{}, m.mediaGenericError()
		}
	} else if err := m.queries.Insert.QueryRow(m.store.Name(), fileName, contentType, fileSize, meta, modelID, modelType, disposition, contentID, uuid, private).Scan(&id); err != nil {
		m.lo.Error("error inserting media", "error", err, "file_name", fileName, "content_type", contentType, "store", m.store.Name())
		return models.Media{}, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return m.Get(id, "")
}

// GetMany fetches multiple media records by their IDs.
func (m *Manager) GetMany(ids []int) ([]models.Media, error) {
	out := make([]models.Media, 0, len(ids))
	for _, id := range ids {
		med, err := m.Get(id, "")
		if err != nil {
			return nil, err
		}
		out = append(out, med)
	}
	return out, nil
}

// Get retrieves the media record by its ID and returns the media.
func (m *Manager) Get(id int, uuid string) (models.Media, error) {
	var media models.Media
	if err := m.queries.Get.Get(&media, id, uuid); err != nil {
		if err == sql.ErrNoRows {
			return media, envelope.NewError(envelope.NotFoundError, m.i18n.T("validation.notFoundMedia"), nil)
		}
		m.lo.Error("error fetching media", "error", err)
		return media, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	if media.Private {
		media.URL = m.GetURL(media.UUID, media.ContentType, media.Filename)
	} else {
		media.URL = m.PublicURL(media.UUID)
	}
	return media, nil
}

// PublicURL returns the stable unsigned app URL for a public media file.
func (m *Manager) PublicURL(uuid string) string {
	return strings.TrimRight(m.rootURL(), "/") + PublicURI + "/" + uuid
}

// LinkHelpArticleMedia links media referenced in the article content and unlinks the rest.
func (m *Manager) LinkHelpArticleMedia(articleID int, content string) error {
	uuids := []string{}
	for _, match := range publicMediaURLRe.FindAllStringSubmatch(content, -1) {
		uuids = append(uuids, match[1])
	}
	if _, err := m.queries.LinkHelpArticleMedia.Exec(articleID, pq.Array(uuids)); err != nil {
		m.lo.Error("error linking help article media", "article_id", articleID, "error", err)
		return fmt.Errorf("linking help article media: %w", err)
	}
	if _, err := m.queries.UnlinkHelpArticleMedia.Exec(articleID, pq.Array(uuids)); err != nil {
		m.lo.Error("error unlinking help article media", "article_id", articleID, "error", err)
		return fmt.Errorf("unlinking help article media: %w", err)
	}
	return nil
}

// ContentIDExists reports whether a media row with the given content_id is linked to a message in the given conversation. Scoped this way so an orphan media row (e.g., from a partial failure) doesn't short-circuit a retry into skipping the upload.
func (m *Manager) ContentIDExists(contentID, conversationUUID string) (bool, string, error) {
	if contentID == "" || conversationUUID == "" {
		return false, "", nil
	}
	var uuid string
	if err := m.queries.ContentIDExists.Get(&uuid, contentID, conversationUUID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, "", nil
		}
		m.lo.Error("error checking if content_id exists", "error", err)
		return false, "", fmt.Errorf("checking if content_id exists: %w", err)
	}
	return true, uuid, nil
}

// GetByContentIDs returns media rows matching any of the given content_ids, scoped to the given conversation to prevent cross-conversation lookup.
func (m *Manager) GetByContentIDs(contentIDs []string, conversationUUID string) ([]models.Media, error) {
	out := []models.Media{}
	if len(contentIDs) == 0 || conversationUUID == "" {
		return out, nil
	}
	if err := m.queries.GetByContentIDs.Select(&out, pq.Array(contentIDs), conversationUUID); err != nil {
		m.lo.Error("error fetching media by content_ids", "error", err)
		return nil, fmt.Errorf("fetching media by content_ids: %w", err)
	}
	return out, nil
}

// GetDraftInlineMedia returns media by UUID only if it's unattached or linked to a message in the given conversation.
func (m *Manager) GetDraftInlineMedia(uuid string, conversationID int) (models.Media, error) {
	var media models.Media
	if err := m.queries.GetDraftInlineMedia.Get(&media, uuid, conversationID); err != nil {
		if err == sql.ErrNoRows {
			return media, envelope.NewError(envelope.NotFoundError, m.i18n.T("validation.notFoundMedia"), nil)
		}
		m.lo.Error("error fetching draft inline media", "uuid", uuid, "error", err)
		return media, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return media, nil
}

// GetBlob retrieves the raw binary content of a media file by its name.
func (m *Manager) GetBlob(name string) ([]byte, error) {
	return m.store.GetBlob(name)
}

// GetURL returns the URL for accessing a media file by its name.
func (m *Manager) GetURL(uuid, contentType, fileName string) string {
	if m.signingKey != "" {
		return m.GetSignedURL(uuid)
	}
	// Keep some content types inline. SVG excluded.
	disposition := "attachment"
	if contentType != "image/svg+xml" &&
		(strings.HasPrefix(contentType, "image/") ||
			strings.HasPrefix(contentType, "video/") ||
			contentType == "application/pdf") {
		disposition = "inline"
	}
	return m.store.GetURL(uuid, disposition, fileName)
}

func (m *Manager) GetURLForDownload(uuid, fileName string) string {
	if m.signingKey != "" {
		return m.GetSignedURL(uuid) + "&download=1"
	}
	return m.store.GetURL(uuid, "attachment", fileName)
}

// GetSignedURL generates a signed URL for secure media access if the store supports it.
// Returns a regular URL if the store doesn't support signed URLs.
func (m *Manager) GetSignedURL(name string) string {
	if m.signingKey != "" {
		return m.signedMediaURL(name)
	}
	if signedStore, ok := m.store.(SignedURLStore); ok {
		return signedStore.GetSignedURL(name)
	}
	// Fallback to regular URL if signed URLs not supported
	return m.GetURL(name, "", "")
}

// GetThumbnailURL returns the URL for an image thumbnail.
func (m *Manager) GetThumbnailURL(uuid string) string {
	if m.signingKey != "" || m.store.Name() == "fs" {
		// FS validates thumbnail requests with the original UUID signature.
		u, err := url.Parse(m.GetSignedURL(uuid))
		if err == nil {
			if idx := strings.LastIndex(u.Path, "/"); idx >= 0 {
				u.Path = u.Path[:idx+1] + image.ThumbPrefix + uuid
			} else {
				u.Path = image.ThumbPrefix + uuid
			}
			return u.String()
		}
	}
	return m.store.GetURL(image.ThumbPrefix+uuid, "inline", "")
}

// SignedURLValidator returns the store's signature validator if available.
// Returns nil if the store doesn't support signed URL validation.
func (m *Manager) SignedURLValidator() func(name, sig string, exp int64) bool {
	if m.signingKey != "" {
		return m.validateMediaSignature
	}
	return m.store.SignedURLValidator()
}

// LinkMessageMediaTx links a message's attachments and inline images to it within the given transaction, stamping a content_id on the inline ones.
func (m *Manager) LinkMessageMediaTx(tx *sqlx.Tx, messageID int, media []models.Media, inlineUUIDs []string) error {
	if len(media) == 0 && len(inlineUUIDs) == 0 {
		return nil
	}
	ids := make([]int, 0, len(media))
	for _, med := range media {
		ids = append(ids, med.ID)
	}
	if _, err := tx.Stmtx(m.queries.LinkMessageMedia).Exec(messageID, pq.Array(ids), pq.Array(inlineUUIDs)); err != nil {
		m.lo.Error("error linking media to message", "message_id", messageID, "error", err)
		return fmt.Errorf("linking media to message:%d: %w", messageID, err)
	}
	return nil
}

// GetByModel retrieves all media files attached to a specific model.
func (m *Manager) GetByModel(modelID int, model string) ([]models.Media, error) {
	var media = make([]models.Media, 0)
	if err := m.queries.GetByModel.Select(&media, model, modelID); err != nil {
		m.lo.Error("error getting model media", "model", model, "model_id", modelID, "error", err)
		return nil, fmt.Errorf("fetching media for model:%s model_id:%d: %w", model, modelID, err)
	}
	return media, nil
}

// Delete deletes a media file from both the storage backend and the database.
func (m *Manager) Delete(name string) error {
	if err := m.store.Delete(name); err != nil {
		m.lo.Error("error deleting media from store", "error", err)
		// If the file does not exist, ignore the error.
		if !errors.Is(err, os.ErrNotExist) {
			return envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
		}
	}

	// Thumbnail files do not exist in the database, only in the storage backend, so return early.
	if strings.HasPrefix(name, image.ThumbPrefix) {
		return nil
	}

	// Delete the media record from the database.
	if _, err := m.queries.Delete.Exec(name); err != nil {
		m.lo.Error("error deleting media from db", "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return nil
}

// DeleteUnlinkedMedia is a blocking function that periodically deletes media files that are not linked to any conversation message or help article.
func (m *Manager) DeleteUnlinkedMedia(ctx context.Context) {
	select {
	case <-ctx.Done():
		return
	case <-time.After(60 * time.Second):
	}
	m.deleteUnlinked()
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(12 * time.Hour):
			m.lo.Info("starting periodic deletion of unlinked media")
			m.deleteUnlinked()
		}
	}
}

// deleteUnlinked runs all unlinked-media sweeps.
func (m *Manager) deleteUnlinked() {
	for _, stmt := range []*sqlx.Stmt{m.queries.GetUnlinkedMessageMedia, m.queries.GetUnlinkedHelpArticleMedia, m.queries.GetUnlinkedResourceImages, m.queries.GetUnlinkedResourceAvatars} {
		if err := m.deleteUnlinkedRows(stmt); err != nil {
			m.lo.Error("error deleting unlinked media", "error", err)
		}
	}
}

// deleteUnlinkedRows deletes the media rows returned by the given query from the storage backend and the database.
func (m *Manager) deleteUnlinkedRows(stmt *sqlx.Stmt) error {
	var media []models.Media
	if err := stmt.Select(&media); err != nil {
		m.lo.Error("error fetching unlinked media", "error", err)
		return err
	}
	for _, mm := range media {
		m.lo.Info("deleting unlinked media", "media_id", mm.ID, "uuid", mm.UUID, "filename", mm.Filename, "model_type", mm.Model.String, "model_id", mm.ModelID.Int)
		if err := m.Delete(mm.UUID); err != nil {
			m.lo.Error("error deleting unlinked media", "media_id", mm.ID, "model_type", mm.Model.String, "model_id", mm.ModelID.Int, "error", err)
			continue
		}

		// If it's an image, also delete the `thumb_uuid` image from store.
		if mm.Model.String != models.ModelResourceImages && mm.Model.String != models.ModelResourceAvatars && strings.HasPrefix(mm.ContentType, "image/") {
			thumbUUID := image.ThumbPrefix + mm.UUID
			if err := m.Delete(thumbUUID); err != nil {
				m.lo.Error("error deleting thumbnail for unlinked media", "media_id", mm.ID, "thumb_uuid", thumbUUID, "error", err)
			}
		}
	}
	return nil
}

// detectContentType detects the content type of a file.
// It trusts the source content type unless it's a generic type like application/octet-stream.
// For generic types, it uses http.DetectContentType (stdlib) as a fast path,
// falling back to mimetype library for deeper inspection using magic numbers.
func (m *Manager) detectContentType(sourceContentType string, content io.ReadSeeker) (string, error) {
	// Set default if empty
	if sourceContentType == "" {
		sourceContentType = "application/octet-stream"
	}

	// Handle "image/svg+xml; charset=utf-8", keep just the type.
	if mediaType, _, err := mime.ParseMediaType(sourceContentType); err == nil && mediaType != "" {
		sourceContentType = mediaType
	}

	// Trust source unless it's a generic/useless type
	if sourceContentType != "application/octet-stream" &&
		sourceContentType != "application/data" &&
		sourceContentType != "application/binary" {
		m.lo.Debug("detected media content type from trusted source", "detected_type", sourceContentType)
		return sourceContentType, nil
	}

	// Ensure we're at the start
	content.Seek(0, io.SeekStart)

	// Fast path: stdlib
	buf := make([]byte, 512)
	n, _ := content.Read(buf)
	detected := http.DetectContentType(buf[:n])

	// If stdlib gives a useful type, use it.
	// stdlib defaults to application/octet-stream for unknown types.
	if detected != "application/octet-stream" {
		content.Seek(0, io.SeekStart)
		m.lo.Debug("detected media content type using stdlib", "detected_type", detected, "source_type", sourceContentType)
		return detected, nil
	}

	// Slow path: mimetype library
	content.Seek(0, io.SeekStart)
	mtype, err := mimetype.DetectReader(content)
	if err != nil {
		m.lo.Error("error detecting content type", "error", err)
		content.Seek(0, io.SeekStart)
		return sourceContentType, nil
	}

	detectedType := mtype.String()
	m.lo.Debug("detected media content type using mimetype lib", "detected_type", detectedType, "source_type", sourceContentType)

	content.Seek(0, io.SeekStart)
	return detectedType, nil
}
