package main

import (
	"encoding/json"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/abhinavxd/libredesk/internal/whatsapp"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"github.com/zerodha/logf"
)

func TestEnqueueAndConsume(t *testing.T) {
	app, mr := testIngesterApp(t)
	ing, err := newWhatsAppIngester(app)
	if err != nil {
		t.Fatalf("newWhatsAppIngester: %v", err)
	}

	body := []byte(`{"object":"whatsapp_business_account","entry":[]}`)
	if err := ing.Enqueue(9, body); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	if got := mr.Exists(whatsAppStream); !got {
		t.Fatal("expected the delivery to be persisted to the stream")
	}

	stream, err := mr.Stream(whatsAppStream)
	if err != nil {
		t.Fatalf("reading the stream: %v", err)
	}
	if len(stream) != 1 {
		t.Fatalf("expected one entry, got %d", len(stream))
	}
	var payload whatsAppJob
	if err := json.Unmarshal([]byte(stream[0].Values[1]), &payload); err != nil {
		t.Fatalf("unmarshal job: %v", err)
	}
	if payload.InboxID != 9 || string(payload.Body) != string(body) {
		t.Fatalf("unexpected job: %+v", payload)
	}
}

func TestHandleDropsUnusableJobs(t *testing.T) {
	app, _ := testIngesterApp(t)
	ing, err := newWhatsAppIngester(app)
	if err != nil {
		t.Fatalf("newWhatsAppIngester: %v", err)
	}
	handler := ing.handle(app)

	if err := handler(t.Context(), []byte("not json")); err != nil {
		t.Fatalf("an unparseable job must be dropped, got %v", err)
	}
	job, _ := json.Marshal(whatsAppJob{InboxID: 1, Body: json.RawMessage(`"not a payload"`)})
	if err := handler(t.Context(), job); err != nil {
		t.Fatalf("an unparseable webhook body must be dropped, got %v", err)
	}
}

func TestIngesterRunAndClose(t *testing.T) {
	app, _ := testIngesterApp(t)
	ing, err := newWhatsAppIngester(app)
	if err != nil {
		t.Fatalf("newWhatsAppIngester: %v", err)
	}

	done := make(chan struct{})
	go func() {
		ing.Run()
		close(done)
	}()
	ing.Close()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not return after Close")
	}
}

// Two deliveries from the same sender must not be ingested at once, or both create a conversation.
func TestLockSenderSerializesPerSender(t *testing.T) {
	app, _ := testIngesterApp(t)
	ing, err := newWhatsAppIngester(app)
	if err != nil {
		t.Fatalf("newWhatsAppIngester: %v", err)
	}

	var (
		inFlight atomic.Int32
		overlaps atomic.Int32
		wg       sync.WaitGroup
	)
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			unlock := ing.lockSender("919876543210")
			defer unlock()
			if inFlight.Add(1) > 1 {
				overlaps.Add(1)
			}
			time.Sleep(2 * time.Millisecond)
			inFlight.Add(-1)
		}()
	}
	wg.Wait()

	if overlaps.Load() != 0 {
		t.Fatalf("expected no overlapping work per sender, got %d", overlaps.Load())
	}
}

// A slow media download for one sender must not stall every other sender.
func TestLockSenderAllowsOtherSenders(t *testing.T) {
	app, _ := testIngesterApp(t)
	ing, err := newWhatsAppIngester(app)
	if err != nil {
		t.Fatalf("newWhatsAppIngester: %v", err)
	}

	release := ing.lockSender("919876543210")
	defer release()

	done := make(chan struct{})
	go func() {
		ing.lockSender("911111111111")()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("a second sender was blocked by the first")
	}
}

func TestLockSenderReleasesEntries(t *testing.T) {
	locks := &keyedLock{entries: make(map[string]*keyedLockEntry)}
	for _, sender := range []string{"a", "b", "c"} {
		locks.lock(sender)()
	}
	locks.mu.Lock()
	defer locks.mu.Unlock()
	if len(locks.entries) != 0 {
		t.Fatalf("expected the table to be empty, got %d entries", len(locks.entries))
	}
}

func TestBuildInboundMeta(t *testing.T) {
	app, _ := testIngesterApp(t)

	if got := buildInboundMeta(app, parsedMessage("text")); got != nil {
		t.Fatalf("expected no meta for an ordinary message, got %s", got)
	}
	got := buildInboundMeta(app, parsedMessage("unsupported"))
	var out map[string]any
	if err := json.Unmarshal(got, &out); err != nil {
		t.Fatalf("unmarshal meta: %v", err)
	}
	if out["wa_unsupported"] != true {
		t.Fatalf("unexpected meta: %s", got)
	}
}

func TestInboxIDFromPath(t *testing.T) {
	tests := []struct {
		name    string
		value   any
		want    int
		wantErr bool
	}{
		{"numeric", "15", 15, false},
		{"not a number", "abc", 0, true},
		{"empty", "", 0, true},
		{"missing", nil, 0, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := &fastglue.Request{RequestCtx: &fasthttp.RequestCtx{}}
			if tc.value != nil {
				r.RequestCtx.SetUserValue("inbox_id", tc.value)
			}
			got, err := inboxIDFromPath(r)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected an error, got %d", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("expected %d, got %d", tc.want, got)
			}
		})
	}
}

func testIngesterApp(t *testing.T) (*App, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	lo := logf.New(logf.Opts{Level: logf.FatalLevel})
	app := &App{lo: &lo, redis: redis.NewClient(&redis.Options{Addr: mr.Addr()})}
	return app, mr
}

func parsedMessage(typ string) whatsapp.ParsedMessage {
	return whatsapp.ParsedMessage{ID: "wamid.X", From: "919876543210", Type: typ}
}
