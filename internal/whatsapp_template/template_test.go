package whatsapp_template

import (
	"encoding/json"
	"testing"

	"github.com/abhinavxd/libredesk/internal/whatsapp_template/models"
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
