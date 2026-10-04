package models

import (
	"time"

	mmodels "github.com/jakedolan443/fernmail/internal/media/models"
	"github.com/jakedolan443/fernmail/internal/resourcepolicy"
	"github.com/lib/pq"
	"github.com/volatiletech/null/v9"
)

const (
	ReviewKindReply = "reply"
	ReviewKindNew   = "new"

	ReviewStatusPending   = "pending"
	ReviewStatusApproved  = "approved"
	ReviewStatusDenied    = "denied"
	ReviewStatusWithdrawn = "withdrawn"
)

// Review is a Contributor's email held until an Admin or Agent decides on it.
// It is not a message: nothing is sent, searched or counted as unread until
// approval queues a real message.
type Review struct {
	ID                  int                `db:"id" json:"-"`
	UUID                string             `db:"uuid" json:"uuid"`
	CreatedAt           time.Time          `db:"created_at" json:"created_at"`
	UpdatedAt           time.Time          `db:"updated_at" json:"updated_at"`
	Kind                string             `db:"kind" json:"kind"`
	Status              string             `db:"status" json:"status"`
	AuthorID            int                `db:"author_id" json:"author_id"`
	AuthorName          string             `db:"author_name" json:"author_name"`
	AddressID           int                `db:"address_id" json:"address_id"`
	Address             string             `db:"address" json:"address"`
	AddressName         string             `db:"address_name" json:"address_name"`
	InboxID             int                `db:"inbox_id" json:"-"`
	ConversationID      null.Int           `db:"conversation_id" json:"-"`
	ConversationUUID    null.String        `db:"conversation_uuid" json:"conversation_uuid"`
	ConversationSubject string             `db:"conversation_subject" json:"conversation_subject"`
	Subject             string             `db:"subject" json:"subject"`
	Content             string             `db:"content" json:"content"`
	Preview             string             `db:"-" json:"preview"`
	To                  pq.StringArray     `db:"to" json:"to"`
	CC                  pq.StringArray     `db:"cc" json:"cc"`
	BCC                 pq.StringArray     `db:"bcc" json:"bcc"`
	ReviewerID          null.Int           `db:"reviewer_id" json:"reviewer_id"`
	ReviewerName        null.String        `db:"reviewer_name" json:"reviewer_name"`
	ReviewedAt          null.Time          `db:"reviewed_at" json:"reviewed_at"`
	DecisionNote        string             `db:"decision_note" json:"decision_note"`
	DismissedAt         null.Time          `db:"dismissed_at" json:"dismissed_at"`
	MessageUUID         null.String        `db:"message_uuid" json:"message_uuid"`
	Attachments         []ReviewAttachment `db:"-" json:"attachments"`
	// ActivationKeys is set when the submission carries key placeholders.
	ActivationKeys *ReviewKeySummary `db:"-" json:"activation_keys,omitempty"`
	// Display is the sanitized HTML to show, prepared per response.
	Display *resourcepolicy.Display `db:"-" json:"display,omitempty"`
}

// ReviewKeySummary tells a reviewer how many activation keys approving will send.
type ReviewKeySummary struct {
	Count   int    `json:"count"`
	AppID   int    `json:"app_id"`
	AppName string `json:"app_name"`
}

// ReviewAttachment is an upload held by a pending or returned submission.
type ReviewAttachment struct {
	ID          int    `db:"id" json:"id"`
	UUID        string `db:"uuid" json:"uuid"`
	Filename    string `db:"filename" json:"filename"`
	ContentType string `db:"content_type" json:"content_type"`
	Size        int    `db:"size" json:"size"`
	Inline      bool   `db:"inline" json:"inline"`
	URL         string `db:"-" json:"url"`
}

// ReviewCounts drives the blue sidebar badge.
type ReviewCounts struct {
	// Pending is the reviewer's queue, or a Contributor's own waiting submissions.
	Pending int `db:"pending" json:"pending"`
	// Returned counts a Contributor's denied or withdrawn new emails awaiting an edit.
	Returned int `db:"returned" json:"returned"`
}

// ReviewInput is a submission as composed. ConversationID is zero for a new email.
type ReviewInput struct {
	AddressID      int
	ConversationID int
	Subject        string
	Content        string
	To, CC, BCC    []string
	Media          []mmodels.Media
}
