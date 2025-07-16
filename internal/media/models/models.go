package models

import (
	"encoding/json"
	"time"

	"github.com/volatiletech/null/v9"
)

const (
	ModelMessages = "messages"
	ModelUser     = "users"

	DispositionInline = "inline"
)

// Media represents an uploaded object.
type Media struct {
	ID          int             `db:"id" json:"id"`
	CreatedAt   time.Time       `db:"created_at" json:"created_at"`
	UUID        string          `db:"uuid" json:"uuid"`
	Filename    string          `db:"filename" json:"filename"`
	ContentType string          `db:"content_type" json:"content_type"`
	Model       null.String     `db:"model_type" json:"-"`
	ModelID     null.Int        `db:"model_id" json:"-"`
	Size        int             `db:"size" json:"size"`
	Store       string          `db:"store" json:"store"`
	Disposition null.String     `db:"disposition" json:"disposition"`
	Meta        json.RawMessage `db:"meta" json:"meta"`
	URL         string          `json:"url"`
	ContentID   string          `json:"-"`
	Content     []byte          `json:"-"`
}
