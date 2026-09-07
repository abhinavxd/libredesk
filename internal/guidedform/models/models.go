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

// Step answer types.
const (
	StepTypeText   = "text"
	StepTypeChoice = "choice"
	StepTypeEmail  = "email"
	StepTypePhone  = "phone"
	StepTypeNumber = "number"
)

// Branch matches an answer against a pattern and, if it matches, sends the flow to NextStepID.
// Pattern is a case-insensitive regular expression matched against the trimmed answer text.
// The first matching branch (in slice order) wins.
type Branch struct {
	Pattern    string `json:"pattern"`
	NextStepID string `json:"next_step_id"`
}

// Step is one question in a guided form. When the visitor's answer doesn't match any Branch:
//   - if DefaultNextStepID is set, the flow jumps there;
//   - else if EndsForm is true, the flow ends here;
//   - else the flow falls through to the next step in the form's Steps order (so a plain
//     linear form needs no branch/default configuration at all), ending only if this is the
//     last step.
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
	// EndsForm explicitly ends the flow here when no branch matches, instead of falling
	// through to the next step in order. Ignored when DefaultNextStepID is set.
	EndsForm bool `json:"ends_form,omitempty"`
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
	// AllowSkipToHuman controls whether a visitor sees a "talk to a human" escape hatch out of
	// this form's flow (widget) and whether it's honored server-side if they use it anyway.
	AllowSkipToHuman bool `json:"allow_skip_to_human" db:"allow_skip_to_human"`
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

// NextStepInOrder returns the step immediately following the one with the given id in the
// form's Steps slice, or ok=false if id is the last step (or not found).
func (f Form) NextStepInOrder(id string) (Step, bool) {
	for i, s := range f.Steps {
		if s.ID == id && i+1 < len(f.Steps) {
			return f.Steps[i+1], true
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
