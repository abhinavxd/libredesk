package ws

import (
	"testing"

	"github.com/zerodha/logf"
)

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
