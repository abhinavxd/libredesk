package models

import (
	"strings"
	"testing"
	"time"

	"github.com/abhinavxd/libredesk/internal/attachment"
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

	got := Transcript(msgs, 50, time.UTC)
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
	got := Transcript(msgs, 2, time.UTC)
	if strings.Contains(got, "first") {
		t.Errorf("expected first message dropped, got %q", got)
	}
	if !strings.Contains(got, "second") || !strings.Contains(got, "third") {
		t.Errorf("expected last two messages kept, got %q", got)
	}
}

func TestTranscriptMessageContext(t *testing.T) {
	stamp := time.Date(2026, time.November, 2, 0, 15, 0, 0, time.FixedZone("NPT", 5*3600+45*60))
	msgs := []Message{
		{SenderType: SenderTypeContact, CreatedAt: stamp, TextContent: "It failed yesterday."},
		{SenderType: SenderTypeAgent, CreatedAt: stamp.Add(time.Hour), TextContent: "Please attach your invoice."},
		{SenderType: SenderTypeContact, CreatedAt: stamp.Add(2 * time.Hour), Attachments: attachment.Attachments{
			{Name: "invoice.pdf", ContentType: "application/pdf"},
			{Name: "screenshot.png", ContentType: "image/png"},
		}},
		{SenderType: SenderTypeContact},
	}
	got := Transcript(msgs, 50 /** max **/, time.UTC)
	for _, part := range []string{
		"[2026-11-01T18:30:00Z] Customer: It failed yesterday.",
		"[2026-11-01T19:30:00Z] Agent: Please attach your invoice.",
		`[2026-11-01T20:30:00Z] Customer: [Attachment: "invoice.pdf" (application/pdf). Contents are unavailable in this transcript.]`,
		`[Attachment: "screenshot.png" (image/png). Contents are unavailable in this transcript.]`,
		"Customer: [The customer sent a message with no readable text.]",
	} {
		if !strings.Contains(got, part) {
			t.Errorf("missing %q in transcript: %s", part, got)
		}
	}
	if strings.Contains(got, "0001-") || strings.Contains(got, "omitted") {
		t.Fatalf("invented timestamp or missing history: %s", got)
	}
}

func TestTranscriptReportsOmittedHistory(t *testing.T) {
	for _, tc := range []struct {
		name      string
		msgs      []Message
		max       int
		truncated bool
	}{
		{name: "database limit", msgs: []Message{{SenderType: SenderTypeContact, Total: 51, TextContent: "Latest message"}}, max: 50, truncated: true},
		{name: "renderer limit", msgs: []Message{{TextContent: "Old message"}, {TextContent: "Latest message"}}, max: 1, truncated: true},
		{name: "complete history", msgs: []Message{{Total: 1, TextContent: "Latest message"}}, max: 1},
		{name: "empty", max: 50},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := Transcript(tc.msgs, tc.max, time.UTC)
			if strings.Contains(got, "Earlier messages are omitted") != tc.truncated {
				t.Fatalf("incorrect history notice: %q", got)
			}
			if len(tc.msgs) > 0 && !strings.Contains(got, "Latest message") || strings.Contains(got, "Old message") {
				t.Fatalf("incorrect messages included: %q", got)
			}
		})
	}
}

func TestTranscriptUsesWorkspaceTimezone(t *testing.T) {
	for _, tc := range []struct {
		zone string
		at   string
		want string
	}{
		{zone: "Asia/Kolkata", at: "2026-11-09T20:30:00Z", want: "2026-11-10T02:00:00+05:30"},
		{zone: "America/New_York", at: "2026-11-01T05:30:00Z", want: "2026-11-01T01:30:00-04:00"},
		{zone: "America/New_York", at: "2026-11-01T06:30:00Z", want: "2026-11-01T01:30:00-05:00"},
	} {
		t.Run(tc.zone+tc.at, func(t *testing.T) {
			loc, err := time.LoadLocation(tc.zone)
			if err != nil {
				t.Fatal(err)
			}
			at, err := time.Parse(time.RFC3339, tc.at)
			if err != nil {
				t.Fatal(err)
			}
			msgs := []Message{{SenderType: SenderTypeContact, CreatedAt: at, TextContent: "Yesterday's payment failed."}}
			got := Transcript(msgs, 50 /** max **/, loc)
			if want := "[" + tc.want + "] Customer: Yesterday's payment failed.\n"; got != want {
				t.Errorf("transcript = %q, want %q", got, want)
			}
			if msgs[0].CreatedAt != at {
				t.Fatal("context rendering changed the source timestamp")
			}
		})
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
