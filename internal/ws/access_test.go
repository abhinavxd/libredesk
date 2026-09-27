package ws

import (
	"errors"
	"testing"

	"github.com/zerodha/logf"
)

type subscriptionAccess struct {
	allowed map[int]bool
	err     error
}

func (s *subscriptionAccess) BroadcastTypingToWidgetClientsOnly(string, bool) {}

func (s *subscriptionAccess) FilterAuthorizedListUUIDs(id int, uuids []string) ([]string, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.allowed[id] {
		return uuids, nil
	}
	return nil, nil
}

func TestRecheckSubscribersDropsRevokedAccess(t *testing.T) {
	lo := logf.New(logf.Opts{})
	h := NewHub(&lo, nil /** userStore **/)
	store := &subscriptionAccess{allowed: map[int]bool{1: true, 2: true}}
	h.SetConversationStore(store)
	first, second := &Client{ID: 1}, &Client{ID: 2}
	h.SubscribeListReplace(first, []string{"ticket", "ticket"})
	h.SubscribeOpenConv(first, "ticket")
	h.SubscribeOpenConv(second, "ticket")
	h.RecheckSubscribers("ticket")
	if got := h.ListSubscribers("ticket"); len(got) != 2 {
		t.Fatalf("expected two unique subscribers, got %d", len(got))
	}
	store.allowed[1] = false
	h.RecheckSubscribers("ticket")
	if got := h.ListSubscribers("ticket"); len(got) != 1 || got[0] != second {
		t.Fatalf("revoked subscriber remained: %v", got)
	}
	if _, ok := h.clientListSubs[first]["ticket"]; ok || h.clientOpenSub[first] != "" {
		t.Fatal("revoked subscriptions were retained")
	}
	store.err = errors.New("access lookup failed")
	h.RecheckSubscribers("ticket")
	if got := h.ListSubscribers("ticket"); len(got) != 0 {
		t.Fatal("subscriber kept after failed access lookup")
	}
	h.RecheckSubscribers("missing")
}
