package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/lib/pq"
	"github.com/volatiletech/null/v9"
)

// SLAPolicy represents a service level agreement policy definition
type SLAPolicy struct {
	ID                int              `db:"id" json:"id"`
	CreatedAt         time.Time        `db:"created_at" json:"created_at"`
	UpdatedAt         time.Time        `db:"updated_at" json:"updated_at"`
	Name              string           `db:"name" json:"name"`
	Description       string           `db:"description" json:"description,omitempty"`
	FirstResponseTime string           `db:"first_response_time" json:"first_response_time,omitempty"`
	EveryResponseTime string           `db:"every_response_time" json:"every_response_time,omitempty"`
	NextResponseTime  string           `db:"next_response_time" json:"next_response_time,omitempty"`
	ResolutionTime    string           `db:"resolution_time" json:"resolution_time,omitempty"`
	Notifications     SlaNotifications `db:"notifications" json:"notifications,omitempty"`
}

type SlaNotifications []SlaNotification

// Value implements the driver.Valuer interface.
func (sn SlaNotifications) Value() (driver.Value, error) {
	return json.Marshal(sn)
}

// Scan implements the sql.Scanner interface.
func (sn *SlaNotifications) Scan(src any) error {
	var data []byte

	switch v := src.(type) {
	case string:
		data = []byte(v)
	case []byte:
		data = v
	default:
		return fmt.Errorf("unsupported type: %T", src)
	}
	return json.Unmarshal(data, sn)
}

// SlaNotification represents the notification settings for an SLA policy
type SlaNotification struct {
	Type          string   `db:"type" json:"type"`
	Recipients    []string `db:"recipients" json:"recipients"`
	TimeDelay     string   `db:"time_delay" json:"time_delay"`
	TimeDelayType string   `db:"time_delay_type" json:"time_delay_type"`
}

// ScheduledSLANotification represents a scheduled SLA notification
type ScheduledSLANotification struct {
	ID               int            `db:"id" json:"id"`
	CreatedAt        time.Time      `db:"created_at" json:"created_at"`
	UpdatedAt        time.Time      `db:"updated_at" json:"updated_at"`
	SlaEventID       null.Int       `db:"sla_event_id" json:"sla_event_id"`
	AppliedSLAID     int            `db:"applied_sla_id" json:"applied_sla_id"`
	Metric           string         `db:"metric" json:"metric"`
	NotificationType string         `db:"notification_type" json:"notification_type"`
	Recipients       pq.StringArray `db:"recipients" json:"recipients"`
	SendAt           time.Time      `db:"send_at" json:"send_at"`
	ProcessedAt      null.Time      `db:"processed_at" json:"processed_at,omitempty"`
}

// AppliedSLA represents an SLA policy applied to a conversation
type AppliedSLA struct {
	ID                      int       `db:"id"`
	CreatedAt               time.Time `db:"created_at"`
	Status                  string    `db:"status"`
	ConversationID          int       `db:"conversation_id"`
	SLAPolicyID             int       `db:"sla_policy_id"`
	FirstResponseDeadlineAt time.Time `db:"first_response_deadline_at"`
	ResolutionDeadlineAt    time.Time `db:"resolution_deadline_at"`
	FirstResponseBreachedAt null.Time `db:"first_response_breached_at"`
	ResolutionBreachedAt    null.Time `db:"resolution_breached_at"`
	FirstResponseMetAt      null.Time `db:"first_response_met_at"`
	ResolutionMetAt         null.Time `db:"resolution_met_at"`

	// Conversation fields.
	ConversationFirstResponseAt null.Time `db:"conversation_first_response_at"`
	ConversationResolvedAt      null.Time `db:"conversation_resolved_at"`
	ConversationUUID            string    `db:"conversation_uuid"`
	ConversationReferenceNumber string    `db:"conversation_reference_number"`
	ConversationSubject         string    `db:"conversation_subject"`
	ConversationAssignedUserID  null.Int  `db:"conversation_assigned_user_id"`
	ConversationStatus          string    `db:"conversation_status"`
}

type SLAEvent struct {
	ID           int       `db:"id"`
	CreatedAt    time.Time `db:"created_at"`
	UpdatedAt    time.Time `db:"updated_at"`
	AppliedSLAID int       `db:"applied_sla_id"`
	SlaPolicyID  int       `db:"sla_policy_id"`
	Type         string    `db:"type"`
	DeadlineAt   time.Time `db:"deadline_at"`
	MetAt        null.Time `db:"met_at"`
	BreachedAt   null.Time `db:"breached_at"`
}
