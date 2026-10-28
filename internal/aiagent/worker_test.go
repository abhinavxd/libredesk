package aiagent

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/abhinavxd/libredesk/internal/ai"
	aimodels "github.com/abhinavxd/libredesk/internal/ai/models"
	"github.com/abhinavxd/libredesk/internal/aiagent/models"
	"github.com/abhinavxd/libredesk/internal/attachment"
	cmodels "github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/stringutil"
	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/volatiletech/null/v9"
	"github.com/zerodha/logf"
)

func TestSplitSuggestions(t *testing.T) {
	body, suggestions := splitSuggestions("Which plan?\n[[suggestions]] [\"Basic\",\"Pro\",\"Enterprise\",\"Extra\"]")
	if body != "Which plan?" {
		t.Fatalf("unexpected body: %q", body)
	}
	want := []string{"Basic", "Pro", "Enterprise"}
	if !reflect.DeepEqual(suggestions, want) {
		t.Fatalf("unexpected suggestions: %#v", suggestions)
	}
}

func TestSplitSuggestionsRejectsInvalidPayload(t *testing.T) {
	body, suggestions := splitSuggestions("Tell me more\n[[suggestions]] not-json")
	if body != "Tell me more" || suggestions != nil {
		t.Fatalf("unexpected result: %q %#v", body, suggestions)
	}
}

func TestSplitArticleSources(t *testing.T) {
	hits := []aimodels.SearchResult{
		{SourceID: 12, SourceType: aimodels.SourceHelpArticle},
		{SourceID: 34, SourceType: aimodels.SourceHelpArticle},
		{SourceID: 34, SourceType: aimodels.SourceHelpArticle},
		{SourceID: 56, SourceType: aimodels.SourceSnippet},
	}
	tests := []struct {
		name    string
		answer  string
		enabled bool
		body    string
		ids     []int
	}{
		{"inline claims", "Use JWT.[[cite:12]] Log out.[[cite:34]]", true, "Use JWT.[[cite:12]] Log out.[[cite:34]]", []int{12, 34}},
		{"repeated inline article", "Use JWT.[[cite:34]] More JWT.[[cite:34]]", true, "Use JWT.[[cite:34]] More JWT.[[cite:34]]", []int{34}},
		{"invalid inline articles", "Answer.[[cite:999]][[cite:56]][[cite:0]][[cite:-1]][[cite:no]]", true, "Answer.", nil},
		{"inline disabled", "Answer.[[cite:12]]", false, "Answer.", nil},
		{"mixed citation styles", "Answer.[[cite:34]]\n[[sources]] [12,34]", true, "Answer.[[cite:34]]", []int{34, 12}},
		{"used articles", "Answer.\n[[sources]] [34,12]", true, "Answer.", []int{34, 12}},
		{"duplicates and unknown IDs", "Answer.\n[[sources]] [12,12,0,-1,999,56,34]", true, "Answer.", []int{12, 34}},
		{"multiple searches and lines", "Answer.\n[[sources]] [12]\n[[sources]] [12,34]", true, "Answer.", []int{12, 34}},
		{"disabled", "Answer.\n[[sources]] [12]", false, "Answer.", nil},
		{"invalid payload", "Answer.\n[[sources]] [12,", true, "Answer.", nil},
		{"noninteger IDs", "Answer.\n[[sources]] [12,1.5]", true, "Answer.", nil},
		{"empty list", "Answer.\n[[sources]] []", true, "Answer.", nil},
		{"null", "Answer.\n[[sources]] null", true, "Answer.", nil},
		{"no marker", "Hello!", true, "Hello!", nil},
		{"inline marker", "Answer. [[sources]] [12]", true, "Answer.", []int{12}},
		{"next line array", "Answer.\n[[sources]]\n[12,34]", true, "Answer.", []int{12, 34}},
		{"multiline array", "Answer.\n[[sources]] [\n12,\n34\n]\nFollowup.", true, "Answer.\nFollowup.", []int{12, 34}},
		{"same line confirmation", "Answer.\n[[sources]] [12] [[confirm]] Did that help?", true, "Answer.\n[[confirm]] Did that help?", []int{12}},
		{"same line suggestions", "Answer.\n[[sources]] [12] [[suggestions]] [\"Yes\",\"No\"]", true, "Answer.\n[[suggestions]] [\"Yes\",\"No\"]", []int{12}},
		{"invalid array with confirmation", "Answer.\n[[sources]] [12, [[confirm]] Did that help?", true, "Answer.\n[[confirm]] Did that help?", nil},
		{"invalid payload with suggestions", "Answer.\n[[sources]] garbage [[suggestions]] [\"Yes\",\"No\"]", true, "Answer.\n[[suggestions]] [\"Yes\",\"No\"]", nil},
		{"noninteger multiline array", "Answer.\n[[sources]]\n[12,1.5]\n[[confirm]] Did that help?", true, "Answer.\n[[confirm]] Did that help?", nil},
		{"multiple markers on one line", "Answer.\n[[sources]] [12] [[sources]] [12,34] [[confirm]] Did that help?", true, "Answer.\n[[confirm]] Did that help?", []int{12, 34}},
		{"disabled multiline", "Answer.\n[[sources]]\n[12,34] [[confirm]] Did that help?", false, "Answer.\n[[confirm]] Did that help?", nil},
		{"with confirmation", "Answer.\n[[sources]] [12]\n[[confirm]]\nDid that help?\n[[suggestions]] [\"Yes\",\"No\"]", true, "Answer.\n[[confirm]]\nDid that help?\n[[suggestions]] [\"Yes\",\"No\"]", []int{12}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, ids := splitArticleSources(tt.answer, hits, tt.enabled)
			if body != tt.body || !reflect.DeepEqual(ids, tt.ids) {
				t.Fatalf("got %q, %v, want %q, %v", body, ids, tt.body, tt.ids)
			}
		})
	}
}

func TestArticleCitationPrompt(t *testing.T) {
	if strings.Contains(buildSystemPrompt(models.Assistant{}), "[[cite:") {
		t.Fatal("disabled assistant requests article references")
	}
	if !strings.Contains(buildSystemPrompt(models.Assistant{CitationsEnabled: true}), "[[cite:") {
		t.Fatal("enabled assistant does not request article references")
	}
}

func TestArticleCitationHTML(t *testing.T) {
	text := "Use **JWT**.[[cite:12]]\nDéconnectez-vous.[[cite:34]][[cite:999]]"
	content := articleCitationHTML(text, []int{12, 34})
	want := "<p>Use <strong>JWT</strong>.<!--ld-cite:12--><br>\nDéconnectez-vous.<!--ld-cite:34--></p>\n"
	if content != want {
		t.Fatalf("got %q, want %q", content, want)
	}
	plain := stringutil.HTML2Text(content)
	if strings.Contains(plain, "cite") || !strings.Contains(plain, "Déconnectez-vous.") {
		t.Fatalf("citation markers changed stored plain text: %q", plain)
	}
	if got := articleCitationHTML("Answer.[[cite:12]]", nil /* articleIDs */); got != "<p>Answer.</p>\n" {
		t.Fatalf("uncited reply retained a marker: %q", got)
	}
	link := articleCitationHTML(`[Article](https://example.com "[[cite:12]]")`, []int{12})
	if strings.Contains(link, "ld-cite") || strings.Contains(link, "[[cite:") {
		t.Fatalf("citation inserted inside an HTML attribute: %q", link)
	}
	if got := numberedCitationText("JWT.[[cite:12]] Logout.[[cite:34]][[cite:999]]", []int{12, 34}); got != "JWT.(1) Logout.(2)" {
		t.Fatalf("unexpected preview: %q", got)
	}
}

func TestPreviousReplyFirst(t *testing.T) {
	customer := func(id int) cmodels.Message {
		return cmodels.Message{ID: id, Type: cmodels.MessageIncoming, SenderType: cmodels.SenderTypeContact}
	}
	human := func(id int) cmodels.Message {
		return cmodels.Message{ID: id, Type: cmodels.MessageOutgoing, SenderType: cmodels.SenderTypeAgent}
	}
	reply := func(id int) cmodels.Message {
		msg := human(id)
		msg.Meta = json.RawMessage(`{"ai_assistant_id":7}`)
		return msg
	}
	tests := []struct {
		name     string
		msgs     []cmodels.Message
		prevSeen int
		want     []int
		respond  bool
	}{
		{"messages sent while replying", []cmodels.Message{customer(1), customer(2), customer(3), reply(4)}, 1, []int{1, 4, 2, 3}, true},
		{"split reply", []cmodels.Message{customer(1), customer(2), reply(3), reply(4)}, 1, []int{1, 3, 4, 2}, true},
		{"human answered meanwhile", []cmodels.Message{customer(1), customer(2), human(3), reply(4)}, 1, []int{1, 4, 2, 3}, false},
		{"no new message", []cmodels.Message{customer(1), reply(2)}, 1, []int{1, 2}, false},
		{"no reply posted", []cmodels.Message{customer(1), customer(2)}, 1, []int{1, 2}, true},
		{"older replies untouched", []cmodels.Message{customer(1), reply(2), customer(3), customer(4), reply(5)}, 3, []int{1, 2, 3, 5, 4}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msgs := previousReplyFirst(tt.msgs, tt.prevSeen)
			var got []int
			for _, m := range msgs {
				got = append(got, m.ID)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
			if respond := lastIsInboundContact(msgs); respond != tt.respond {
				t.Fatalf("respond = %v, want %v", respond, tt.respond)
			}
		})
	}
}

func TestBuildHistoryPreservesCustomerMessages(t *testing.T) {
	db := testutil.NewDB(t, "aiagent_worker_history")
	lo := logf.New(logf.Opts{})
	aiManager, err := ai.New(ai.Opts{Ctx: t.Context(), DB: db, Lo: &lo, I18n: testutil.NewI18n(t)})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(aiManager.Close)
	m := &Manager{ai: aiManager, lo: &lo}
	const contactID = 7
	const emptyMessage = "[The customer sent a message with no readable text.]"
	previous := []cmodels.Message{
		{ID: 1, SenderID: contactID, SenderType: cmodels.SenderTypeContact, Type: cmodels.MessageIncoming, TextContent: "Where is my order?"},
		{ID: 2, SenderType: cmodels.SenderTypeAgent, Type: cmodels.MessageOutgoing, TextContent: "Can you share your order number?"},
	}
	tests := []struct {
		name        string
		content     string
		contentType string
		attachments attachment.Attachments
		want        string
	}{
		{name: "blank", want: emptyMessage},
		{name: "whitespace", content: " \n\t ", want: emptyMessage},
		{name: "external image", content: `<img src="https://example.com/screenshot.png">`, contentType: cmodels.ContentTypeHTML, want: emptyMessage},
		{name: "empty HTML", content: "<p><br></p>", contentType: cmodels.ContentTypeHTML, want: emptyMessage},
		{name: "plain text", content: "Order 123", want: "Order 123"},
		{name: "HTML text", content: "<p>Order 123</p>", contentType: cmodels.ContentTypeHTML, want: "Order 123"},
		{name: "quoted reply", content: "> Order 123", want: "> Order 123"},
		{
			name:        "image without vision",
			attachments: attachment.Attachments{{Name: "screenshot.png", ContentType: "image/png", UUID: "image"}},
			want:        `[The customer attached an image "screenshot.png" that you cannot view. Ask them to describe it if it matters.]`,
		},
		{
			name:        "PDF attachment",
			attachments: attachment.Attachments{{Name: "invoice.pdf", ContentType: "application/pdf"}},
			want:        `[The customer attached a file "invoice.pdf" (application/pdf) that cannot be read here. Ask them to share the relevant details as text if it matters.]`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			latest := cmodels.Message{
				ID: 3, SenderID: contactID, SenderType: cmodels.SenderTypeContact, Type: cmodels.MessageIncoming,
				Content: tt.content, TextContent: tt.content, ContentType: tt.contentType, Attachments: tt.attachments,
			}
			if tt.contentType == cmodels.ContentTypeHTML {
				latest.TextContent = stringutil.HTML2Text(tt.content)
			}
			history := m.buildHistory(append(previous, latest), contactID)
			want := []aimodels.ChatMessage{
				{Role: aimodels.RoleUser, Content: previous[0].TextContent},
				{Role: aimodels.RoleAssistant, Content: previous[1].TextContent},
				{Role: aimodels.RoleUser, Content: tt.want},
			}
			if !reflect.DeepEqual(history, want) {
				t.Fatalf("got %#v, want %#v", history, want)
			}
		})
	}
	t.Run("subject without body", func(t *testing.T) {
		history := m.buildHistory([]cmodels.Message{{SenderType: cmodels.SenderTypeContact, SenderID: contactID, Type: cmodels.MessageIncoming}}, contactID)
		conv := cmodels.Conversation{Subject: null.StringFrom("How do I reset my password?")}
		block := customerContextBlock(conv, nil /** contactLines **/)
		history = append([]aimodels.ChatMessage{{Role: aimodels.RoleUser, Content: block}}, history...)
		if len(history) != 2 || !strings.Contains(history[0].Content, conv.Subject.String) || history[1].Content != emptyMessage {
			t.Fatalf("subject or customer message missing: %#v", history)
		}
	})
	t.Run("consecutive blank messages", func(t *testing.T) {
		blank := cmodels.Message{SenderType: cmodels.SenderTypeContact, SenderID: contactID, Type: cmodels.MessageIncoming}
		history := m.buildHistory([]cmodels.Message{previous[0], blank, blank}, contactID)
		want := []aimodels.ChatMessage{
			{Role: aimodels.RoleUser, Content: previous[0].TextContent},
			{Role: aimodels.RoleUser, Content: emptyMessage},
			{Role: aimodels.RoleUser, Content: emptyMessage},
		}
		if !reflect.DeepEqual(history, want) {
			t.Fatalf("got %#v, want %#v", history, want)
		}
	})
	t.Run("empty assistant message", func(t *testing.T) {
		history := m.buildHistory([]cmodels.Message{{SenderType: cmodels.SenderTypeAgent, Type: cmodels.MessageOutgoing}}, contactID)
		if len(history) != 0 {
			t.Fatalf("empty assistant message kept: %#v", history)
		}
	})
	t.Run("empty history", func(t *testing.T) {
		if history := m.buildHistory(nil /** msgs **/, contactID); len(history) != 0 {
			t.Fatalf("unexpected history: %#v", history)
		}
	})
}
