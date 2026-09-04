package models

import (
	"strings"
	"testing"
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
