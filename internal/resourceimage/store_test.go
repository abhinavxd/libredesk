package resourceimage

import (
	"context"
	"errors"
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/abhinavxd/libredesk/internal/migrations"
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
	first := NewStore(db, media, "fs")
	second := NewStore(db, media, "fs")
	var calls atomic.Int32
	fetch := func(context.Context, string, func() error) ([]byte, error) {
		calls.Add(1)
		return []byte("saved PNG"), nil
	}
	first.fetch, second.fetch = fetch, fetch
	ctx := context.Background()
	results := make(chan error, 8)
	for i := range 8 {
		go func() {
			store := first
			if i%2 == 0 {
				store = second
			}
			body, err := store.Get(ctx, messageID, "source", "https://example.com/image", nil)
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
	restarted := NewStore(db, media, "fs")
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
	store := NewStore(db, media, "fs")
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
	restarted := NewStore(db, media, "fs")
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
