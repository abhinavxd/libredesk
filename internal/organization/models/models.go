package models

import (
	"encoding/json"
	"time"

	"github.com/lib/pq"
	"github.com/volatiletech/null/v9"
)

type Organization struct {
	ID                int             `db:"id" json:"id"`
	CreatedAt         time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt         time.Time       `db:"updated_at" json:"updated_at"`
	Name              string          `db:"name" json:"name"`
	Domains           pq.StringArray  `db:"domains" json:"domains"`
	Notes             string          `db:"notes" json:"notes"`
	ExternalID        null.String     `db:"external_id" json:"external_id"`
	CustomAttributes  json.RawMessage `db:"custom_attributes" json:"custom_attributes"`
}
