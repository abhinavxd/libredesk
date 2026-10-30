package models

import (
	"encoding/json"
	"testing"

	"github.com/lib/pq"
	"github.com/volatiletech/null/v9"
)

func TestSupportsTextContent(t *testing.T) {
	for _, tc := range []struct {
		name     string
		types    pq.StringArray
		header   string
		category string
		buttons  string
		want     bool
	}{
		{"body", pq.StringArray{"BODY"}, "", "UTILITY", `[]`, true},
		{"text header", pq.StringArray{"HEADER", "BODY"}, "TEXT", "MARKETING", `[]`, true},
		{"repeated supported types", pq.StringArray{"BODY", "BODY"}, "", "UTILITY", `[]`, true},
		{"carousel", pq.StringArray{"BODY", "CAROUSEL"}, "", "MARKETING", `[]`, false},
		{"unknown component", pq.StringArray{"BODY", "FUTURE_COMPONENT"}, "", "UTILITY", `[]`, false},
		{"legacy unknown", nil, "", "UTILITY", `[]`, false},
		{"empty", pq.StringArray{}, "", "UTILITY", `[]`, false},
		{"media", pq.StringArray{"HEADER", "BODY"}, "IMAGE", "UTILITY", `[]`, false},
		{"authentication", pq.StringArray{"BODY", "BUTTONS"}, "", "AUTHENTICATION", `[]`, false},
		{"flow button", pq.StringArray{"BODY", "BUTTONS"}, "", "UTILITY", `[{"type":"FLOW"}]`, false},
		{"url button", pq.StringArray{"BODY", "BUTTONS"}, "", "UTILITY", `[{"type":"URL"}]`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			value := Template{ComponentTypes: tc.types, HeaderType: null.StringFrom(tc.header), Category: tc.category, Buttons: json.RawMessage(tc.buttons)}
			if got := value.SupportsTextContent(); got != tc.want {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}
