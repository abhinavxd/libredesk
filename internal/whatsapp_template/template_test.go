package whatsapp_template

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/abhinavxd/libredesk/internal/whatsapp"
	"github.com/abhinavxd/libredesk/internal/whatsapp_template/models"
	"github.com/volatiletech/null/v9"
)

// A URL button with a {{1}} placeholder must ship a button example or Meta rejects the submission.
func TestBuildSubmissionCSATButtonExample(t *testing.T) {
	buttons, _ := json.Marshal([]map[string]any{{
		"type": "URL",
		"text": "Rate us",
		"url":  "http://localhost:9000/csat/{{1}}",
	}})
	sub, err := buildSubmission(models.Template{
		InboxID:      2,
		Name:         "libredesk_csat_2",
		Language:     "en_US",
		Category:     "UTILITY",
		BodyContent:  "Your conversation has been resolved.",
		Buttons:      buttons,
		SampleValues: json.RawMessage(`{"1":"example"}`),
	})
	if err != nil {
		t.Fatalf("buildSubmission errored: %v", err)
	}
	for _, c := range sub.Components {
		if c.Type != "BUTTONS" {
			continue
		}
		want := "http://localhost:9000/csat/example"
		if len(c.Buttons) == 0 || len(c.Buttons[0].Example) != 1 || c.Buttons[0].Example[0] != want {
			t.Fatalf("expected URL button example %q, got %+v", want, c.Buttons)
		}
		return
	}
	t.Fatalf("expected a BUTTONS component, got %+v", sub.Components)
}

// Numbered placeholders are positional, so Meta wants a nested body example and no parameter_format.
func TestBuildSubmissionPositionalBodyExample(t *testing.T) {
	sub, err := buildSubmission(models.Template{
		Name:         "order_update",
		Language:     "en_US",
		Category:     "utility",
		BodyContent:  "Hi {{1}}, order {{2}} shipped.",
		SampleValues: json.RawMessage(`{"1":"Ravi","2":"A1"}`),
	})
	if err != nil {
		t.Fatalf("buildSubmission errored: %v", err)
	}
	if sub.ParameterFormat != "" {
		t.Fatalf("expected no parameter_format for positional placeholders, got %q", sub.ParameterFormat)
	}
	if sub.Category != "UTILITY" {
		t.Fatalf("expected the category to be upper-cased, got %q", sub.Category)
	}
	body, ok := componentByType(sub, "BODY")
	if !ok {
		t.Fatal("expected a BODY component")
	}
	vals, ok := body.Example["body_text"].([][]string)
	if !ok || len(vals) != 1 || len(vals[0]) != 2 || vals[0][0] != "Ravi" || vals[0][1] != "A1" {
		t.Fatalf("expected nested positional body example, got %#v", body.Example)
	}
}

func TestBuildSubmissionNamedBodyAndHeaderExample(t *testing.T) {
	sub, err := buildSubmission(models.Template{
		Name:          "order_update",
		Language:      "en_US",
		Category:      "UTILITY",
		HeaderType:    null.StringFrom("TEXT"),
		HeaderContent: null.StringFrom("Order {{order_id}}"),
		BodyContent:   "Hi {{name}}, order {{order_id}} shipped.",
		FooterContent: null.StringFrom("Libredesk"),
		SampleValues:  json.RawMessage(`{"name":"Ravi","order_id":"A1"}`),
	})
	if err != nil {
		t.Fatalf("buildSubmission errored: %v", err)
	}
	if sub.ParameterFormat != "NAMED" {
		t.Fatalf("expected parameter_format NAMED, got %q", sub.ParameterFormat)
	}
	header, ok := componentByType(sub, "HEADER")
	if !ok {
		t.Fatal("expected a HEADER component")
	}
	if _, ok := header.Example["header_text_named_params"]; !ok {
		t.Fatalf("expected named header example, got %#v", header.Example)
	}
	body, _ := componentByType(sub, "BODY")
	params, ok := body.Example["body_text_named_params"].([]map[string]any)
	if !ok || len(params) != 2 || params[0]["param_name"] != "name" {
		t.Fatalf("expected named body params in placeholder order, got %#v", body.Example)
	}
	if _, ok := componentByType(sub, "FOOTER"); !ok {
		t.Fatal("expected a FOOTER component")
	}
}

func TestBuildSubmissionMissingSampleValue(t *testing.T) {
	_, err := buildSubmission(models.Template{
		Name:        "order_update",
		Language:    "en_US",
		Category:    "UTILITY",
		BodyContent: "Hi {{name}}",
	})
	if err == nil {
		t.Fatal("expected an error when a placeholder has no sample value")
	}
	if !strings.Contains(err.Error(), "name") {
		t.Fatalf("expected the error to name the placeholder, got %q", err.Error())
	}
}

// A media header carries no text, so it must ship without an example.
func TestBuildSubmissionMediaHeaderHasNoExample(t *testing.T) {
	sub, err := buildSubmission(models.Template{
		Name:        "promo",
		Language:    "en_US",
		Category:    "MARKETING",
		HeaderType:  null.StringFrom("IMAGE"),
		BodyContent: "Seasonal offer.",
	})
	if err != nil {
		t.Fatalf("buildSubmission errored: %v", err)
	}
	header, ok := componentByType(sub, "HEADER")
	if !ok {
		t.Fatal("expected a HEADER component")
	}
	if header.Format != "IMAGE" || header.Text != "" || header.Example != nil {
		t.Fatalf("unexpected media header: %+v", header)
	}
}

func TestBuildEditDropsNameAndLanguage(t *testing.T) {
	edit, err := buildEdit(models.Template{
		Name:         "libredesk_csat_2",
		Language:     "en_US",
		Category:     "UTILITY",
		BodyContent:  "Hi {{1}}",
		SampleValues: json.RawMessage(`{"1":"Ravi"}`),
	})
	if err != nil {
		t.Fatalf("buildEdit errored: %v", err)
	}
	raw, err := json.Marshal(edit)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	for _, field := range []string{`"name"`, `"language"`} {
		if strings.Contains(string(raw), field) {
			t.Fatalf("edit payload must not carry %s: %s", field, raw)
		}
	}
	if len(edit.Components) == 0 {
		t.Fatal("expected the edit to carry components")
	}
}

func TestBuildSubmissionPositionalHeaderExample(t *testing.T) {
	sub, err := buildSubmission(models.Template{
		Name: "order_update", Language: "en_US", Category: "UTILITY",
		HeaderType: null.StringFrom("TEXT"), HeaderContent: null.StringFrom("Order {{1}}"),
		BodyContent: "Hi {{2}}", SampleValues: json.RawMessage(`{"1":"A1","2":"Ravi"}`),
	})
	if err != nil {
		t.Fatalf("buildSubmission errored: %v", err)
	}
	header, _ := componentByType(sub, "HEADER")
	vals, ok := header.Example["header_text"].([]string)
	if !ok || len(vals) != 1 || vals[0] != "A1" {
		t.Fatalf("expected a flat positional header example, got %#v", header.Example)
	}
}

func TestBuildSubmissionMissingHeaderSampleValue(t *testing.T) {
	_, err := buildSubmission(models.Template{
		Name: "order_update", Language: "en_US", Category: "UTILITY",
		HeaderType: null.StringFrom("TEXT"), HeaderContent: null.StringFrom("Order {{order_id}}"),
		BodyContent: "Hi", SampleValues: json.RawMessage(`{}`),
	})
	if err == nil || !strings.Contains(err.Error(), "order_id") {
		t.Fatalf("expected a missing header sample error, got %v", err)
	}
}

func TestBuildSubmissionMissingButtonSampleValue(t *testing.T) {
	buttons, _ := json.Marshal([]map[string]any{{"type": "URL", "text": "Track", "url": "https://x.test/{{track}}"}})
	_, err := buildSubmission(models.Template{
		Name: "order_update", Language: "en_US", Category: "UTILITY",
		BodyContent: "Hi", Buttons: buttons, SampleValues: json.RawMessage(`{}`),
	})
	if err == nil || !strings.Contains(err.Error(), "track") {
		t.Fatalf("expected a missing button sample error, got %v", err)
	}
}

// Quick replies and static links carry no example, so they must pass through untouched.
func TestBuildSubmissionButtonsWithoutPlaceholders(t *testing.T) {
	buttons, _ := json.Marshal([]map[string]any{
		{"type": "QUICK_REPLY", "text": "Yes"},
		{"type": "URL", "text": "Home", "url": "https://x.test/"},
	})
	sub, err := buildSubmission(models.Template{
		Name: "promo", Language: "en_US", Category: "MARKETING", BodyContent: "Hi", Buttons: buttons,
	})
	if err != nil {
		t.Fatalf("buildSubmission errored: %v", err)
	}
	comp, ok := componentByType(sub, "BUTTONS")
	if !ok || len(comp.Buttons) != 2 {
		t.Fatalf("unexpected buttons component: %+v", comp)
	}
	for _, b := range comp.Buttons {
		if len(b.Example) != 0 {
			t.Fatalf("expected no example on %s, got %v", b.Text, b.Example)
		}
	}
}

// A button that already carries its example must not be rewritten from the sample values.
func TestBuildSubmissionKeepsExistingButtonExample(t *testing.T) {
	buttons, _ := json.Marshal([]map[string]any{{"type": "URL", "text": "Track", "url": "https://x.test/{{1}}", "example": []string{"https://x.test/kept"}}})
	sub, err := buildSubmission(models.Template{
		Name: "order_update", Language: "en_US", Category: "UTILITY",
		BodyContent: "Hi", Buttons: buttons, SampleValues: json.RawMessage(`{"1":"other"}`),
	})
	if err != nil {
		t.Fatalf("buildSubmission errored: %v", err)
	}
	comp, _ := componentByType(sub, "BUTTONS")
	if comp.Buttons[0].Example[0] != "https://x.test/kept" {
		t.Fatalf("unexpected example: %v", comp.Buttons[0].Example)
	}
}

// An unreadable buttons column must not stop the rest of the template from being submitted.
func TestBuildSubmissionSkipsUnreadableButtons(t *testing.T) {
	sub, err := buildSubmission(models.Template{
		Name: "promo", Language: "en_US", Category: "MARKETING", BodyContent: "Hi", Buttons: json.RawMessage(`not json`),
	})
	if err != nil {
		t.Fatalf("buildSubmission errored: %v", err)
	}
	if _, ok := componentByType(sub, "BUTTONS"); ok {
		t.Fatal("expected no buttons component")
	}
}

func TestMapTemplateEventToStatus(t *testing.T) {
	tests := map[string]string{
		"APPROVED":                 models.StatusApproved,
		"approved":                 models.StatusApproved,
		"REINSTATED":               models.StatusApproved,
		"REJECTED":                 models.StatusRejected,
		"PAUSED":                   models.StatusPaused,
		"DISABLED":                 models.StatusDisabled,
		"PENDING_DELETION":         "",
		"FLAGGED":                  "",
		"":                         "",
		"some_future_meta_event":   "",
		"TEMPLATE_QUALITY_UPDATE ": "",
	}
	for event, want := range tests {
		if got := mapTemplateEventToStatus(event); got != want {
			t.Errorf("%q: expected %q, got %q", event, want, got)
		}
	}
}

func TestMetaToRow(t *testing.T) {
	mt := whatsapp.MetaTemplate{
		ID:       "123",
		Name:     "order_update",
		Language: "en_US",
		Category: "UTILITY",
		Status:   "approved",
		Components: []whatsapp.TemplateComponent{
			{Type: "header", Format: "text", Text: "Order {{1}}"},
			{Type: "BODY", Text: "Hi {{1}}"},
			{Type: "FOOTER", Text: "Libredesk"},
			{Type: "BUTTONS", Buttons: []whatsapp.TemplateButton{{Type: "URL", Text: "Track", URL: "https://x.test/{{1}}"}}},
		},
		RejectedReason: "NONE",
	}
	row := metaToRow(9, mt)
	if row.InboxID != 9 || row.MetaTemplateID.String != "123" || row.Status != "APPROVED" {
		t.Fatalf("unexpected row: %+v", row)
	}
	if row.HeaderType.String != "TEXT" || row.HeaderContent.String != "Order {{1}}" {
		t.Fatalf("unexpected header: %+v", row)
	}
	if row.BodyContent != "Hi {{1}}" || row.FooterContent.String != "Libredesk" {
		t.Fatalf("unexpected body/footer: %+v", row)
	}
	// Meta sends "NONE" when nothing is wrong, which would read as a real rejection reason.
	if row.RejectionReason.Valid {
		t.Fatalf("expected no rejection reason, got %q", row.RejectionReason.String)
	}
	var btns []whatsapp.TemplateButton
	if err := json.Unmarshal(row.Buttons, &btns); err != nil || len(btns) != 1 || btns[0].Text != "Track" {
		t.Fatalf("unexpected buttons %s (err %v)", row.Buttons, err)
	}

	empty := metaToRow(9, whatsapp.MetaTemplate{Name: "x", Language: "en_US", Status: "PENDING"})
	if string(empty.Buttons) != `[]` || string(empty.SampleValues) != `{}` {
		t.Fatalf("expected JSON defaults, got buttons=%s sample=%s", empty.Buttons, empty.SampleValues)
	}

	rejected := metaToRow(9, whatsapp.MetaTemplate{Name: "x", Language: "en_US", Status: "REJECTED", RejectedReason: "INVALID_FORMAT"})
	if rejected.RejectionReason.String != "INVALID_FORMAT" {
		t.Fatalf("expected the real rejection reason to be kept, got %+v", rejected.RejectionReason)
	}
}

func TestReservedContentChanged(t *testing.T) {
	buttons := func(text, url string) json.RawMessage {
		b, _ := json.Marshal([]map[string]any{{"type": "URL", "text": text, "url": url}})
		return b
	}
	existing := models.Template{BodyContent: "Rate us please", Buttons: buttons("Rate us", "https://x.test/csat/{{1}}")}

	tests := []struct {
		name    string
		desired models.Template
		want    bool
	}{
		{"identical", models.Template{BodyContent: "Rate us please", Buttons: buttons("Rate us", "https://x.test/csat/{{1}}")}, false},
		{"whitespace only", models.Template{BodyContent: "  Rate us please  ", Buttons: buttons(" Rate us ", "https://x.test/csat/{{1}}")}, false},
		{"body changed", models.Template{BodyContent: "New copy", Buttons: buttons("Rate us", "https://x.test/csat/{{1}}")}, true},
		{"button text changed", models.Template{BodyContent: "Rate us please", Buttons: buttons("Give feedback", "https://x.test/csat/{{1}}")}, true},
		{"button url changed", models.Template{BodyContent: "Rate us please", Buttons: buttons("Rate us", "https://new.test/csat/{{1}}")}, true},
		{"button removed", models.Template{BodyContent: "Rate us please", Buttons: json.RawMessage(`[]`)}, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := reservedContentChanged(existing, tc.desired); got != tc.want {
				t.Fatalf("expected %v, got %v", tc.want, got)
			}
		})
	}
}

func TestParseSampleValues(t *testing.T) {
	got := parseSampleValues(json.RawMessage(`{"name":"Ravi","count":2,"flag":true}`))
	if got["name"] != "Ravi" || got["count"] != "2" || got["flag"] != "true" {
		t.Fatalf("unexpected sample values: %+v", got)
	}
	if parseSampleValues(json.RawMessage(`{}`)) != nil {
		t.Fatal("expected nil for an empty object")
	}
	if parseSampleValues(nil) != nil {
		t.Fatal("expected nil for absent sample values")
	}
	if parseSampleValues(json.RawMessage(`not json`)) != nil {
		t.Fatal("expected nil for malformed sample values")
	}
}

func TestCSATTemplateName(t *testing.T) {
	if got := models.CSATTemplateName(15); got != "libredesk_csat_15" {
		t.Fatalf("unexpected reserved name %q", got)
	}
	if !strings.HasPrefix(models.CSATTemplateName(15), models.CSATTemplateNamePrefix) {
		t.Fatal("reserved names must carry the reserved prefix the delete guard checks")
	}
}

func componentByType(sub whatsapp.TemplateSubmission, typ string) (whatsapp.TemplateComponent, bool) {
	for _, c := range sub.Components {
		if c.Type == typ {
			return c, true
		}
	}
	return whatsapp.TemplateComponent{}, false
}
