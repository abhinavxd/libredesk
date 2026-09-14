package ws

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/abhinavxd/libredesk/internal/ws/models"
	"github.com/zerodha/logf"
)

// fakeBackplane records published payloads and lets a test feed inbound ones.
type fakeBackplane struct {
	in        chan []byte
	mu        sync.Mutex
	published [][]byte
}

func (f *fakeBackplane) Publish(_ context.Context, payload []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.published = append(f.published, append([]byte(nil), payload...))
	return nil
}

func (f *fakeBackplane) Subscribe(_ context.Context) (<-chan []byte, error) {
	return f.in, nil
}

func newTestClient(h *Hub, id int) *Client {
	return &Client{ID: id, Hub: h, Send: make(chan models.WSMessage, 8)}
}

func received(c *Client) bool {
	select {
	case <-c.Send:
		return true
	case <-time.After(100 * time.Millisecond):
		return false
	}
}

// A client subscribed to several conversations gets a conversation broadcast once, not per-UUID.
func TestBroadcastToConversationsLocalDedup(t *testing.T) {
	lo := logf.New(logf.Opts{})
	h := NewHub(&lo, nil)
	c := newTestClient(h, 1)
	h.AddClient(c)
	h.SubscribeListReplace(c, []string{"a", "b", "c"})

	h.broadcastToConversationsLocal([]string{"a", "b", "c"}, []byte(`{"x":1}`))

	if !received(c) {
		t.Fatal("subscriber should receive the broadcast")
	}
	select {
	case <-c.Send:
		t.Fatal("subscriber received the broadcast more than once")
	default:
	}
}

// Envelopes from other instances are delivered locally; the hub's own are skipped.
func TestConsumeBackplaneDeliversRemoteSkipsOwn(t *testing.T) {
	lo := logf.New(logf.Opts{})
	h := NewHub(&lo, nil)
	fb := &fakeBackplane{in: make(chan []byte, 4)}
	h.SetBackplane(fb)

	c := newTestClient(h, 1)
	h.AddClient(c)
	h.SubscribeListReplace(c, []string{"conv-1"})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go h.ConsumeBackplane(ctx)

	feed := func(origin string) {
		env := envelope{Origin: origin, Kind: envelopeConvs, Convs: []string{"conv-1"}, Data: []byte(`{"x":1}`)}
		payload, err := json.Marshal(env)
		if err != nil {
			t.Fatal(err)
		}
		fb.in <- payload
	}

	// Remote origin -> delivered.
	feed("some-other-instance")
	if !received(c) {
		t.Fatal("expected delivery of remote envelope")
	}

	// Own origin -> skipped (already delivered locally by the publisher).
	feed(h.instanceID)
	if received(c) {
		t.Fatal("hub must not redeliver its own envelope")
	}
}

// The public fan-out methods also publish to the backplane.
func TestBroadcastToConversationsPublishes(t *testing.T) {
	lo := logf.New(logf.Opts{})
	h := NewHub(&lo, nil)
	fb := &fakeBackplane{in: make(chan []byte, 4)}
	h.SetBackplane(fb)

	h.BroadcastToConversations([]string{"conv-1"}, []byte(`{"x":1}`))

	fb.mu.Lock()
	defer fb.mu.Unlock()
	if len(fb.published) != 1 {
		t.Fatalf("expected 1 published envelope, got %d", len(fb.published))
	}
	var env envelope
	if err := json.Unmarshal(fb.published[0], &env); err != nil {
		t.Fatal(err)
	}
	if env.Origin != h.instanceID || env.Kind != envelopeConvs {
		t.Fatalf("unexpected envelope: %+v", env)
	}
}
