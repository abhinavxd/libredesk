package resourceimage

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	mmodels "github.com/abhinavxd/libredesk/internal/media/models"
	"github.com/abhinavxd/libredesk/internal/resourcepolicy"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type mediaStore interface {
	Upload(string, string, io.ReadSeeker) (string, string, error)
	GetBlob(string) ([]byte, error)
	Delete(string) error
}

type Store struct {
	db         *sqlx.DB
	media      mediaStore
	provider   string
	fetch      func(context.Context, string, func() error) ([]byte, error)
	slots      chan struct{}
	mu         sync.Mutex
	active     map[string]chan struct{}
	readPolicy func(*sqlx.Tx) (resourcepolicy.Config, error)
	cleanup    chan struct{}
}

func NewStore(db *sqlx.DB, media mediaStore, provider string, readPolicy func(*sqlx.Tx) (resourcepolicy.Config, error)) *Store {
	return &Store{
		db:         db,
		readPolicy: readPolicy,
		cleanup:    make(chan struct{}, 1),
		media:      media,
		provider:   provider,
		fetch:      NewFetcher().FetchAuthorized,
		slots:      make(chan struct{}, 4),
		active:     make(map[string]chan struct{}),
	}
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

// acquire serializes fills for one image and bounds all storage work before it
// can borrow a database connection. Waiting requests hold no database resources.
func (s *Store) acquire(ctx context.Context, key string) (func(), error) {
	for {
		s.mu.Lock()
		done, busy := s.active[key]
		if !busy {
			done = make(chan struct{})
			s.active[key] = done
		}
		s.mu.Unlock()
		if !busy {
			releaseKey := func() {
				s.mu.Lock()
				delete(s.active, key)
				close(done)
				s.mu.Unlock()
			}
			select {
			case s.slots <- struct{}{}:
				return func() {
					<-s.slots
					releaseKey()
				}, nil
			case <-ctx.Done():
				releaseKey()
				return nil, ctx.Err()
			}
		}
		select {
		case <-done:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

func (s *Store) get(ctx context.Context, model string, ownerID int, sourceID, source string, authorize func(*sqlx.Tx) error) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	release, err := s.acquire(ctx, fmt.Sprintf("%s:%d:%s", model, ownerID, sourceID))
	if err != nil {
		return nil, err
	}
	defer release()

	lookup := func() (string, error) {
		return s.lookup(ctx, model, ownerID, sourceID, source, authorize)
	}
	check := func() error {
		_, err := lookup()
		return err
	}
	read := func(name string) ([]byte, error) {
		body, err := s.media.GetBlob(name)
		if err != nil {
			return nil, err
		}
		if len(body) == 0 || len(body) > MaxImageBytes {
			return nil, ErrImage
		}
		// Permissions and ownership may change while storage is being read.
		if err := check(); err != nil {
			return nil, err
		}
		return body, nil
	}
	name, err := lookup()
	if err != nil {
		return nil, err
	}
	if name != "" {
		body, readErr := read(name)
		if readErr == nil {
			return body, nil
		}
		// An eviction may remove the blob after lookup. Treat that as a miss.
		current, err := lookup()
		if err != nil {
			return nil, err
		}
		if current == name {
			return nil, readErr
		}
		if current != "" {
			return read(current)
		}
	}

	// The fetcher rechecks authorization after waiting for a download slot.
	body, err := s.fetch(ctx, source, check)
	if err != nil {
		return nil, err
	}
	if err := check(); err != nil {
		return nil, err
	}
	if len(body) == 0 || len(body) > MaxImageBytes {
		return nil, ErrImage
	}
	name = uuid.NewString()
	if err := s.reserve(ctx, name, int64(len(body))); err != nil {
		return nil, err
	}
	cleanup := true
	defer func() {
		if cleanup {
			s.discardUpload(name)
		}
	}()

	if err := check(); err != nil {
		return nil, err
	}
	if _, _, err := s.media.Upload(name, "image/png", bytes.NewReader(body)); err != nil {
		return nil, err
	}

	// Publish metadata in a short transaction. Another server may have
	// filled the same cache entry while we fetched; keep its entry in that case.
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if err := lockCache(ctx, tx); err != nil {
		return nil, err
	}
	if err := s.validateReservation(ctx, tx, name); err != nil {
		return nil, err
	}
	if err := s.authorizeOwner(ctx, tx, model, ownerID, source, authorize); err != nil {
		return nil, err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO media (store, filename, content_type, size, meta, model_id, model_type, disposition, content_id, uuid, private, created_at)
 VALUES ($1, 'external-image.png', 'image/png', $2, '{}'::jsonb, $3, $6, 'inline', $4, $5, true, clock_timestamp())
 ON CONFLICT DO NOTHING`, s.provider, len(body), ownerID, sourceID, name, model)
	if err != nil {
		return nil, err
	}
	var savedName string
	if err := tx.GetContext(ctx, &savedName, `SELECT uuid FROM media WHERE model_type = $1 AND model_id = $2 AND content_id = $3`, model, ownerID, sourceID); err != nil {
		return nil, err
	}
	if savedName == name {
		if _, err := tx.ExecContext(ctx, "DELETE FROM resource_image_cache_pending WHERE uuid = $1", name); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		// A connection failure can leave the commit outcome unknown. Do not delete
		// a blob that the transaction may have successfully published.
		cleanup = false
		return nil, err
	}
	if savedName != name {
		return read(savedName)
	}
	cleanup = false
	return body, nil
}

func (s *Store) authorizeOwner(ctx context.Context, tx *sqlx.Tx, model string, ownerID int, source string, authorize func(*sqlx.Tx) error) error {
	var id int
	var err error
	if model == mmodels.ModelResourceAvatars {
		err = tx.GetContext(ctx, &id, "SELECT id FROM users WHERE id = $1 AND avatar_url = $2 FOR SHARE", ownerID, source)
	} else {
		err = tx.GetContext(ctx, &id, "SELECT id FROM conversation_messages WHERE id = $1 FOR KEY SHARE", ownerID)
	}
	if err != nil {
		return err
	}
	if authorize != nil {
		return authorize(tx)
	}
	return nil
}

// lookup closes its transaction before the caller starts any network or storage I/O.
func (s *Store) lookup(ctx context.Context, model string, ownerID int, sourceID, source string, authorize func(*sqlx.Tx) error) (string, error) {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	if err := s.authorizeOwner(ctx, tx, model, ownerID, source, authorize); err != nil {
		return "", err
	}
	var name string
	err = tx.GetContext(ctx, &name, `SELECT uuid FROM media WHERE model_type = $1 AND model_id = $2 AND content_id = $3`, model, ownerID, sourceID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return "", err
	}
	if err := tx.Commit(); err != nil {
		return "", err
	}
	return name, nil
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
