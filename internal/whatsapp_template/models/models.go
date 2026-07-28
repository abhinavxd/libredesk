package models

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/volatiletech/null/v9"
)

// Reserved per-inbox CSAT template, auto-provisioned and hidden from the agent picker.
const CSATTemplateNamePrefix = "libredesk_csat_"

func CSATTemplateName(inboxID int) string {
	return fmt.Sprintf("%s%d", CSATTemplateNamePrefix, inboxID)
}

// Status values mirror Meta's template lifecycle.
const (
	StatusPending              = "PENDING"
	StatusApproved             = "APPROVED"
	StatusRejected             = "REJECTED"
	StatusPendingDeletion      = "PENDING_DELETION"
	StatusDisabled             = "DISABLED"
	StatusPaused               = "PAUSED"
	StatusInAppeal             = "IN_APPEAL"
	StatusPendingQualityReview = "PENDING_QUALITY_REVIEW"
)

// Category values supported on Meta.
const (
	CategoryMarketing      = "MARKETING"
	CategoryUtility        = "UTILITY"
	CategoryAuthentication = "AUTHENTICATION"
)

// Template is a stored WhatsApp template, mirroring Meta's record plus a
// libredesk-side scoping (inbox_id) and submission state.
type Template struct {
	ID              int             `db:"id" json:"id"`
	CreatedAt       time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt       time.Time       `db:"updated_at" json:"updated_at"`
	InboxID         int             `db:"inbox_id" json:"inbox_id"`
	MetaTemplateID  null.String     `db:"meta_template_id" json:"meta_template_id"`
	Name            string          `db:"name" json:"name"`
	Language        string          `db:"language" json:"language"`
	Category        string          `db:"category" json:"category"`
	Status          string          `db:"status" json:"status"`
	HeaderType      null.String     `db:"header_type" json:"header_type"`
	HeaderContent   null.String     `db:"header_content" json:"header_content"`
	BodyContent     string          `db:"body_content" json:"body_content"`
	FooterContent   null.String     `db:"footer_content" json:"footer_content"`
	Buttons         json.RawMessage `db:"buttons" json:"buttons"`
	SampleValues    json.RawMessage `db:"sample_values" json:"sample_values"`
	RejectionReason null.String     `db:"rejection_reason" json:"rejection_reason"`
}
