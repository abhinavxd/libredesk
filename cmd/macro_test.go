package main

import (
	"encoding/json"
	"testing"

	"github.com/abhinavxd/libredesk/internal/macro/models"
	"github.com/abhinavxd/libredesk/internal/testutil"
)

// Both list endpoints marshal these structs directly, their JSON shape is the API contract.
func TestMacroJSONShapes(t *testing.T) {
	full := testutil.JSONKeys(t, models.Macro{Actions: json.RawMessage(`[]`)})
	for _, key := range []string{"id", "name", "actions", "visibility", "visible_when", "message_content", "user_id", "team_id", "usage_count", "created_at", "updated_at"} {
		if !full[key] {
			t.Errorf("Macro JSON is missing %q", key)
		}
	}
	if full["has_message_content"] {
		t.Error("Macro JSON must not have has_message_content")
	}

	compact := testutil.JSONKeys(t, models.MacroCompact{Actions: json.RawMessage(`[]`)})
	for _, key := range []string{"id", "name", "actions", "visibility", "visible_when", "has_message_content", "user_id", "team_id", "usage_count", "created_at", "updated_at"} {
		if !compact[key] {
			t.Errorf("MacroCompact JSON is missing %q", key)
		}
	}
	if compact["message_content"] {
		t.Error("MacroCompact JSON must not have message_content")
	}
}

func TestDecorateMacroActions(t *testing.T) {
	app := newValidatorTestApp(t)

	out, err := decorateMacroActions(app, json.RawMessage(`[{"type":"add_tags","value":["urgent"]}]`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var actions []map[string]any
	if err := json.Unmarshal(out, &actions); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if len(actions) != 1 {
		t.Fatalf("got %d actions, want 1", len(actions))
	}
	if _, ok := actions[0]["display_value"]; !ok {
		t.Error("decorated action is missing display_value")
	}

	if _, err := decorateMacroActions(app, json.RawMessage(`{not json`)); err == nil {
		t.Error("expected an error for malformed actions JSON")
	}
}
