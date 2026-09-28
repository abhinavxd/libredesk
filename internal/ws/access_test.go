package ws

import (
	"testing"

	"github.com/zerodha/logf"
)

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
