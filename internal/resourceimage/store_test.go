package resourceimage

import (
	"context"
	"errors"
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/abhinavxd/libredesk/internal/migrations"
	"github.com/abhinavxd/libredesk/internal/resourcepolicy"
	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/jmoiron/sqlx"
)

type memoryMedia struct {
	mu    sync.Mutex
	blobs map[string][]byte
}

func (m *memoryMedia) Upload(name, _ string, body io.ReadSeeker) (string, string, error) {
	data, err := io.ReadAll(body)
	m.mu.Lock()
	defer m.mu.Unlock()
	m.blobs[name] = data
	return name, "", err
}

func (m *memoryMedia) GetBlob(name string) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	data, ok := m.blobs[name]
	if !ok {
		return nil, errors.New("missing blob")
	}
	return data, nil
}

func (m *memoryMedia) Delete(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.blobs, name)
	return nil
}

func TestPersistentImagesAndPermissions(t *testing.T) {
	db := testutil.NewDB(t, "resource_image_store")
	db.MustExec("DROP TABLE message_image_permissions")
	db.MustExec("DROP INDEX index_media_resource_image_source")
	for range 2 {
		if err := migrations.V2_9_2(db, nil, nil); err != nil {
			t.Fatal(err)
		}
	}
	var userID, inboxID, conversationID, messageID int
	for _, fixture := range []struct {
		dest  *int
		query string
		args  []any
	}{
		{&userID, "INSERT INTO users (type, email, first_name) VALUES ('agent', 'image@example.com', 'Images') RETURNING id", nil},
		{&inboxID, "INSERT INTO inboxes (name, channel) VALUES ('Images', 'email') RETURNING id", nil},
	} {
		if err := db.Get(fixture.dest, fixture.query, fixture.args...); err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Get(&conversationID, "INSERT INTO conversations (contact_id, inbox_id, status_id) VALUES ($1, $2, (SELECT id FROM conversation_statuses LIMIT 1)) RETURNING id", userID, inboxID); err != nil {
		t.Fatal(err)
	}
	if err := db.Get(&messageID, "INSERT INTO conversation_messages (conversation_id, sender_id, sender_type, type, status, content) VALUES ($1, $2, 'contact', 'incoming', 'received', 'image') RETURNING id", conversationID, userID); err != nil {
		t.Fatal(err)
	}
	media := &memoryMedia{blobs: make(map[string][]byte)}
	first := NewStore(db, media, "fs", defaultCachePolicy)
	var calls atomic.Int32
	fetch := func(context.Context, string, func() error) ([]byte, error) {
		calls.Add(1)
		return []byte("saved PNG"), nil
	}
	first.fetch = fetch
	ctx := context.Background()
	results := make(chan error, 8)
	for range 8 {
		go func() {
			body, err := first.Get(ctx, messageID, "source", "https://example.com/image", nil)
			if err == nil && string(body) != "saved PNG" {
				err = errors.New("wrong stored content")
			}
			results <- err
		}()
	}
	for range 8 {
		if err := <-results; err != nil {
			t.Fatal(err)
		}
	}
	if calls.Load() != 1 {
		t.Fatalf("fetched %d times", calls.Load())
	}
	db.SetMaxOpenConns(1)
	tx, err := db.Beginx()
	if err != nil {
		t.Fatal(err)
	}
	allowed, err := first.MessageAllowedTx(ctx, tx, userID, messageID, "unknown")
	tx.Rollback()
	if err != nil || allowed {
		t.Fatalf("transaction-bound message permission: %v, %v", allowed, err)
	}
	restarted := NewStore(db, media, "fs", defaultCachePolicy)
	restarted.fetch = func(context.Context, string, func() error) ([]byte, error) {
		t.Error("refetched persisted image")
		return nil, errors.New("unexpected fetch")
	}
	if _, err := restarted.Get(ctx, messageID, "source", "https://example.com/image", nil); err != nil {
		t.Fatal(err)
	}
	hash := strings.Repeat("a", 64)
	if err := first.AllowMessage(ctx, userID, messageID, hash); err != nil {
		t.Fatal(err)
	}
	for _, check := range []struct {
		user, message int
		hash          string
		want          bool
	}{
		{userID, messageID, hash, true},
		{userID + 1000, messageID, hash, false},
		{userID, messageID + 1000, hash, false},
		{userID, messageID, strings.Repeat("b", 64), false},
	} {
		got, err := restarted.MessageAllowed(ctx, check.user, check.message, check.hash)
		if err != nil || got != check.want {
			t.Fatalf("permission: %v %v, want %v", got, err, check.want)
		}
	}
	media.mu.Lock()
	clear(media.blobs)
	media.mu.Unlock()
	if _, err := restarted.Get(ctx, messageID, "source", "https://example.com/image", nil); err == nil {
		t.Fatal("missing storage silently accepted")
	}
	restarted.fetch = func(context.Context, string, func() error) ([]byte, error) { return nil, errors.New("fetch failed") }
	if _, err := restarted.Get(ctx, messageID, "failed", "https://example.com/failed", nil); err == nil {
		t.Fatal("fetch failure accepted")
	}
	var count int
	if err := db.Get(&count, "SELECT count(*) FROM media WHERE model_type = 'resource_images'"); err != nil || count != 1 {
		t.Fatalf("media count: %d, %v", count, err)
	}
	db.MustExec("DELETE FROM conversation_messages WHERE id = $1", messageID)
	if got, err := restarted.MessageAllowed(ctx, userID, messageID, hash); err != nil || got {
		t.Fatalf("permission survived deletion: %v %v", got, err)
	}
}

func TestPersistentAvatarUsesOnlyConfiguredSource(t *testing.T) {
	db := testutil.NewDB(t, "resource_avatar_store")
	db.MustExec("DROP INDEX index_media_resource_avatar_source")
	for range 2 {
		if err := migrations.V2_9_3(db, nil, nil); err != nil {
			t.Fatal(err)
		}
	}
	source := "https://avatars.example/person"
	var userID int
	if err := db.Get(&userID, "INSERT INTO users (type, email, first_name, avatar_url) VALUES ('contact', 'avatar@example.com', 'Avatar', $1) RETURNING id", source); err != nil {
		t.Fatal(err)
	}
	media := &memoryMedia{blobs: make(map[string][]byte)}
	store := NewStore(db, media, "fs", defaultCachePolicy)
	calls := 0
	store.fetch = func(context.Context, string, func() error) ([]byte, error) { calls++; return []byte("avatar"), nil }
	ctx := context.Background()
	if id, err := store.AvatarOwner(ctx, source); err != nil || id != userID {
		t.Fatalf("avatar owner: %d %v", id, err)
	}
	if _, err := store.AvatarOwner(ctx, "https://unconfigured.example/image"); err == nil {
		t.Fatal("unconfigured URL accepted")
	}
	if _, err := store.GetAvatar(ctx, userID, "first", source, nil); err != nil {
		t.Fatal(err)
	}
	restarted := NewStore(db, media, "fs", defaultCachePolicy)
	restarted.fetch = store.fetch
	if _, err := restarted.GetAvatar(ctx, userID, "first", source, nil); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("fetched avatar %d times", calls)
	}
	db.SetMaxOpenConns(1)
	for _, sourceID := range []string{"first", "uncached"} {
		checks := 0
		_, err := store.GetAvatar(ctx, userID, sourceID, source, func(tx *sqlx.Tx) error {
			checks++
			var id int
			if err := tx.Get(&id, "SELECT id FROM users WHERE id = $1", userID); err != nil {
				t.Fatal(err)
			}
			return ErrImage
		})
		if !errors.Is(err, ErrImage) || checks != 1 || calls != 1 {
			t.Fatalf("denied %s image was served or fetched: checks=%d calls=%d err=%v", sourceID, checks, calls, err)
		}
	}
	db.MustExec("UPDATE users SET avatar_url = NULL WHERE id = $1", userID)
	if _, err := restarted.GetAvatar(ctx, userID, "first", source, nil); err == nil {
		t.Fatal("removed avatar remained accessible")
	}
	if calls != 1 {
		t.Fatal("removed avatar triggered a fetch")
	}
}

// ioCheckedMedia runs a database operation inside each storage operation, so a
// one-connection pool detects accidentally holding a transaction across I/O.
type ioCheckedMedia struct {
	memoryMedia
	check func() error
}

func (m *ioCheckedMedia) Upload(name, kind string, body io.ReadSeeker) (string, string, error) {
	if err := m.check(); err != nil {
		return "", "", err
	}
	return m.memoryMedia.Upload(name, kind, body)
}

func (m *ioCheckedMedia) GetBlob(name string) ([]byte, error) {
	if err := m.check(); err != nil {
		return nil, err
	}
	return m.memoryMedia.GetBlob(name)
}

func TestImageIOReleasesDatabaseConnection(t *testing.T) {
	db := testutil.NewDB(t, "resource_image_io")
	db.SetMaxOpenConns(1)
	const source = "https://avatars.example/person"
	var userID int
	if err := db.Get(&userID, "INSERT INTO users (type, email, first_name, avatar_url) VALUES ('contact', 'io@example.com', 'IO', $1) RETURNING id", source); err != nil {
		t.Fatal(err)
	}
	check := func() error {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		var n int
		return db.GetContext(ctx, &n, "SELECT 1")
	}
	media := &ioCheckedMedia{memoryMedia: memoryMedia{blobs: make(map[string][]byte)}, check: check}
	store := NewStore(db, media, "fs", defaultCachePolicy)
	store.fetch = func(_ context.Context, _ string, authorize func() error) ([]byte, error) {
		if err := check(); err != nil {
			return nil, err
		}
		if err := authorize(); err != nil {
			return nil, err
		}
		return []byte("avatar"), nil
	}
	for range 2 {
		if _, err := store.GetAvatar(context.Background(), userID, "source", source, nil); err != nil {
			t.Fatal(err)
		}
	}
	// Revocation during a cached read must still prevent delivery.
	media.check = func() error {
		_, err := db.Exec("UPDATE users SET avatar_url = NULL WHERE id = $1", userID)
		return err
	}
	if _, err := store.GetAvatar(context.Background(), userID, "source", source, nil); err == nil {
		t.Fatal("served revoked avatar after storage read")
	}
	// Changing the owner while uploading must prevent publication and remove the blob.
	db.MustExec("UPDATE users SET avatar_url = $1 WHERE id = $2", source, userID)
	if _, err := store.GetAvatar(context.Background(), userID, "other", source, nil); err == nil {
		t.Fatal("published revoked avatar after upload")
	}
	var count int
	if err := db.Get(&count, "SELECT count(*) FROM media WHERE model_type = 'resource_avatars'"); err != nil || count != 1 {
		t.Fatalf("cache count: %d, %v", count, err)
	}
	if len(media.blobs) != 1 {
		t.Fatalf("failed upload leaked a blob: %d", len(media.blobs))
	}
}

func TestConcurrentStoresPublishOneImage(t *testing.T) {
	db := testutil.NewDB(t, "resource_image_race")
	const source = "https://avatars.example/person"
	var userID int
	if err := db.Get(&userID, "INSERT INTO users (type, email, first_name, avatar_url) VALUES ('contact', 'race@example.com', 'Race', $1) RETURNING id", source); err != nil {
		t.Fatal(err)
	}
	media := &memoryMedia{blobs: make(map[string][]byte)}
	started := make(chan struct{}, 2)
	proceed := make(chan struct{})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	results := make(chan error, 2)
	for range 2 {
		store := NewStore(db, media, "fs", defaultCachePolicy)
		store.fetch = func(ctx context.Context, _ string, authorize func() error) ([]byte, error) {
			started <- struct{}{}
			select {
			case <-proceed:
			case <-ctx.Done():
				return nil, ctx.Err()
			}
			if err := authorize(); err != nil {
				return nil, err
			}
			return []byte("avatar"), nil
		}
		go func() { _, err := store.GetAvatar(ctx, userID, "source", source, nil); results <- err }()
	}
	for range 2 {
		select {
		case <-started:
		case <-ctx.Done():
			t.Fatal("concurrent fetch did not start")
		}
	}
	close(proceed)
	for range 2 {
		if err := <-results; err != nil {
			t.Fatal(err)
		}
	}
	var count int
	if err := db.Get(&count, "SELECT count(*) FROM media WHERE model_type = 'resource_avatars'"); err != nil || count != 1 {
		t.Fatalf("cache count: %d, %v", count, err)
	}
	if len(media.blobs) != 1 {
		t.Fatalf("duplicate cache fill leaked blobs: %d", len(media.blobs))
	}
}

func TestStorageAdmissionDoesNotBorrowConnections(t *testing.T) {
	store := NewStore(nil, nil, "fs", defaultCachePolicy)
	releases := make([]func(), 0, cap(store.slots))
	for i := range cap(store.slots) {
		release, err := store.acquire(context.Background(), string(rune('a'+i)))
		if err != nil {
			t.Fatal(err)
		}
		releases = append(releases, release)
	}
	defer func() {
		for _, release := range releases {
			release()
		}
	}()
	for _, key := range []string{"a", "new"} {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		_, err := store.acquire(ctx, key)
		cancel()
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("waiting for %s: %v", key, err)
		}
	}
	if len(store.active) != cap(store.slots) {
		t.Fatal("canceled waiter left an active key")
	}
}

func defaultCachePolicy(*sqlx.Tx) (resourcepolicy.Config, error) {
	return resourcepolicy.Default(), nil
}
