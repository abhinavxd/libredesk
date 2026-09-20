package resourceimage

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"time"

	mmodels "github.com/abhinavxd/libredesk/internal/media/models"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type mediaStore interface {
	Upload(string, string, io.ReadSeeker) (string, string, error)
	GetBlob(string) ([]byte, error)
	Delete(string) error
}

type Store struct {
	db       *sqlx.DB
	media    mediaStore
	provider string
	fetch    func(context.Context, string, func() error) ([]byte, error)
}

func NewStore(db *sqlx.DB, media mediaStore, provider string) *Store {
	return &Store{db: db, media: media, provider: provider, fetch: NewFetcher().FetchAuthorized}
}

func (s *Store) Get(ctx context.Context, messageID int, sourceID, source string, authorize func(*sqlx.Tx) error) ([]byte, error) {
	return s.get(ctx, mmodels.ModelResourceImages, messageID, sourceID, source, authorize)
}

func (s *Store) AvatarOwner(ctx context.Context, source string) (int, error) {
	var id int
	err := s.db.GetContext(ctx, &id, "SELECT id FROM users WHERE avatar_url = $1 ORDER BY id LIMIT 1", source)
	return id, err
}

func (s *Store) GetAvatar(ctx context.Context, userID int, sourceID, source string, authorize func(*sqlx.Tx) error) ([]byte, error) {
	return s.get(ctx, mmodels.ModelResourceAvatars, userID, sourceID, source, authorize)
}

func (s *Store) get(ctx context.Context, model string, ownerID int, sourceID, source string, authorize func(*sqlx.Tx) error) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var existingID int
	var ownerErr error
	if model == mmodels.ModelResourceAvatars {
		ownerErr = tx.GetContext(ctx, &existingID, "SELECT id FROM users WHERE id = $1 AND avatar_url = $2 FOR SHARE", ownerID, source)
	} else {
		ownerErr = tx.GetContext(ctx, &existingID, "SELECT id FROM conversation_messages WHERE id = $1 FOR KEY SHARE", ownerID)
	}
	if err := ownerErr; err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, fmt.Sprintf("%s:%d:%s", model, ownerID, sourceID)); err != nil {
		return nil, err
	}
	if authorize != nil {
		if err := authorize(tx); err != nil {
			return nil, err
		}
	}
	var name string
	err = tx.GetContext(ctx, &name, `SELECT uuid FROM media WHERE model_type = $1 AND model_id = $2 AND content_id = $3`, model, ownerID, sourceID)
	if err == nil {
		body, err := s.media.GetBlob(name)
		if err != nil {
			return nil, err
		}
		if len(body) > MaxImageBytes {
			return nil, ErrImage
		}
		if err := tx.Commit(); err != nil {
			return nil, err
		}
		return body, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	body, err := s.fetch(ctx, source, func() error {
		if authorize != nil {
			return authorize(tx)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if authorize != nil {
		if err := authorize(tx); err != nil {
			return nil, err
		}
	}
	name, _, err = s.media.Upload(uuid.NewString(), "image/png", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO media (store, filename, content_type, size, meta, model_id, model_type, disposition, content_id, uuid, private)
 VALUES ($1, 'external-image.png', 'image/png', $2, '{}'::jsonb, $3, $6, 'inline', $4, $5, true)`, s.provider, len(body), ownerID, sourceID, name, model)
	if err != nil {
		tx.Rollback()
		s.media.Delete(name)
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return body, nil
}

func (s *Store) MessageAllowed(ctx context.Context, userID, messageID int, contentHash string) (bool, error) {
	return s.MessageAllowedTx(ctx, nil, userID, messageID, contentHash)
}

func (s *Store) MessageAllowedTx(ctx context.Context, tx *sqlx.Tx, userID, messageID int, contentHash string) (bool, error) {
	var query sqlx.QueryerContext = s.db
	if tx != nil {
		query = tx
	}
	var allowed bool
	err := sqlx.GetContext(ctx, query, &allowed, `SELECT EXISTS (SELECT 1 FROM message_image_permissions WHERE user_id = $1 AND message_id = $2 AND content_hash = $3)`, userID, messageID, contentHash)
	return allowed, err
}

func (s *Store) AllowMessage(ctx context.Context, userID, messageID int, contentHash string) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO message_image_permissions (user_id, message_id, content_hash) VALUES ($1, $2, $3)
 ON CONFLICT (user_id, message_id) DO UPDATE SET content_hash = EXCLUDED.content_hash`, userID, messageID, contentHash)
	return err
}
