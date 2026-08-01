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
