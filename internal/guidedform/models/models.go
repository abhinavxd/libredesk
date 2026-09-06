// Package models holds the data structures for guided pre-chat forms.
package models

import (
	"encoding/json"
	"time"

	"github.com/volatiletech/null/v9"
)

// Completion actions for a finished guided form.
const (
	CompleteActionTeam      = "team"
	CompleteActionAssistant = "ai_assistant"
	CompleteActionUnassign  = "unassigned"
)

// AppliesTo mirrors the custom attribute "applies_to" values an answer can be saved against.
const (
	AppliesToContact      = "contact"
	AppliesToConversation = "conversation"
)

// Branch matches an answer against a pattern and, if it matches, sends the flow to NextStepID.
// Pattern is a case-insensitive regular expression matched against the trimmed answer text.
// The first matching branch (in slice order) wins.
type Branch struct {
	Pattern    string `json:"pattern"`
	NextStepID string `json:"next_step_id"`
}

// Step is one question in a guided form. When the visitor's answer doesn't match any Branch,
// the flow moves to DefaultNextStepID; a step with no branches and no default is terminal.
type Step struct {
	ID                string   `json:"id"`
	Question          string   `json:"question"`
	Type              string   `json:"type"` // text, choice, email, phone, number
	Options           []string `json:"options,omitempty"`
	SaveAs            string   `json:"save_as,omitempty"`
	CustomAttributeID int      `json:"custom_attribute_id,omitempty"`
	Required          bool     `json:"required"`
	Branches          []Branch `json:"branches,omitempty"`
	DefaultNextStepID string   `json:"default_next_step_id,omitempty"`
}

// IsTerminal reports whether this step ends the flow when nothing else matches.
func (s Step) IsTerminal() bool {
	return len(s.Branches) == 0 && s.DefaultNextStepID == ""
}

// Form is a guided, branching pre-chat question flow configured for a live chat inbox.
type Form struct {
	ID                    int             `json:"id" db:"id"`
	CreatedAt             time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt             time.Time       `json:"updated_at" db:"updated_at"`
	UserID                int             `json:"user_id" db:"user_id"`
	Name                  string          `json:"name" db:"name"`
	InboxID               int             `json:"inbox_id" db:"inbox_id"`
	Enabled               bool            `json:"enabled" db:"enabled"`
	StartStepID           string          `json:"start_step_id" db:"start_step_id"`
	StepsRaw              json.RawMessage `json:"-" db:"steps"`
	Steps                 []Step          `json:"steps" db:"-"`
	OnCompleteAction      string          `json:"on_complete_action" db:"on_complete_action"`
	OnCompleteAssistantID null.Int        `json:"on_complete_assistant_id" db:"on_complete_assistant_id"`
	OnCompleteTeamID      null.Int        `json:"on_complete_team_id" db:"on_complete_team_id"`
	CompletionMessage     string          `json:"completion_message" db:"completion_message"`
}

// UnmarshalSteps decodes StepsRaw (as loaded from the DB) into Steps.
func (f *Form) UnmarshalSteps() error {
	if len(f.StepsRaw) == 0 {
		f.Steps = nil
		return nil
	}
	return json.Unmarshal(f.StepsRaw, &f.Steps)
}

// MarshalSteps encodes Steps into StepsRaw for persistence.
func (f *Form) MarshalSteps() error {
	b, err := json.Marshal(f.Steps)
	if err != nil {
		return err
	}
	f.StepsRaw = b
	return nil
}

// StepByID returns the step with the given id, or ok=false if not found.
func (f Form) StepByID(id string) (Step, bool) {
	for _, s := range f.Steps {
		if s.ID == id {
			return s, true
		}
	}
	return Step{}, false
}

// Progress tracks where a conversation is within a guided form's flow. It is persisted as a
// reserved key in the conversation's custom attributes JSONB so no extra table is needed.
type Progress struct {
	FormID  int            `json:"form_id"`
	StepID  string         `json:"step_id"`
	Answers map[string]any `json:"answers"`
	// LastMessageID is the id of the inbound message last used to advance the flow, so a
	// duplicate trigger for the same message doesn't advance it twice.
	LastMessageID int `json:"last_message_id"`
}
