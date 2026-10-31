package conversation

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/abhinavxd/libredesk/internal/attachment"
	"github.com/abhinavxd/libredesk/internal/conversation/models"
	hcmodels "github.com/abhinavxd/libredesk/internal/helpcenter/models"
	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/volatiletech/null/v9"
)

func TestBuildTranscriptPDF(t *testing.T) {
	for _, text := range []string{
		"",
		"Conversation #1234\n\nCafé, Grüße, Καλημέρα, Привет\nAttachment: invoice.pdf\n",
		"Windows\r\n\r\nIndented\tmessage",
		strings.Repeat("https://example.com/verylongpath", 100),
	} {
		pdf, err := BuildTranscriptPDF([]byte(text))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.HasPrefix(pdf, []byte("%PDF-")) || !bytes.Contains(pdf, []byte("%%EOF")) {
			t.Fatal("transcript is not a complete PDF")
		}
	}
}

func TestBuildTranscriptPDFPagination(t *testing.T) {
	pdf, err := BuildTranscriptPDF([]byte(strings.Repeat("Message on a new line\n", 200)))
	if err != nil {
		t.Fatal(err)
	}
	if pages := bytes.Count(pdf, []byte("/Type /Page\n")); pages < 4 {
		t.Fatalf("long transcript has %d pages, want at least 4", pages)
	}
}

func TestBuildTranscriptPDFUnsupportedCharacters(t *testing.T) {
	for _, text := range []string{"Hello 世界", "नमस्कार", "Hello 👋"} {
		pdf, err := BuildTranscriptPDF([]byte(text))
		if !errors.Is(err, ErrTranscriptPDFUnsupportedCharacters) || len(pdf) != 0 {
			t.Fatalf("unsupported text %q: got %d bytes and error %v", text, len(pdf), err)
		}
	}
}

func TestBuildTranscript(t *testing.T) {
	m := &Manager{i18n: testutil.NewI18n(t)}

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

func TestBuildTranscriptArticleReferences(t *testing.T) {
	m := Manager{
		articleReferenceStore: citationArticleStore{references: []hcmodels.ArticleReference{
			{ID: 12, Title: "Refund policy", HelpCenterSlug: "support", Locale: "en", Slug: "refunds"},
		}},
		settingsStore: citationSettingsStore{rootURL: "https://desk.example.com"},
		i18n:          testutil.NewI18n(t),
	}
	for _, text := range []string{"Answer.", ""} {
		messages := []models.Message{{
			Content:     "<p>Answer.</p>",
			TextContent: text,
			Meta:        json.RawMessage(`{"ai_assistant_id":1,"ai_article_ids":[12]}`),
		}}
		out := string(m.BuildTranscript(models.Conversation{}, messages, time.Now() /* downloadedAt */))
		for _, want := range []string{"Answer.", "Article references", "Refund policy", "https://desk.example.com/hc/support/en/articles/refunds"} {
			if !strings.Contains(out, want) {
				t.Errorf("transcript missing %q: %s", want, out)
			}
		}
		if strings.Contains(messages[0].Content, "https://") {
			t.Fatal("transcript rendering changed the saved reply")
		}
	}
}
