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
	StepTypeInfo   = "info"
)

// ContactField values a step's answer can be linked to directly, instead of (or in addition to
// being unavailable alongside) a custom attribute. These are the same core fields the static
// pre-chat form's default fields set.
const (
	ContactFieldName  = "name"
	ContactFieldEmail = "email"
)

// Branch routing actions: an alternative to continuing within the same form via NextStepID.
const (
	BranchActionTeam      = CompleteActionTeam
	BranchActionAssistant = CompleteActionAssistant
	BranchActionForm      = "form"
)

// Branch matches an answer against a pattern and, if it matches, routes the flow onward.
// Pattern is a case-insensitive regular expression matched against the trimmed answer text.
// The first matching branch (in slice order) wins. Exactly one of NextStepID or Action should
// be set: NextStepID continues within this form; Action routes away from it entirely - straight
// to a team or AI assistant, or into a different guided form's flow from its own start step.
type Branch struct {
	Pattern     string   `json:"pattern"`
	NextStepID  string   `json:"next_step_id,omitempty"`
	Action      string   `json:"action,omitempty"`
	TeamID      null.Int `json:"team_id,omitempty"`
	AssistantID null.Int `json:"assistant_id,omitempty"`
	FormID      null.Int `json:"form_id,omitempty"`
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
	// ContactField, when set, saves the answer directly to that core contact field (see the
	// ContactField* constants) instead of a custom attribute. Takes precedence over
	// CustomAttributeID when both are somehow set.
	ContactField string `json:"contact_field,omitempty"`
	Required     bool   `json:"required"`
	Branches          []Branch `json:"branches,omitempty"`
	DefaultNextStepID string   `json:"default_next_step_id,omitempty"`
	// EndsForm explicitly ends the flow here when no branch matches, instead of falling
	// through to the next step in order. Ignored when DefaultNextStepID or DefaultAction is set.
	EndsForm bool `json:"ends_form,omitempty"`
	// DefaultAction, when set, is the "otherwise" equivalent of a Branch's Action: routes away
	// from this form entirely when no branch matches, instead of continuing to
	// DefaultNextStepID. Same values as Branch.Action.
	DefaultAction      string   `json:"default_action,omitempty"`
	DefaultTeamID      null.Int `json:"default_team_id,omitempty"`
	DefaultAssistantID null.Int `json:"default_assistant_id,omitempty"`
	DefaultFormID      null.Int `json:"default_form_id,omitempty"`
}

// Form is a guided, branching pre-chat question flow configured for a live chat inbox.
type Form struct {
	ID                    int             `json:"id" db:"id"`
	CreatedAt             time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt             time.Time       `json:"updated_at" db:"updated_at"`
	UserID                int             `json:"user_id" db:"user_id"`
	Name                  string          `json:"name" db:"name"`
	// DisplayName is the name shown to the visitor as the bot's identity (message author,
	// widget assignee). Falls back to Name when empty, so setting it is optional.
	DisplayName           string          `json:"display_name" db:"display_name"`
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
	// AbandonedTimeoutMinutes, when > 0, auto-resolves a conversation still assigned to this
	// form's bot if its own latest question has gone unanswered for this long - so a visitor
	// who opens chat and never replies doesn't clutter the inbox forever. 0 disables it.
	AbandonedTimeoutMinutes int `json:"abandoned_timeout_minutes" db:"abandoned_timeout_minutes"`
}

// EffectiveDisplayName returns DisplayName, falling back to Name when unset.
func (f Form) EffectiveDisplayName() string {
	if f.DisplayName != "" {
		return f.DisplayName
	}
	return f.Name
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
