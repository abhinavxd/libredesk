// package models has the models for the customer satisfaction survey responses.
package models

import (
	"time"

	"github.com/volatiletech/null/v9"
)

// CSATResponse represents a customer satisfaction survey response.
type CSATResponse struct {
	ID                int         `db:"id"`
	CreatedAt         time.Time   `db:"created_at"`
	UpdatedAt         time.Time   `db:"updated_at"`
	UUID              string      `db:"uuid"`
	ConversationID    int         `db:"conversation_id"`
	Rating            int         `db:"rating"`
	Feedback          null.String `db:"feedback"`
	ResponseTimestamp null.Time   `db:"response_timestamp"`
}
