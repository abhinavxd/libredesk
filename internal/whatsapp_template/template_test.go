package whatsapp_template

import (
	"encoding/json"
	"testing"

	"github.com/abhinavxd/libredesk/internal/whatsapp_template/models"
)

// The reserved CSAT template's URL button carries a {{1}} placeholder, so its
// submission must ship a button example or Meta rejects it (and buildSubmission
// errors on the missing sample value).
func TestBuildSubmissionCSATButtonExample(t *testing.T) {
	buttons, _ := json.Marshal([]map[string]any{{
		"type":    "URL",
		"text":    "Rate us",
		"url":     "http://localhost:9000/csat/{{1}}",
		"example": []string{"http://localhost:9000/csat/example"},
	}})
	sub, err := buildSubmission(models.Template{
		InboxID:     2,
		Name:        "libredesk_csat_2",
		Language:    "en_US",
		Category:    "UTILITY",
		BodyContent: "Your conversation has been resolved.",
		Buttons:     buttons,
	})
	if err != nil {
		t.Fatalf("buildSubmission errored: %v", err)
	}
	for _, c := range sub.Components {
		if c.Type != "BUTTONS" {
			continue
		}
		if len(c.Buttons) == 0 || len(c.Buttons[0].Example) == 0 {
			t.Fatalf("expected URL button to carry an example, got %+v", c.Buttons)
		}
		return
	}
	t.Fatalf("expected a BUTTONS component, got %+v", sub.Components)
}
