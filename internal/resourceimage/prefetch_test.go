package resourceimage

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/abhinavxd/libredesk/internal/resourcepolicy"
	"github.com/abhinavxd/libredesk/internal/setting"
	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/jmoiron/sqlx"
	"github.com/zerodha/logf"
)

func TestPrefetchOnReceipt(t *testing.T) {
	db := testutil.NewDB(t, "receipt_images")
	var userID, inboxID, conversationID int
	if err := db.Get(&userID, "INSERT INTO users (type, email, first_name) VALUES ('contact', 'receipt@example.com', 'Receipt') RETURNING id"); err != nil {
		t.Fatal(err)
	}
	if err := db.Get(&inboxID, "INSERT INTO inboxes (name, channel) VALUES ('Receipt', 'email') RETURNING id"); err != nil {
		t.Fatal(err)
	}
	if err := db.Get(&conversationID, "INSERT INTO conversations (contact_id, inbox_id, status_id) VALUES ($1, $2, (SELECT id FROM conversation_statuses LIMIT 1)) RETURNING id", userID, inboxID); err != nil {
		t.Fatal(err)
	}
	newMessage := func() int {
		t.Helper()
		var id int
		if err := db.Get(&id, "INSERT INTO conversation_messages (conversation_id, sender_id, sender_type, type, status, content) VALUES ($1, $2, 'contact', 'incoming', 'received', 'receipt') RETURNING id", conversationID, userID); err != nil {
			t.Fatal(err)
		}
		return id
	}
	media := &memoryMedia{blobs: make(map[string][]byte)}
	store := NewStore(db, media, "fs")
	var calls atomic.Int32
	store.fetch = func(context.Context, string, func() error) ([]byte, error) {
		calls.Add(1)
		return []byte("saved PNG"), nil
	}
	content := `<img src="https://images.example/a"><img src="https://images.example/a"><img src="https://other.example/b"><img src="https://127.0.0.1/private"><iframe src="https://images.example/frame"></iframe>`
	ctx := context.Background()
	for _, cfg := range []resourcepolicy.Config{resourcepolicy.Blocked(), {Mode: resourcepolicy.Allowlist, AllowedDomains: []string{"images.example"}}, {Mode: "invalid"}} {
		if err := store.Prefetch(ctx, newMessage(), content, func(*sqlx.Tx) (resourcepolicy.Config, error) { return cfg, nil }); err != nil {
			t.Fatal(err)
		}
	}
	if calls.Load() != 0 {
		t.Fatal("prefetched without load_on_receipt")
	}
	lo := logf.New(logf.Opts{})
	settings, err := setting.New(setting.Opts{DB: db, Lo: &lo})
	if err != nil {
		t.Fatal(err)
	}
	readPolicy := settings.GetResourcePolicyTx
	db.SetMaxOpenConns(1)
	messageID := newMessage()
	for range 2 {
		if err := store.Prefetch(ctx, messageID, content, readPolicy); err != nil {
			t.Fatal(err)
		}
	}
	if calls.Load() != 2 {
		t.Fatalf("expected two unique images fetched once, got %d", calls.Load())
	}
	restarted := NewStore(db, media, "fs")
	restarted.fetch = func(context.Context, string, func() error) ([]byte, error) {
		t.Error("refetched a persisted image")
		return nil, ErrImage
	}
	if err := restarted.Prefetch(ctx, messageID, content, readPolicy); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := db.Get(&count, "SELECT count(*) FROM media WHERE model_type = 'resource_images' AND model_id = $1", messageID); err != nil || count != 2 {
		t.Fatalf("images not persisted: %d, %v", count, err)
	}

	if err := store.Prefetch(ctx, newMessage(), content, func(*sqlx.Tx) (resourcepolicy.Config, error) {
		return resourcepolicy.Default(), errors.New("unavailable policy")
	}); err == nil {
		t.Fatal("ignored policy read failure")
	}
	if calls.Load() != 2 {
		t.Fatal("fetched while policy was unavailable")
	}

	failedMessage := newMessage()
	store.fetch = func(_ context.Context, source string, _ func() error) ([]byte, error) {
		if source == "https://images.example/a" {
			return nil, ErrImage
		}
		return []byte("saved PNG"), nil
	}
	if err := store.Prefetch(ctx, failedMessage, content, readPolicy); err == nil {
		t.Fatal("expected failed image report")
	}
	if err := db.Get(&count, "SELECT count(*) FROM media WHERE model_type = 'resource_images' AND model_id = $1", failedMessage); err != nil || count != 1 {
		t.Fatalf("failure prevented other images from persisting: %d, %v", count, err)
	}

	var blocked atomic.Bool
	interrupted := newMessage()
	store.fetch = func(context.Context, string, func() error) ([]byte, error) {
		blocked.Store(true)
		return []byte("saved PNG"), nil
	}
	if err := store.Prefetch(ctx, interrupted, `<img src="https://images.example/a">`, func(*sqlx.Tx) (resourcepolicy.Config, error) {
		if blocked.Load() {
			return resourcepolicy.Blocked(), nil
		}
		return resourcepolicy.Default(), nil
	}); err == nil {
		t.Fatal("mode change during fetch was ignored")
	}
	if err := db.Get(&count, "SELECT count(*) FROM media WHERE model_type = 'resource_images' AND model_id = $1", interrupted); err != nil || count != 0 {
		t.Fatalf("image persisted after blocking: %d, %v", count, err)
	}
}
