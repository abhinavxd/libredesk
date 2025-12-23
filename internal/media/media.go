// Package media provides functionality for managing files backed by fs or S3.
package media

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/image"
	"github.com/abhinavxd/libredesk/internal/media/models"
	"github.com/gabriel-vasile/mimetype"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/knadh/go-i18n"
	"github.com/volatiletech/null/v9"
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
	GetURL(name, disposition, fileName string) string
	GetBlob(name string) ([]byte, error)
	Name() string
}

type Manager struct {
	store   Store
	lo      *logf.Logger
	i18n    *i18n.I18n
	queries queries
}

// Opts provides options for configuring the Manager.
type Opts struct {
	Store Store
	Lo    *logf.Logger
	DB    *sqlx.DB
	I18n  *i18n.I18n
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
	var (
		uuid = uuid.New()
		err  error
	)

	// Override content type after upload (in case it was detected incorrectly).
	_, contentType, err = m.Upload(uuid.String(), contentType, content)
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

// Upload saves the media file to the storage backend - returns the generated filename and content type (after detection).
func (m *Manager) Upload(fileName, contentType string, content io.ReadSeeker) (string, string, error) {
	// On store file is named by UUID to avoid collisions and the actual filename is stored in DB.
	m.lo.Debug("detecting content type for file before upload", "uuid", fileName, "source_content_type", contentType)

	// Detect content type and override if needed.
	contentType, err := m.detectContentType(contentType, content)
	if err != nil {
		m.lo.Error("error detecting content type", "error", err)
		return "", "", err
	}

	fName, err := m.store.Put(fileName, contentType, content)
	if err != nil {
		m.lo.Error("error uploading media", "error", err)
		return "", "", envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorUploading", "name", "{globals.terms.media}"), nil)
	}
	return fName, contentType, nil
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
	media.URL = m.GetURL(media.UUID, media.ContentType, media.Filename)
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
func (m *Manager) GetURL(uuid, contentType, fileName string) string {
	// Keep some content types inline.
	disposition := "attachment"
	if strings.HasPrefix(contentType, "image/") ||
		strings.HasPrefix(contentType, "video/") ||
		contentType == "application/pdf" {
		disposition = "inline"
	}
	return m.store.GetURL(uuid, disposition, fileName)
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

	// Thumbnail files do not exist in the database, only in the storage backend, so return early.
	if strings.HasPrefix(name, image.ThumbPrefix) {
		return nil
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

		// If it's an image, also delete the `thumb_uuid` image from store.
		if strings.HasPrefix(mm.ContentType, "image/") {
			thumbUUID := image.ThumbPrefix + mm.UUID
			m.lo.Debug("deleting thumbnail for unlinked media", "thumb_uuid", thumbUUID)
			if err := m.Delete(thumbUUID); err != nil {
				m.lo.Error("error deleting thumbnail for unlinked media", "error", err)
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
