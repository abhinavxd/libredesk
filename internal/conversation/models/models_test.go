package models

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/volatiletech/null/v9"
)

func TestTranscript(t *testing.T) {
	msgs := []Message{
		{
			SenderType:  SenderTypeContact,
			ContentType: ContentTypeHTML,
			Content:     `<p>My payment on <a href="https://example.com/pay">this page</a> failed.</p>`,
			TextContent: "My payment on this page failed.",
		},
		{
			SenderType:  SenderTypeAgent,
			ContentType: ContentTypeText,
			TextContent: "Looking into it.",
		},
		{
			SenderType:  SenderTypeContact,
			ContentType: ContentTypeHTML,
			Content:     "",
			TextContent: "Any update?",
		},
		{
			SenderType:  SenderTypeAgent,
			ContentType: ContentTypeHTML,
			Content:     "<p></p>",
			TextContent: "",
		},
	}

	got := Transcript(msgs, 50)
	want := "Customer: My payment on [this page](https://example.com/pay) failed.\n" +
		"Agent: Looking into it.\n" +
		"Customer: Any update?\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestTranscriptMaxMessages(t *testing.T) {
	msgs := []Message{
		{SenderType: SenderTypeContact, ContentType: ContentTypeText, TextContent: "first"},
		{SenderType: SenderTypeAgent, ContentType: ContentTypeText, TextContent: "second"},
		{SenderType: SenderTypeContact, ContentType: ContentTypeText, TextContent: "third"},
	}
	got := Transcript(msgs, 2)
	if strings.Contains(got, "first") {
		t.Errorf("expected first message dropped, got %q", got)
	}
	if !strings.Contains(got, "second") || !strings.Contains(got, "third") {
		t.Errorf("expected last two messages kept, got %q", got)
	}
}

func TestShouldEvaluateAutomation(t *testing.T) {
	const systemUserID = 99

	tests := []struct {
		name     string
		senderID int
		meta     string
		want     bool
	}{
		{"agent reply", 7, `{}`, true},
		{"agent reply, empty meta", 7, ``, true},
		{"system sender (automation reply, continuity)", systemUserID, `{}`, false},
		{"agent sender but automated (CSAT on agent resolve)", 7, `{"is_automated":true}`, false},
		{"system sender and automated", systemUserID, `{"is_automated":true}`, false},
		{"is_automated explicitly false", 7, `{"is_automated":false}`, true},
		{"malformed meta treated as not automated", 7, `not-json`, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := Message{SenderID: tt.senderID, Meta: []byte(tt.meta)}
			if got := m.ShouldEvaluateAutomation(systemUserID); got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConversationSeenByJSON(t *testing.T) {
	seenAt := time.Date(2026, 9, 10, 14, 30, 0, 0, time.UTC)
	conv := Conversation{
		UUID: "test-conv-uuid",
		SeenBy: []ConversationSeenBy{
			{
				UserID:     2,
				FirstName:  "Jane",
				LastName:   "Doe",
				AvatarURL:  null.StringFrom("https://example.com/avatar.png"),
				LastSeenAt: seenAt,
			},
		},
	}

	b, err := json.Marshal(conv)
	if err != nil {
		t.Fatalf("failed to marshal conversation: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(b, &parsed); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	seenByRaw, ok := parsed["seen_by"].([]any)
	if !ok || len(seenByRaw) != 1 {
		t.Fatalf("expected seen_by array with 1 item, got %v", parsed["seen_by"])
	}

	item := seenByRaw[0].(map[string]any)
	if item["user_id"] != float64(2) {
		t.Errorf("expected user_id 2, got %v", item["user_id"])
	}
	if item["first_name"] != "Jane" {
		t.Errorf("expected first_name Jane, got %v", item["first_name"])
	}
	if item["last_name"] != "Doe" {
		t.Errorf("expected last_name Doe, got %v", item["last_name"])
	}
	if item["avatar_url"] != "https://example.com/avatar.png" {
		t.Errorf("expected avatar_url https://example.com/avatar.png, got %v", item["avatar_url"])
	}
	if item["last_seen_at"] != "2026-09-10T14:30:00Z" {
		t.Errorf("expected last_seen_at 2026-09-10T14:30:00Z, got %v", item["last_seen_at"])
	}
}
