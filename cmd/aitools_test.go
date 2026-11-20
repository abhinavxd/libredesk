package main

import (
	"strings"
	"testing"
	"time"

	cmodels "github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/volatiletech/null/v9"
)

func TestRenderAIConversationsUsesWorkspaceTimezone(t *testing.T) {
	loc, err := time.LoadLocation("Asia/Kolkata")
	if err != nil {
		t.Fatal(err)
	}
	at := time.Date(2026, time.November, 9, 20, 30, 0, 0, time.UTC)
	rows := []cmodels.AIConversationSummary{{
		ReferenceNumber: "123", CreatedAt: at, LastMessageAt: null.TimeFrom(at.Add(time.Hour)),
	}}
	got := renderAIConversations(rows, loc)
	for _, want := range []string{"Created: 2026-11-10T02:00:00+05:30", "Last message: 2026-11-10T03:00:00+05:30"} {
		if !strings.Contains(got, want) {
			t.Errorf("tool result missing %q: %q", want, got)
		}
	}
	rows[0].LastMessageAt = null.Time{}
	if got := renderAIConversations(rows, loc); strings.Contains(got, "Last message:") {
		t.Fatalf("tool result invented a last message time: %q", got)
	}
}
