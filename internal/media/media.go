// Package media provides functionality for managing files backed by fs or S3.
package media

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/media/models"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/knadh/go-i18n"
	"github.com/volatiletech/null/v9"
	"github.com/zerodha/fastglue"
	"github.com/zerodha/logf"
)

var (
	//go:embed queries.sql
	efs embed.FS
)

// Store defines the interface for media storage operations.
type Store interface {
	Put(name, contentType string, content io.ReadSeeker) (string, error)
	Delete(name string) error
	GetURL(name string) string
	GetBlob(name string) ([]byte, error)
	Name() string
}

// SignedURLStore defines the interface for stores that support signed URLs.
// This is optional and only implemented by stores that need signed URL functionality (like fs).
type SignedURLStore interface {
	Store
	GetSignedURL(name string, expiresAt time.Time, secret []byte) string
	VerifySignature(name, signature string, expiresAt time.Time, secret []byte) bool
}

type Manager struct {
	store   Store
	lo      *logf.Logger
	i18n    *i18n.I18n
	queries queries
	secret  string
}

// Opts provides options for configuring the Manager.
type Opts struct {
	Store  Store
	Lo     *logf.Logger
	DB     *sqlx.DB
	I18n   *i18n.I18n
	Secret string
}

// New initializes and returns a new Manager instance for handling media operations.
func New(opt Opts) (*Manager, error) {
	var q queries
	if err := dbutil.ScanSQLFile("queries.sql", &q, opt.DB, efs); err != nil {
		return nil, err
	}
	return &Manager{
		store:   opt.Store,
		lo:      opt.Lo,
		i18n:    opt.I18n,
		queries: q,
		secret:  opt.Secret,
	}, nil
}

// queries holds the prepared SQL statements.
type queries struct {
	Insert                  *sqlx.Stmt `query:"insert-media"`
	Get                     *sqlx.Stmt `query:"get-media"`
	GetByUUID               *sqlx.Stmt `query:"get-media-by-uuid"`
	Delete                  *sqlx.Stmt `query:"delete-media"`
	Attach                  *sqlx.Stmt `query:"attach-to-model"`
	GetByModel              *sqlx.Stmt `query:"get-model-media"`
	GetUnlinkedMessageMedia *sqlx.Stmt `query:"get-unlinked-message-media"`
	ContentIDExists         *sqlx.Stmt `query:"content-id-exists"`
}

// UploadAndInsert uploads file on storage and inserts an entry in db.
func (m *Manager) UploadAndInsert(srcFilename, contentType, contentID string, modelType null.String, modelID null.Int, content io.ReadSeeker, fileSize int, disposition null.String, meta []byte) (models.Media, error) {
	var uuid = uuid.New()
	_, err := m.Upload(uuid.String(), contentType, content)
	if err != nil {
		return models.Media{}, err
	}

	media, err := m.Insert(disposition, srcFilename, contentType, contentID, modelType, uuid.String(), modelID, fileSize, meta)
	if err != nil {
		m.store.Delete(uuid.String())
		return models.Media{}, err
	}
	return media, nil
}

// Upload saves the media file to the storage backend and returns the generated filename.
func (m *Manager) Upload(fileName, contentType string, content io.ReadSeeker) (string, error) {
	fName, err := m.store.Put(fileName, contentType, content)
	if err != nil {
		m.lo.Error("error uploading media", "error", err)
		return "", envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorUploading", "name", "{globals.terms.media}"), nil)
	}
	return fName, nil
}

// Insert inserts media details into the database and returns the inserted media record.
func (m *Manager) Insert(disposition null.String, fileName, contentType, contentID string, modelType null.String, uuid string, modelID null.Int, fileSize int, meta []byte) (models.Media, error) {
	var id int
	if err := m.queries.Insert.QueryRow(m.store.Name(), fileName, contentType, fileSize, meta, modelID, modelType, disposition, contentID, uuid).Scan(&id); err != nil {
		m.lo.Error("error inserting media", "error", err)
		return models.Media{}, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorInserting", "name", "{globals.terms.media}"), nil)
	}
	return m.Get(id, "")
}

// Get retrieves the media record by its ID and returns the media.
func (m *Manager) Get(id int, uuid string) (models.Media, error) {
	var media models.Media
	if err := m.queries.Get.Get(&media, id, uuid); err != nil {
		if err == sql.ErrNoRows {
			return media, envelope.NewError(envelope.NotFoundError, m.i18n.Ts("globals.messages.notFound", "name", "{globals.terms.media}"), nil)
		}
		m.lo.Error("error fetching media", "error", err)
		return media, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.media}"), nil)
	}
	media.URL = m.store.GetURL(media.UUID)
	return media, nil
}

// ContentIDExists checks if a content_id exists in the database and returns the UUID of the media file.
func (m *Manager) ContentIDExists(contentID string) (bool, string, error) {
	var uuid string
	if err := m.queries.ContentIDExists.Get(&uuid, contentID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, "", nil
		}
		m.lo.Error("error checking if content_id exists", "error", err)
		return false, "", fmt.Errorf("checking if content_id exists: %w", err)
	}
	return true, uuid, nil
}

// GetBlob retrieves the raw binary content of a media file by its name.
func (m *Manager) GetBlob(name string) ([]byte, error) {
	return m.store.GetBlob(name)
}

// GetURL returns the URL for accessing a media file by its name.
func (m *Manager) GetURL(name string) string {
	return m.store.GetURL(name)
}

// Attach associates a media file with a specific model by its ID and model name.
func (m *Manager) Attach(id int, model string, modelID int) error {
	if _, err := m.queries.Attach.Exec(id, model, modelID); err != nil {
		m.lo.Error("error attaching media to model", "model", model, "model_id", modelID, "media_id", id, "error", err)
		return fmt.Errorf("attaching media;%d to model:%s model_id:%d: %w", id, model, modelID, err)
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
			return envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorDeleting", "name", "{globals.terms.media}"), nil)
		}
	}
	// Delete the media record from the database.
	if _, err := m.queries.Delete.Exec(name); err != nil {
		m.lo.Error("error deleting media from db", "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorDeleting", "name", "{globals.terms.media}"), nil)
	}
	return nil
}

// DeleteUnlinkedMedia is a blocking function that periodically deletes media files that are not linked to any conversation message.
func (m *Manager) DeleteUnlinkedMedia(ctx context.Context) {
	m.deleteUnlinkedMessageMedia()
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(12 * time.Hour):
			m.lo.Info("starting periodic deletion of unlinked media")
			if err := m.deleteUnlinkedMessageMedia(); err != nil {
				m.lo.Error("error deleting unlinked media", "error", err)
			}
		}
	}
}

// deleteUnlinkedMessageMedia fetches all media files that are not linked to any message and deletes them from the storage backend and the database.
func (m *Manager) deleteUnlinkedMessageMedia() error {
	var media []models.Media
	if err := m.queries.GetUnlinkedMessageMedia.Select(&media); err != nil {
		m.lo.Error("error fetching unlinked media", "error", err)
		return err
	}
	for _, mm := range media {
		m.lo.Debug("deleting media not linked to any message", "media_id", mm.ID)
		if err := m.Delete(mm.UUID); err != nil {
			m.lo.Error("error deleting unlinked media", "error", err)
			continue
		}
	}
	return nil
}

// GetSignedURL returns a signed URL for accessing a media file with expiration.
// This delegates to the store if it supports signed URLs (like fs), otherwise returns the normal URL.
func (m *Manager) GetSignedURL(name string, expiresAt time.Time) string {
	// Check if the store supports signed URLs
	if signedStore, ok := m.store.(SignedURLStore); ok {
		return signedStore.GetSignedURL(name, expiresAt, []byte(m.secret))
	}
	// Fallback to regular URL for stores that handle signing internally (like S3)
	return m.store.GetURL(name)
}

// VerifySignature verifies a signed URL signature using the request parameters.
// This is used by middleware to verify widget media access.
func (m *Manager) VerifySignature(r *fastglue.Request) error {
	uuid := r.RequestCtx.UserValue("uuid")
	if uuid == nil {
		return fmt.Errorf("missing uuid parameter")
	}
	
	signature := string(r.RequestCtx.QueryArgs().Peek("signature"))
	expiresStr := string(r.RequestCtx.QueryArgs().Peek("expires"))
	
	if signature == "" || expiresStr == "" {
		return fmt.Errorf("missing signature or expires parameter")
	}
	
	// Parse expiration time
	var expires int64
	if _, err := fmt.Sscanf(expiresStr, "%d", &expires); err != nil {
		return fmt.Errorf("invalid expires parameter: %v", err)
	}
	
	expiresAt := time.Unix(expires, 0)
	
	// Check if store supports signature verification
	if signedStore, ok := m.store.(SignedURLStore); ok {
		// Strip thumb_ prefix for signature verification to match the base UUID
		verificationName := strings.TrimPrefix(uuid.(string), "thumb_")
		if !signedStore.VerifySignature(verificationName, signature, expiresAt, []byte(m.secret)) {
			return fmt.Errorf("signature verification failed")
		}
		return nil
	}
	
	// For stores that don't support signing (like S3), always allow
	return nil
}
