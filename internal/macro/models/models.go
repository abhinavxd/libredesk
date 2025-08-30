package models

import (
	"encoding/json"
	"time"

	"github.com/lib/pq"
)

type Macro struct {
	ID             int             `db:"id" json:"id"`
	CreatedAt      time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt      time.Time       `db:"updated_at" json:"updated_at"`
	Name           string          `db:"name" json:"name"`
	Actions        json.RawMessage `db:"actions" json:"actions"`
	Visibility     string          `db:"visibility" json:"visibility"`
	VisibleWhen    pq.StringArray  `db:"visible_when" json:"visible_when"`
	MessageContent string          `db:"message_content" json:"message_content"`
	UserID         *int            `db:"user_id" json:"user_id,string"`
	TeamID         *int            `db:"team_id" json:"team_id,string"`
	UsageCount     int             `db:"usage_count" json:"usage_count"`
}
