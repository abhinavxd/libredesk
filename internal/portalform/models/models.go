package models

import (
	"encoding/json"
	"time"
)

const (
	TargetAttribute = "attribute"
	TargetMessage   = "message"

	FieldTypeText     = "text"
	FieldTypeTextarea = "textarea"
	FieldTypeSelect   = "select"
	FieldTypeCheckbox = "checkbox"
	FieldTypeNumber   = "number"
	FieldTypeDate     = "date"
	FieldTypeEmail    = "email"
	FieldTypeLink     = "link"
)

type Field struct {
	Key          string   `json:"key"`
	Label        string   `json:"label"`
	Type         string   `json:"type"`
	Required     bool     `json:"required"`
	Placeholder  string   `json:"placeholder,omitempty"`
	Options      []string `json:"options,omitempty"`
	Target       string   `json:"target"`
	AttributeKey string   `json:"attribute_key,omitempty"`
}

type Form struct {
	ID         int             `db:"id" json:"id"`
	CreatedAt  time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt  time.Time       `db:"updated_at" json:"updated_at"`
	Name       string          `db:"name" json:"name"`
	AskSubject bool            `db:"ask_subject" json:"ask_subject"`
	FieldsJSON json.RawMessage `db:"fields" json:"-"`
	Fields     []Field         `db:"-" json:"fields"`
}
