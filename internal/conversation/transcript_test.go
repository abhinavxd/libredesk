package conversation

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/abhinavxd/libredesk/internal/attachment"
	"github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/knadh/go-i18n"
	"github.com/volatiletech/null/v9"
)

func newTestI18n(t *testing.T) *i18n.I18n {
	t.Helper()
	b, err := os.ReadFile("../../i18n/en-US.json")
	if err != nil {
		t.Fatalf("reading i18n file: %v", err)
	}
	in, err := i18n.New(b)
	if err != nil {
		t.Fatalf("loading i18n: %v", err)
	}
	return in
}

func TestBuildTranscript(t *testing.T) {
	m := &Manager{i18n: newTestI18n(t)}

	created := time.Date(2026, time.May, 11, 10, 0, 0, 0, time.UTC)
	downloaded := time.Date(2026, time.June, 2, 16, 10, 0, 0, time.UTC)

	conversation := models.Conversation{
		ReferenceNumber: "1234",
		Subject:         null.StringFrom("Refund request"),
		CreatedAt:       created,
		Contact:         models.ConversationContact{FirstName: "John", LastName: "Doe"},
	}

	messages := []models.Message{
		{
			Type:        models.MessageIncoming,
			SenderType:  models.SenderTypeContact,
			CreatedAt:   created.Add(time.Minute),
			TextContent: "Hi, I need a refund for order #555",
			Author:      models.MessageAuthor{FirstName: "John", LastName: "Doe"},
		},
		{
			Type:        models.MessageOutgoing,
			SenderType:  models.SenderTypeAgent,
			CreatedAt:   created.Add(5 * time.Minute),
			TextContent: "Sure, processing it now.",
			Author:      models.MessageAuthor{FirstName: "Priya"},
			Attachments: attachment.Attachments{{Name: "invoice.pdf"}},
		},
		{
			Type:        models.MessageOutgoing,
			SenderType:  models.SenderTypeAgent,
			CreatedAt:   created.Add(6 * time.Minute),
			Content:     "<p>Falls back to <b>HTML</b></p>",
			ContentType: models.ContentTypeHTML,
			Author:      models.MessageAuthor{FirstName: "Priya"},
		},
	}

	out := string(m.BuildTranscript(conversation, messages, downloaded))

	wantContains := []string{
		"Conversation #1234",
		"Subject: Refund request",
		"Contact: John Doe",
		"Created at: May 11, 2026 10:00 AM UTC",
		"Downloaded at: Jun 02, 2026 04:10 PM UTC",
		"[May 11, 2026 10:01 AM UTC] John Doe (Contact):",
		"Hi, I need a refund for order #555",
		"[May 11, 2026 10:05 AM UTC] Priya (Agent):",
		"Attachment: invoice.pdf",
		"Falls back to HTML",
	}
	for _, want := range wantContains {
		if !strings.Contains(out, want) {
			t.Errorf("transcript missing %q\n---\n%s", want, out)
		}
	}
}
