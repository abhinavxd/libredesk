package ws

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/abhinavxd/libredesk/internal/ws/models"
	"github.com/zerodha/logf"
)

type fakeUserStore struct {
	offline chan int
}

func (f *fakeUserStore) UpdateLastActive(int) (bool, error) { return false, nil }

func (f *fakeUserStore) MarkAgentOffline(userID int) bool {
	f.offline <- userID
	return true
}

func TestRemoveClient(t *testing.T) {
	newHubWith := func(n int) (*Hub, []*Client) {
		lo := logf.New(logf.Opts{})
		h := NewHub(&lo, nil)
		clients := make([]*Client, n)
		for i := range clients {
			clients[i] = &Client{ID: 7, Hub: h}
			h.AddClient(clients[i])
		}
		return h, clients
	}

	for n := 1; n <= 4; n++ {
		for remove := 0; remove < n; remove++ {
			h, clients := newHubWith(n)
			h.RemoveClient(clients[remove])

			got := h.clients[7]
			if len(got) != n-1 {
				t.Fatalf("n=%d remove=%d: want len %d, got %d", n, remove, n-1, len(got))
			}
			if n-1 == 0 {
				if _, ok := h.clients[7]; ok {
					t.Fatalf("n=%d remove=%d: expected map entry deleted", n, remove)
				}
				continue
			}
			// Order preserved and the removed client is gone.
			want := append(append([]*Client{}, clients[:remove]...), clients[remove+1:]...)
			for i := range want {
				if got[i] != want[i] {
					t.Fatalf("n=%d remove=%d: order changed at %d", n, remove, i)
				}
			}
			// The slot past the new length must not still reference the removed client.
			full := got[:len(got)+1]
			if full[len(got)] != nil {
				t.Fatalf("n=%d remove=%d: stale pointer left in hidden slot", n, remove)
			}
		}
	}
}

func TestRemoveClientUnknown(t *testing.T) {
	lo := logf.New(logf.Opts{})
	h := NewHub(&lo, nil)
	a := &Client{ID: 1, Hub: h}
	h.AddClient(a)
	h.RemoveClient(&Client{ID: 1, Hub: h})
	if len(h.clients[1]) != 1 || h.clients[1][0] != a {
		t.Fatalf("removing an unregistered client must not change the slice: %v", h.clients[1])
	}
}

func TestRetainSubscribersDropsRevokedAccess(t *testing.T) {
	lo := logf.New(logf.Opts{})
	h := NewHub(&lo, nil /** userStore **/)
	first, second := &Client{ID: 1}, &Client{ID: 2}
	h.SubscribeListReplace(first, []string{"ticket", "ticket"})
	h.SubscribeOpenConv(first, "ticket")
	h.SubscribeOpenConv(second, "ticket")
	h.RetainSubscribers("ticket", []int{1, 2})
	if got := h.ListSubscribers("ticket"); len(got) != 2 {
		t.Fatalf("expected two unique subscribers, got %d", len(got))
	}
	h.RetainSubscribers("ticket", []int{2})
	if got := h.ListSubscribers("ticket"); len(got) != 1 || got[0] != second {
		t.Fatalf("revoked subscriber remained: %v", got)
	}
	if _, ok := h.clientListSubs[first]["ticket"]; ok || h.clientOpenSub[first] != "" {
		t.Fatal("revoked subscriptions were retained")
	}
	h.RetainSubscribers("missing", nil /** allowedAgentIDs **/)
}

func TestClientBufferDropIsVisible(t *testing.T) {
	var logs bytes.Buffer
	lo := logf.New(logf.Opts{Writer: &logs})
	client := &Client{ID: 7, Hub: NewHub(&lo, nil /** userStore **/), Send: make(chan models.WSMessage, 1)}
	client.SendMessage([]byte("first"), 1 /** typ **/)
	if logs.Len() != 0 {
		t.Fatal("successful enqueue produced a warning")
	}
	client.SendMessage([]byte("second"), 1 /** typ **/)
	if !strings.Contains(logs.String(), "level=warn") || !strings.Contains(logs.String(), "client_id=7") {
		t.Fatalf("missing drop warning: %s", logs.String())
	}
	logs.Reset()
	client.close()
	client.SendMessage([]byte("closed"), 1 /** typ **/)
	if logs.Len() != 0 {
		t.Fatal("normal closed connection produced a warning")
	}
}

func TestAgentMarkedOfflineAfterLastSocketCloses(t *testing.T) {
	cases := []struct {
		name        string
		run         func(h *Hub)
		wantOffline bool
	}{
		{"last socket closes", func(h *Hub) {
			tab := &Client{ID: 7, Hub: h}
			h.AddClient(tab)
			h.RemoveClient(tab)
		}, true},
		{"page reload reconnects within grace", func(h *Hub) {
			tab := &Client{ID: 7, Hub: h}
			h.AddClient(tab)
			h.RemoveClient(tab)
			h.AddClient(&Client{ID: 7, Hub: h})
		}, false},
		{"another tab stays open", func(h *Hub) {
			first, second := &Client{ID: 7, Hub: h}, &Client{ID: 7, Hub: h}
			h.AddClient(first)
			h.AddClient(second)
			h.RemoveClient(first)
		}, false},
		{"closed socket removed twice", func(h *Hub) {
			tab := &Client{ID: 7, Hub: h}
			h.AddClient(tab)
			h.RemoveClient(tab)
			h.RemoveClient(tab)
		}, true},
		{"server shutting down", func(h *Hub) {
			tab := &Client{ID: 7, Hub: h}
			h.AddClient(tab)
			h.closing.Store(true)
			h.RemoveClient(tab)
		}, false},
		{"shutdown starts during grace", func(h *Hub) {
			tab := &Client{ID: 7, Hub: h}
			h.AddClient(tab)
			h.RemoveClient(tab)
			h.closing.Store(true)
		}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h, store := offlineTestHub()
			tc.run(h)
			select {
			case id := <-store.offline:
				if !tc.wantOffline {
					t.Fatalf("agent %d marked offline", id)
				}
				if id != 7 {
					t.Fatalf("marked agent %d offline, want 7", id)
				}
				select {
				case <-store.offline:
					t.Fatal("agent marked offline twice")
				case <-time.After(5 * h.offlineGrace):
				}
			case <-time.After(10 * h.offlineGrace):
				if tc.wantOffline {
					t.Fatal("agent was not marked offline")
				}
			}
		})
	}
}

func offlineTestHub() (*Hub, *fakeUserStore) {
	lo := logf.New(logf.Opts{})
	store := &fakeUserStore{offline: make(chan int, 4)}
	h := NewHub(&lo, store)
	h.offlineGrace = 20 * time.Millisecond
	return h, store
}
