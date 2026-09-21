package resourceimage

import (
	"context"
	"errors"
	"io"
	"sync/atomic"
	"testing"
	"time"

	"github.com/abhinavxd/libredesk/internal/migrations"
	"github.com/abhinavxd/libredesk/internal/resourcepolicy"
	"github.com/abhinavxd/libredesk/internal/setting"
	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

const cacheTestSource = "https://avatars.example/person"

func cacheFixture(t *testing.T, database string, limit int64) (*Store, *memoryMedia, *setting.Manager, int) {
	t.Helper()
	db := testutil.NewDB(t, database)
	settings, err := setting.New(setting.Opts{DB: db})
	if err != nil {
		t.Fatal(err)
	}
	cfg := resourcepolicy.Default()
	cfg.MaxCacheBytes = limit
	if err := settings.SetResourcePolicy(cfg); err != nil {
		t.Fatal(err)
	}
	var userID int
	if err := db.Get(&userID, `INSERT INTO users (type, email, first_name, avatar_url)
 VALUES ('contact', 'cache@example.com', 'Cache', $1) RETURNING id`, cacheTestSource); err != nil {
		t.Fatal(err)
	}
	media := &memoryMedia{blobs: make(map[string][]byte)}
	store := NewStore(db, media, "fs", settings.GetResourcePolicyTx)
	store.fetch = func(_ context.Context, _ string, authorize func() error) ([]byte, error) {
		if err := authorize(); err != nil {
			return nil, err
		}
		return []byte("data"), nil
	}
	return store, media, settings, userID
}

func cachedName(t *testing.T, store *Store, sourceID string) string {
	t.Helper()
	var name string
	if err := store.db.Get(&name, "SELECT uuid FROM media WHERE content_id = $1", sourceID); err != nil {
		t.Fatal(err)
	}
	return name
}

func TestCacheBudgetFIFOAndOnDemandRefetch(t *testing.T) {
	store, media, _, userID := cacheFixture(t, "cache_fifo", 8)
	// The migration is safe on an already upgraded installation.
	for range 2 {
		if err := migrations.V2_9_4(store.db, nil, nil); err != nil {
			t.Fatal(err)
		}
	}
	ctx := context.Background()
	var fetches atomic.Int32
	store.fetch = func(_ context.Context, _ string, authorize func() error) ([]byte, error) {
		fetches.Add(1)
		return []byte("data"), authorize()
	}
	get := func(id string) {
		t.Helper()
		if _, err := store.GetAvatar(ctx, userID, id, cacheTestSource, nil); err != nil {
			t.Fatal(err)
		}
	}
	get("a")
	a := cachedName(t, store, "a")
	store.db.MustExec("UPDATE media SET created_at = now() - interval '2 hours' WHERE uuid = $1", a)
	get("b")
	b := cachedName(t, store, "b")
	store.db.MustExec("UPDATE media SET created_at = now() - interval '1 hour' WHERE uuid = $1", b)
	get("a") // Viewing A must not promote it above B.
	get("c")
	if _, ok := media.blobs[a]; ok {
		t.Fatal("recently viewed oldest image survived FIFO eviction")
	}
	if _, ok := media.blobs[b]; !ok {
		t.Fatal("newer image was evicted first")
	}
	if fetches.Load() != 3 || len(media.blobs) != 2 {
		t.Fatalf("unexpected cache: fetches=%d blobs=%d", fetches.Load(), len(media.blobs))
	}
	// A denied request cannot refetch an evicted entry.
	if _, err := store.GetAvatar(ctx, userID, "a", cacheTestSource, func(*sqlx.Tx) error { return ErrImage }); !errors.Is(err, ErrImage) {
		t.Fatalf("expected permission failure, got %v", err)
	}
	if fetches.Load() != 3 {
		t.Fatal("refetched without permission")
	}
	get("a")
	if fetches.Load() != 4 || len(media.blobs) != 2 {
		t.Fatal("evicted image was not refetched on demand within budget")
	}
}

func TestCacheBudgetSharesMessageAndAvatarStorage(t *testing.T) {
	store, media, _, userID := cacheFixture(t, "cache_shared", 8)
	// An existing message image counts toward the same budget as an avatar.
	image, attachment := uuid.NewString(), uuid.NewString()
	store.db.MustExec(`INSERT INTO media (uuid, store, filename, content_type, size, model_type, created_at)
 VALUES ($1, 'fs', 'image.png', 'image/png', 8, 'resource_images', now() - interval '1 hour'),
 ($2, 'fs', 'attachment.png', 'image/png', 1000, 'messages', now() - interval '2 hours')`, image, attachment)
	media.blobs[image], media.blobs[attachment] = []byte("12345678"), []byte("attachment")
	if _, err := store.GetAvatar(context.Background(), userID, "a", cacheTestSource, nil); err != nil {
		t.Fatal(err)
	}
	if _, ok := media.blobs[image]; ok {
		t.Fatal("message image did not count against avatar budget")
	}
	if _, ok := media.blobs[attachment]; !ok {
		t.Fatal("ordinary attachment was evicted")
	}
}

func TestCacheRejectsEmptyAndOversizedWithoutEviction(t *testing.T) {
	store, media, _, userID := cacheFixture(t, "cache_oversize", 4)
	if _, err := store.GetAvatar(context.Background(), userID, "a", cacheTestSource, nil); err != nil {
		t.Fatal(err)
	}
	name := cachedName(t, store, "a")
	for _, body := range [][]byte{nil, []byte("large")} {
		store.fetch = func(context.Context, string, func() error) ([]byte, error) { return body, nil }
		if _, err := store.GetAvatar(context.Background(), userID, "bad", cacheTestSource, nil); !errors.Is(err, ErrImage) {
			t.Fatalf("invalid image accepted: %v", err)
		}
		if _, ok := media.blobs[name]; !ok || len(media.blobs) != 1 {
			t.Fatal("invalid image evicted existing content")
		}
	}
}

func TestCacheCleanerRespondsToLoweredLimit(t *testing.T) {
	store, media, settings, userID := cacheFixture(t, "cache_lowered", 8)
	for _, id := range []string{"a", "b"} {
		if _, err := store.GetAvatar(context.Background(), userID, id, cacheTestSource, nil); err != nil {
			t.Fatal(err)
		}
	}
	a := cachedName(t, store, "a")
	store.db.MustExec("UPDATE media SET created_at = now() - interval '1 hour' WHERE uuid = $1", a)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	errs := make(chan error, 4)
	go func() {
		defer close(done)
		store.RunCacheCleaner(ctx, func(err error) { errs <- err })
	}()
	defer func() { cancel(); <-done }()
	cfg := resourcepolicy.Default()
	cfg.MaxCacheBytes = 4
	if err := settings.SetResourcePolicy(cfg); err != nil {
		t.Fatal(err)
	}
	store.RequestCleanup()
	deadline := time.After(3 * time.Second)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		media.mu.Lock()
		_, remains := media.blobs[a]
		media.mu.Unlock()
		if !remains {
			break
		}
		select {
		case err := <-errs:
			t.Fatal(err)
		case <-deadline:
			t.Fatal("lowered limit did not trigger cleanup")
		case <-ticker.C:
		}
	}
}

func TestConcurrentCacheReservationsRespectBudget(t *testing.T) {
	store, media, settings, userID := cacheFixture(t, "cache_concurrent", 8)
	checked := &budgetCheckedMedia{memoryMedia: media, limit: 8}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	results := make(chan error, 8)
	for i := range 8 {
		other := NewStore(store.db, checked, "fs", settings.GetResourcePolicyTx)
		other.fetch = store.fetch
		go func() {
			_, err := other.GetAvatar(ctx, userID, string(rune('a'+i)), cacheTestSource, nil)
			results <- err
		}()
	}
	for range 8 {
		if err := <-results; err != nil {
			t.Fatal(err)
		}
	}
	var usage cacheUsage
	if err := store.db.Get(&usage, cacheUsageSQL); err != nil || usage.Bytes > 8 {
		t.Fatalf("cache exceeded budget: %#v, %v", usage, err)
	}
	if len(media.blobs) > 2 {
		t.Fatal("stored blobs exceeded the budget")
	}
}

type budgetCheckedMedia struct {
	*memoryMedia
	limit int
}

func (m *budgetCheckedMedia) Upload(name, _ string, body io.ReadSeeker) (string, string, error) {
	data, err := io.ReadAll(body)
	if err != nil {
		return "", "", err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	used := len(data)
	for _, blob := range m.blobs {
		used += len(blob)
	}
	if used > m.limit {
		return "", "", errors.New("physical cache exceeded budget during upload")
	}
	m.blobs[name] = data
	return name, "", nil
}

func TestLoweredLimitRejectsInFlightUpload(t *testing.T) {
	store, _, settings, userID := cacheFixture(t, "cache_inflight", 8)
	store.db.SetMaxOpenConns(1)
	media := &ioCheckedMedia{
		memoryMedia: memoryMedia{blobs: make(map[string][]byte)},
		check: func() error {
			cfg := resourcepolicy.Default()
			cfg.MaxCacheBytes = 3
			return settings.SetResourcePolicy(cfg)
		},
	}
	store.media = media
	if _, err := store.GetAvatar(context.Background(), userID, "a", cacheTestSource, nil); !errors.Is(err, ErrImage) {
		t.Fatalf("published an upload exceeding the newly lowered limit: %v", err)
	}
	var usage cacheUsage
	if err := store.db.Get(&usage, cacheUsageSQL); err != nil || usage.Bytes != 0 || len(media.blobs) != 0 {
		t.Fatalf("rejected upload left reserved bytes or storage: %#v, %v", usage, err)
	}
}

type failedDeleteMedia struct {
	*memoryMedia
	fail bool
}

func (m *failedDeleteMedia) Delete(name string) error {
	if m.fail {
		return errors.New("storage offline")
	}
	return m.memoryMedia.Delete(name)
}

func TestFailedEvictionRemainsAccountedAndRetryable(t *testing.T) {
	store, media, _, userID := cacheFixture(t, "cache_delete_retry", 4)
	if _, err := store.GetAvatar(context.Background(), userID, "a", cacheTestSource, nil); err != nil {
		t.Fatal(err)
	}
	failing := &failedDeleteMedia{memoryMedia: media, fail: true}
	store.media = failing
	if _, err := store.GetAvatar(context.Background(), userID, "b", cacheTestSource, nil); err == nil {
		t.Fatal("stored image despite failed eviction")
	}
	var usage cacheUsage
	if err := store.db.Get(&usage, cacheUsageSQL); err != nil || usage.Bytes != 4 || usage.Deleting != 4 {
		t.Fatalf("failed delete freed budget: %#v, %v", usage, err)
	}
	failing.fail = false
	if err := store.Trim(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(media.blobs) != 0 {
		t.Fatal("cleanup did not retry failed delete")
	}
}

func TestInterruptedUploadReservationIsReclaimed(t *testing.T) {
	store, media, _, _ := cacheFixture(t, "cache_abandoned", 4)
	name := uuid.NewString()
	store.db.MustExec(`INSERT INTO resource_image_cache_pending (uuid, size, state, created_at)
 VALUES ($1, 4, 'upload', now() - interval '10 minutes')`, name)
	media.blobs[name] = []byte("data")
	if err := store.Trim(context.Background()); err != nil {
		t.Fatal(err)
	}
	var usage cacheUsage
	if err := store.db.Get(&usage, cacheUsageSQL); err != nil || usage.Bytes != 0 || len(media.blobs) != 0 {
		t.Fatalf("abandoned upload remains: %#v, %v", usage, err)
	}
}
