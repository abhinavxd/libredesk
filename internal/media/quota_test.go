package media

import (
	"bytes"
	"errors"
	"io"
	"os"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"

	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/volatiletech/null/v9"
	"github.com/zerodha/logf"
)

type quotaTestStore struct {
	Store
	mu            sync.Mutex
	blobs         map[string][]byte
	writes        int
	failWrite     bool
	failedDeletes int
}

func (s *quotaTestStore) Name() string { return "fs" }
func (s *quotaTestStore) Put(name, _ string, src io.ReadSeeker) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.writes++
	s.blobs[name] = []byte("partial")
	if s.failWrite {
		return "", syscall.ENOSPC
	}
	body, err := io.ReadAll(src)
	s.blobs[name] = body
	return name, err
}
func (s *quotaTestStore) Delete(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.failedDeletes > 0 {
		s.failedDeletes--
		return errors.New("storage temporarily unavailable")
	}
	if _, ok := s.blobs[name]; !ok {
		return os.ErrNotExist
	}
	delete(s.blobs, name)
	return nil
}
func quotaFixture(t *testing.T, name string, limit int64) (*Manager, *quotaTestStore) {
	t.Helper()
	db := testutil.NewDB(t, name)
	lo := logf.New(logf.Opts{})
	store := &quotaTestStore{blobs: map[string][]byte{}}
	m, err := New(Opts{DB: db, Store: store, Lo: &lo, I18n: testutil.NewI18n(t), RootURL: func() string { return "http://localhost" }, SigningKey: "test", MaxStorageBytes: limit})
	if err != nil {
		t.Fatal(err)
	}
	return m, store
}
func quotaUpload(m *Manager, size int) error {
	// Intentionally underreport the size: reservations must use the actual reader size.
	_, err := m.UploadAndInsert("file.txt", "text/plain", "", null.String{}, null.Int{}, bytes.NewReader(bytes.Repeat([]byte("x"), size)), 1, null.StringFrom("attachment"), []byte("{}"), true)
	return err
}
func TestDurableQuotaReservesBeforeConcurrentWrites(t *testing.T) {
	m, store := quotaFixture(t, "durable_quota_concurrency", 10)
	var success atomic.Int32
	var wg sync.WaitGroup
	for range 20 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := quotaUpload(m, 4)
			if err == nil {
				success.Add(1)
				return
			}
			var e envelope.Error
			if !errors.As(err, &e) || e.Code != 507 {
				t.Errorf("unexpected error: %v", err)
			}
		}()
	}
	wg.Wait()
	if success.Load() != 2 || store.writes != 2 {
		t.Fatalf("successes=%d storage writes=%d; rejected uploads must not write", success.Load(), store.writes)
	}
	usage, err := m.GetStorageUsage()
	if err != nil || usage.UsedBytes != 8 {
		t.Fatalf("usage=%+v err=%v", usage, err)
	}
}
func TestDurableDiskFullCleanupAndSweep(t *testing.T) {
	for _, retry := range []bool{false, true} {
		name := "durable_disk_full"
		if retry {
			name += "_retry"
		}
		t.Run(name, func(t *testing.T) {
			m, store := quotaFixture(t, name, 10)
			store.failWrite = true
			if retry {
				store.failedDeletes = 2
			}
			notices := 0
			m.SetStorageFullNotifier(func() { notices++ })
			err := quotaUpload(m, 4)
			var e envelope.Error
			if !errors.As(err, &e) || e.Code != 507 || notices != 1 {
				t.Fatalf("error=%v notices=%d", err, notices)
			}
			if retry {
				usage, _ := m.GetStorageUsage()
				if usage.UsedBytes != 4 {
					t.Fatalf("failed cleanup lost accounting: %+v", usage)
				}
				m.db.MustExec("UPDATE media SET created_at=now()-interval '8 days'")
				m.deleteUnlinked()
			}
			usage, _ := m.GetStorageUsage()
			if usage.UsedBytes != 0 || len(store.blobs) != 0 {
				t.Fatalf("cleanup left usage=%d blobs=%d", usage.UsedBytes, len(store.blobs))
			}
		})
	}
}
