package conversation

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/abhinavxd/libredesk/internal/inbox"
	whatsappChannel "github.com/abhinavxd/libredesk/internal/inbox/channel/whatsapp"
	mmodels "github.com/abhinavxd/libredesk/internal/media/models"
	"github.com/abhinavxd/libredesk/internal/template"
	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/abhinavxd/libredesk/internal/user"
	wmodels "github.com/abhinavxd/libredesk/internal/webhook/models"
	wtmodels "github.com/abhinavxd/libredesk/internal/whatsapp/template/models"
	"github.com/abhinavxd/libredesk/internal/ws"
	"github.com/jmoiron/sqlx"
	"github.com/knadh/go-i18n"
	"github.com/volatiletech/null/v9"
	"github.com/zerodha/logf"
)

type whatsAppReplyMedia struct{ mediaStore }

type whatsAppReplyWebhook struct{ webhookStore }

type whatsAppReplyTemplates struct {
	WhatsAppTemplateStore
	template wtmodels.Template
}

func (whatsAppReplyMedia) LinkMessageMediaTx(*sqlx.Tx, int, []mmodels.Media, []string, int) error {
	return nil
}

func (whatsAppReplyWebhook) TriggerEvent(wmodels.WebhookEvent, any) {}

func (s whatsAppReplyTemplates) GetByID(int) (wtmodels.Template, error) {
	return s.template, nil
}

func TestQueueWhatsAppReplyVariables(t *testing.T) {
	db := testutil.NewDB(t, "whatsapp_reply_variables")
	lo := logf.New(logf.Opts{Level: logf.FatalLevel})
	lang := testutil.NewI18n(t)
	inboxes, err := inbox.New(&lo, db, lang, "0123456789abcdef0123456789abcdef")
	if err != nil {
		t.Fatal(err)
	}
	users, err := user.New(lang, user.Opts{DB: db, Lo: &lo})
	if err != nil {
		t.Fatal(err)
	}
	m := &Manager{
		db: db, lo: &lo, i18n: lang, inboxStore: inboxes, userStore: users,
		template: &template.Manager{}, mediaStore: whatsAppReplyMedia{},
		webhookStore: whatsAppReplyWebhook{}, wsHub: ws.NewHub(&lo, nil /** userStore **/),
	}
	if err := dbutil.ScanSQLFile("queries.sql", &m.q, db, efs); err != nil {
		t.Fatal(err)
	}
	var contactID, senderID, inboxID int
	if err := db.Get(&contactID, `INSERT INTO users (type, first_name, last_name)
		VALUES ('contact', 'Customer', 'A') RETURNING id`); err != nil {
		t.Fatal(err)
	}
	if err := db.Get(&senderID, `INSERT INTO users (type, email, first_name, last_name)
		VALUES ('agent', 'agent@example.com', 'Agent', '') RETURNING id`); err != nil {
		t.Fatal(err)
	}
	if err := db.Get(&inboxID, `INSERT INTO inboxes (channel, name, "from")
		VALUES ('whatsapp', 'Reply variables', '') RETURNING id`); err != nil {
		t.Fatal(err)
	}
	if _, err := users.LinkChannelIdentity(contactID, "whatsapp", "15550000100"); err != nil {
		t.Fatal(err)
	}
	var conv models.Conversation
	if err := db.Get(&conv, `INSERT INTO conversations (contact_id, inbox_id, status_id, subject, last_inbound_at)
		VALUES ($1, $2, (SELECT id FROM conversation_statuses WHERE category = 'open' LIMIT 1), 'Support request', NOW())
		RETURNING id, uuid, reference_number`, contactID, inboxID); err != nil {
		t.Fatal(err)
	}
	m.whatsappTemplate = whatsAppReplyTemplates{template: wtmodels.Template{
		ID: 7, InboxID: inboxID, Name: "greeting", Language: "en", Status: wtmodels.StatusApproved,
		BodyContent: "Hello {{1}}", ComponentTypes: []string{"BODY"},
	}}

	for _, tc := range []struct {
		name    string
		content string
		meta    map[string]any
		want    string
	}{
		{"contact and reference", "Hi {{ .Contact.FirstName }}, reference {{ .Conversation.ReferenceNumber }}", map[string]any{}, "Hi Customer, reference " + conv.ReferenceNumber},
		{"automated reply", "{{ .Recipient.FullName }}: {{ .Conversation.Subject }}", map[string]any{"is_automated": true}, "Customer A: Support request"},
		{"repeated variables", "{{ .Contact.FirstName }} {{ .Contact.FirstName }}", map[string]any{}, "Customer Customer"},
		{"zero template id", "{{ .Contact.FirstName }}", map[string]any{"whatsapp_template_id": 0}, "Customer"},
		{"negative template id", "{{ .Contact.FirstName }}", map[string]any{"whatsapp_template_id": -1}, "Customer"},
		{"decoded template id", "ignored", map[string]any{"whatsapp_template_id": float64(7), "whatsapp_template_params": map[string]string{"body:1": "{{ .Contact.FirstName }}"}}, "Hello {{ .Contact.FirstName }}"},
		{"plain text", "Hello", map[string]any{}, "Hello"},
		{"rendered text at limit", strings.Repeat("x", whatsAppMaxTextLength-len("Customer")) + "{{ .Contact.FirstName }}", map[string]any{}, strings.Repeat("x", whatsAppMaxTextLength-len("Customer")) + "Customer"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			message, err := m.QueueReply(nil, /** media **/
				inboxID, senderID, contactID, conv.UUID, tc.content,
				nil, /** to **/
				nil, /** cc **/
				nil, /** bcc **/
				tc.meta)
			if err != nil {
				t.Fatal(err)
			}
			if message.Content != tc.want || message.TextContent != tc.want {
				t.Fatalf("queued content = %q, text = %q, want %q", message.Content, message.TextContent, tc.want)
			}
			if tc.name == "decoded template id" {
				var meta struct {
					WhatsApp whatsappChannel.SendMeta `json:"whatsapp"`
				}
				if err := json.Unmarshal(message.Meta, &meta); err != nil {
					t.Fatal(err)
				}
				if message.ContentType != models.ContentTypeText || meta.WhatsApp.TemplateParams["body:1"] != "{{ .Contact.FirstName }}" {
					t.Fatalf("Meta template content was changed: %+v", message)
				}
			}
		})
	}

	t.Run("expanded content exceeds limit", func(t *testing.T) {
		if _, err := db.Exec(`UPDATE users SET first_name = $1 WHERE id = $2`, strings.Repeat("a", 64), contactID); err != nil {
			t.Fatal(err)
		}
		content := strings.Repeat("x", whatsAppMaxTextLength-32) + "{{ .Contact.FirstName }}"
		if _, err := m.QueueReply(nil, /** media **/
			inboxID, senderID, contactID, conv.UUID, content,
			nil, /** to **/
			nil, /** cc **/
			nil, /** bcc **/
			map[string]any{}); err == nil || !strings.Contains(err.Error(), "4096") {
			t.Fatalf("expected rendered content to exceed the limit, got %v", err)
		}
	})
}

func TestRenderTemplateBody(t *testing.T) {
	tests := []struct {
		name   string
		body   string
		params map[string]string
		want   string
	}{
		{
			name:   "named placeholders",
			body:   "Hi {{name}}, order {{order_id}} is {{status}}.",
			params: map[string]string{"body:name": "Ravi", "body:order_id": "A1", "body:status": "shipped"},
			want:   "Hi Ravi, order A1 is shipped.",
		},
		{
			name:   "positional placeholders",
			body:   "Hi {{1}}, order {{2}}.",
			params: map[string]string{"body:1": "Ravi", "body:2": "A1"},
			want:   "Hi Ravi, order A1.",
		},
		{
			name:   "header params never fill the body",
			body:   "Order {{order_id}}",
			params: map[string]string{"header:order_id": "A1"},
			want:   "Order {{order_id}}",
		},
		{
			name:   "unmatched placeholder stays verbatim",
			body:   "Hi {{name}}",
			params: map[string]string{"body:other": "x"},
			want:   "Hi {{name}}",
		},
		{
			name:   "no params",
			body:   "Hi {{name}}",
			params: nil,
			want:   "Hi {{name}}",
		},
		{
			name:   "repeated placeholder fills every occurrence",
			body:   "{{name}} and {{name}}",
			params: map[string]string{"body:name": "Ravi"},
			want:   "Ravi and Ravi",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := renderTemplateBody(tc.body, tc.params); got != tc.want {
				t.Fatalf("expected %q, got %q", tc.want, got)
			}
		})
	}
}

func TestValidateTemplateParams(t *testing.T) {
	urlButton, _ := json.Marshal([]map[string]any{{"type": "URL", "text": "Track order", "url": "https://x.test/{{1}}"}})
	staticButton, _ := json.Marshal([]map[string]any{{"type": "URL", "text": "Home", "url": "https://x.test/"}})

	base := wtmodels.Template{BodyContent: "Hi {{name}}, order {{order_id}}."}
	withHeader := wtmodels.Template{
		BodyContent:   "Hi {{name}}",
		HeaderType:    null.StringFrom("TEXT"),
		HeaderContent: null.StringFrom("Order {{order_id}}"),
	}

	tests := []struct {
		name      string
		template  wtmodels.Template
		params    map[string]string
		wantMatch string
	}{
		{
			name:     "all body params filled",
			template: base,
			params:   map[string]string{"body:name": "Ravi", "body:order_id": "A1"},
		},
		{
			name:      "missing body param",
			template:  base,
			params:    map[string]string{"body:name": "Ravi"},
			wantMatch: "order_id",
		},
		{
			name:      "blank body param",
			template:  base,
			params:    map[string]string{"body:name": "Ravi", "body:order_id": "   "},
			wantMatch: "order_id",
		},
		{
			name:      "missing text header param",
			template:  withHeader,
			params:    map[string]string{"body:name": "Ravi"},
			wantMatch: "header's {{order_id}}",
		},
		{
			name:     "header param filled",
			template: withHeader,
			params:   map[string]string{"body:name": "Ravi", "header:order_id": "A1"},
		},
		{
			name:      "missing url button param",
			template:  wtmodels.Template{BodyContent: "Hi", Buttons: urlButton},
			params:    nil,
			wantMatch: "Track order",
		},
		{
			name:     "url button param filled",
			template: wtmodels.Template{BodyContent: "Hi", Buttons: urlButton},
			params:   map[string]string{"button_url_0": "A1"},
		},
		{
			name:     "static url button needs no param",
			template: wtmodels.Template{BodyContent: "Hi", Buttons: staticButton},
			params:   nil,
		},
		{
			name:     "media header needs no param",
			template: wtmodels.Template{BodyContent: "Hi", HeaderType: null.StringFrom("IMAGE")},
			params:   nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := testManager(t).validateTemplateParams(tc.template, tc.params)
			if tc.wantMatch == "" {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected an error mentioning %q", tc.wantMatch)
			}
			if !strings.Contains(err.Error(), tc.wantMatch) {
				t.Fatalf("expected the error to mention %q, got %q", tc.wantMatch, err.Error())
			}
		})
	}
}

func TestValidateWhatsAppContent(t *testing.T) {
	tests := []struct {
		name           string
		content        string
		hasAttachments bool
		wantMatch      string
	}{
		{name: "attachment without caption", hasAttachments: true},
		{name: "caption at limit", content: strings.Repeat("a", 1024), hasAttachments: true},
		{name: "caption over limit", content: strings.Repeat("a", 1025), hasAttachments: true, wantMatch: "1024"},
		{name: "unicode caption counts runes", content: strings.Repeat("श", 1024), hasAttachments: true},
		{name: "text at limit", content: strings.Repeat("a", 4096)},
		{name: "text over limit", content: strings.Repeat("a", 4097), wantMatch: "4096"},
		{name: "empty text without attachment", wantMatch: "attach a file"},
	}

	manager := testManager(t)
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := manager.validateWhatsAppContent(tc.content, tc.hasAttachments)
			if tc.wantMatch == "" {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected an error mentioning %q", tc.wantMatch)
			}
			if !strings.Contains(strings.ToLower(err.Error()), tc.wantMatch) {
				t.Fatalf("expected the error to mention %q, got %q", tc.wantMatch, err.Error())
			}
		})
	}
}

func TestExtractInt(t *testing.T) {
	tests := []struct {
		name string
		meta map[string]any
		want int
	}{
		{"int", map[string]any{"id": 7}, 7},
		{"int64", map[string]any{"id": int64(7)}, 7},
		{"float64 from json", map[string]any{"id": float64(7)}, 7},
		{"json number", map[string]any{"id": json.Number("7")}, 7},
		{"string is not a number", map[string]any{"id": "7"}, 0},
		{"absent", map[string]any{}, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := extractInt(tc.meta, "id"); got != tc.want {
				t.Fatalf("expected %d, got %d", tc.want, got)
			}
		})
	}
}

func TestExtractStringMap(t *testing.T) {
	t.Run("typed map", func(t *testing.T) {
		got := extractStringMap(map[string]any{"p": map[string]string{"body:name": "Ravi"}}, "p")
		if got["body:name"] != "Ravi" {
			t.Fatalf("unexpected params: %+v", got)
		}
	})

	t.Run("decoded map coerces scalars", func(t *testing.T) {
		got := extractStringMap(map[string]any{"p": map[string]any{
			"body:name":  "Ravi",
			"body:count": json.Number("2"),
			"body:flag":  true,
		}}, "p")
		if got["body:name"] != "Ravi" || got["body:count"] != "2" || got["body:flag"] != "true" {
			t.Fatalf("unexpected params: %+v", got)
		}
	})

	t.Run("empty and missing", func(t *testing.T) {
		if got := extractStringMap(map[string]any{"p": map[string]any{}}, "p"); got != nil {
			t.Fatalf("expected nil for an empty map, got %+v", got)
		}
		if got := extractStringMap(map[string]any{}, "p"); got != nil {
			t.Fatalf("expected nil for a missing key, got %+v", got)
		}
	})
}

func TestCSATTokensHiddenFromAgentResponses(t *testing.T) {
	tests := []struct {
		name string
		meta string
		want string
	}{
		{"all copies", `{"is_csat":true,"csat_uuid":"secret","whatsapp_template_params":{"button_url_0":"secret","body:name":"Alex"},"whatsapp":{"template_params":{"button_url_0":"secret"},"template_name":"survey"}}`, `{"is_csat":true,"whatsapp_template_params":{"body:name":"Alex"},"whatsapp":{"template_params":{},"template_name":"survey"}}`},
		{"nested copies only", `{"is_csat":true,"whatsapp_template_params":{"button_url_0":"secret"},"whatsapp":{"template_params":{"button_url_0":"secret"}}}`, `{"is_csat":true,"whatsapp_template_params":{},"whatsapp":{"template_params":{}}}`},
		{"token without flag", `{"csat_uuid":"secret","whatsapp_template_params":{"button_url_0":"secret"}}`, `{"whatsapp_template_params":{}}`},
		{"ordinary template", `{"is_csat":false,"whatsapp_template_params":{"button_url_0":"order-123"},"whatsapp":{"template_params":{"button_url_0":"order-123"}}}`, `{"is_csat":false,"whatsapp_template_params":{"button_url_0":"order-123"},"whatsapp":{"template_params":{"button_url_0":"order-123"}}}`},
		{"null maps", `{"is_csat":true,"csat_uuid":"secret","whatsapp_template_params":null,"whatsapp":{"template_params":null}}`, `{"is_csat":true,"whatsapp_template_params":null,"whatsapp":{"template_params":null}}`},
	}
	for _, tt := range tests {
		for name, redact := range map[string]func(json.RawMessage) json.RawMessage{
			"HTTP": func(meta json.RawMessage) json.RawMessage {
				msg := models.Message{Meta: meta}
				msg.StripCSATUUID()
				return msg.Meta
			},
			"WebSocket": stripCSATUUID,
		} {
			t.Run(tt.name+"/"+name, func(t *testing.T) {
				original := json.RawMessage(tt.meta)
				got := redact(original)
				var want any
				if err := json.Unmarshal([]byte(tt.want), &want); err != nil {
					t.Fatal(err)
				}
				var result any
				if err := json.Unmarshal(got, &result); err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(result, want) {
					t.Fatalf("got %s, want %s", got, tt.want)
				}
				if string(original) != tt.meta {
					t.Fatal("redaction mutated the stored metadata")
				}
			})
		}
	}
}

func TestStripCSATUUID(t *testing.T) {
	stripped := stripCSATUUID(json.RawMessage(`{"is_csat":true,"csat_uuid":"secret-uuid"}`))
	var out map[string]any
	if err := json.Unmarshal(stripped, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := out["csat_uuid"]; ok {
		t.Fatal("csat_uuid must not reach the websocket payload")
	}
	if out["is_csat"] != true {
		t.Fatalf("expected the rest of the meta to survive, got %+v", out)
	}

	same := json.RawMessage(`{"is_csat":true}`)
	if string(stripCSATUUID(same)) != string(same) {
		t.Fatal("meta without a csat_uuid must pass through unchanged")
	}
	if got := stripCSATUUID(nil); got != nil {
		t.Fatalf("expected nil meta to pass through, got %q", got)
	}
	malformed := json.RawMessage(`not json`)
	if string(stripCSATUUID(malformed)) != string(malformed) {
		t.Fatal("malformed meta must pass through unchanged")
	}
}

// testManager carries the real language file, so a renamed i18n key fails the test.
func testManager(t *testing.T) *Manager {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "i18n", "en-US.json"))
	if err != nil {
		t.Fatalf("reading the language file: %v", err)
	}
	lang, err := i18n.New(raw)
	if err != nil {
		t.Fatalf("loading i18n: %v", err)
	}
	return &Manager{i18n: lang}
}
