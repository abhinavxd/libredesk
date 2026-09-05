package conversation

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	wtmodels "github.com/abhinavxd/libredesk/internal/whatsapp_template/models"
	"github.com/knadh/go-i18n"
	"github.com/volatiletech/null/v9"
)

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
