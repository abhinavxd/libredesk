// package models has the models for the customer satisfaction survey responses.
package models

import (
	"encoding/json"
	"time"

	"github.com/volatiletech/null/v9"
)

// CSATResponse represents a customer satisfaction survey response.
type CSATResponse struct {
	ID                int             `db:"id" json:"id"`
	CreatedAt         time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt         time.Time       `db:"updated_at" json:"updated_at"`
	UUID              string          `db:"uuid" json:"uuid"`
	ConversationID    int             `db:"conversation_id" json:"conversation_id"`
	Rating            int             `db:"rating" json:"rating"`
	Feedback          null.String     `db:"feedback" json:"feedback"`
	Meta              json.RawMessage `db:"meta" json:"meta"`
	ResponseTimestamp null.Time       `db:"response_timestamp" json:"response_timestamp"`
}

func (c CSATResponse) FeedbackPending() bool {
	var meta struct {
		Pending bool `json:"feedback_pending"`
	}
	_ = json.Unmarshal(c.Meta, &meta)
	return c.ResponseTimestamp.Valid && meta.Pending
}
