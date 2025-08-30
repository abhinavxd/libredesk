package models

import (
	"encoding/json"
	"time"

	"github.com/volatiletech/null/v9"
)

const (
	// TODO: pick these table names from their respective package/models/models.go
	ModelMessages = "messages"
	ModelUser     = "users"

	DispositionInline = "inline"
)

// Media represents an uploaded object in DB and storage backend.
type Media struct {
	ID          int             `db:"id" json:"id"`
	CreatedAt   time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time       `db:"updated_at" json:"updated_at"`
	UUID        string          `db:"uuid" json:"uuid"`
	Store       string          `db:"store" json:"store"`
	Filename    string          `db:"filename" json:"filename"`
	ContentType string          `db:"content_type" json:"content_type"`
	ContentID   string          `db:"content_id" json:"content_id"`
	ModelID     null.Int        `db:"model_id" json:"model_id"`
	Model       null.String     `db:"model_type" json:"model_type"`
	Disposition null.String     `db:"disposition" json:"disposition"`
	Size        int             `db:"size" json:"size"`
	Meta        json.RawMessage `db:"meta" json:"meta"`

	// Pseudo fields
	URL     string `json:"url"`
	Content []byte `json:"-"`
}
