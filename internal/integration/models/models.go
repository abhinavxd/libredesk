package models

import (
	"encoding/json"
	"time"
)

// Integration represents an external integration configuration.
type Integration struct {
	ID        int             `db:"id" json:"id"`
	CreatedAt time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt time.Time       `db:"updated_at" json:"updated_at"`
	Provider  string          `db:"provider" json:"provider"`
	Config    json.RawMessage `db:"config" json:"config"`
	Enabled   bool            `db:"enabled" json:"enabled"`
}
