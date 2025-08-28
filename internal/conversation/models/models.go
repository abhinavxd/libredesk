package models

import (
	"encoding/json"
	"net/textproto"
	"time"

	"github.com/abhinavxd/libredesk/internal/attachment"
	mmodels "github.com/abhinavxd/libredesk/internal/media/models"
	umodels "github.com/abhinavxd/libredesk/internal/user/models"
	"github.com/lib/pq"
	"github.com/volatiletech/null/v9"
)

var (
	StatusOpen     = "Open"
	StatusReplied  = "Replied"
	StatusResolved = "Resolved"
	StatusClosed   = "Closed"
	StatusSnoozed  = "Snoozed"

	AssigneeTypeTeam = "team"
	AssigneeTypeUser = "user"

	AllConversations            = "all"
	AssignedConversations       = "assigned"
	UnassignedConversations     = "unassigned"
	TeamUnassignedConversations = "team_unassigned"

	MessageIncoming = "incoming"
	MessageOutgoing = "outgoing"
	MessageActivity = "activity"

	SenderTypeAgent   = "agent"
	SenderTypeContact = "contact"

	MessageStatusPending  = "pending"
	MessageStatusSent     = "sent"
	MessageStatusFailed   = "failed"
	MessageStatusReceived = "received"

	ActivityStatusChange       = "status_change"
	ActivityPriorityChange     = "priority_change"
	ActivityAssignedUserChange = "assigned_user_change"
	ActivityAssignedTeamChange = "assigned_team_change"
	ActivitySelfAssign         = "self_assign"
	ActivityTagAdded           = "tag_added"
	ActivityTagRemoved         = "tag_removed"
	ActivitySLASet             = "sla_set"

	ContentTypeText = "text"
	ContentTypeHTML = "html"
)

// ConversationListItem represents a conversation in list views
type ConversationListItem struct {
	Total              int                     `db:"total" json:"-"`
	ID                 int                     `db:"id" json:"id"`
	CreatedAt          time.Time               `db:"created_at" json:"created_at"`
	UpdatedAt          time.Time               `db:"updated_at" json:"updated_at"`
	UUID               string                  `db:"uuid" json:"uuid"`
	WaitingSince       null.Time               `db:"waiting_since" json:"waiting_since"`
	AssigneeLastSeenAt null.Time               `db:"assignee_last_seen_at" json:"assignee_last_seen_at"`
	Contact            ConversationListContact `db:"contact" json:"contact"`
	InboxChannel       string                  `db:"inbox_channel" json:"inbox_channel"`
	InboxName          string                  `db:"inbox_name" json:"inbox_name"`
	SLAPolicyID        null.Int                `db:"sla_policy_id" json:"sla_policy_id"`
	FirstReplyAt       null.Time               `db:"first_reply_at" json:"first_reply_at"`
	LastReplyAt        null.Time               `db:"last_reply_at" json:"last_reply_at"`
	ResolvedAt         null.Time               `db:"resolved_at" json:"resolved_at"`
	Subject            null.String             `db:"subject" json:"subject"`
	LastMessage        null.String             `db:"last_message" json:"last_message"`
	LastMessageAt      null.Time               `db:"last_message_at" json:"last_message_at"`
	LastMessageSender  null.String             `db:"last_message_sender" json:"last_message_sender"`
	NextSLADeadlineAt  null.Time               `db:"next_sla_deadline_at" json:"next_sla_deadline_at"`
	PriorityID         null.Int                `db:"priority_id" json:"priority_id"`
	UnreadMessageCount int                     `db:"unread_message_count" json:"unread_message_count"`
	Status             null.String             `db:"status" json:"status"`
	Priority           null.String             `db:"priority" json:"priority"`
	FirstResponseDueAt null.Time               `db:"first_response_deadline_at" json:"first_response_deadline_at"`
	ResolutionDueAt    null.Time               `db:"resolution_deadline_at" json:"resolution_deadline_at"`
	AppliedSLAID       null.Int                `db:"applied_sla_id" json:"applied_sla_id"`
	NextResponseDueAt  null.Time               `db:"next_response_deadline_at" json:"next_response_deadline_at"`
	NextResponseMetAt  null.Time               `db:"next_response_met_at" json:"next_response_met_at"`
}

// ConversationListContact represents contact info in conversation list views
type ConversationListContact struct {
	CreatedAt time.Time   `db:"created_at" json:"created_at"`
	UpdatedAt time.Time   `db:"updated_at" json:"updated_at"`
	FirstName string      `db:"first_name" json:"first_name"`
	LastName  string      `db:"last_name" json:"last_name"`
	AvatarURL null.String `db:"avatar_url" json:"avatar_url"`
}

type Conversation struct {
	ID                    int                    `db:"id" json:"id"`
	CreatedAt             time.Time              `db:"created_at" json:"created_at"`
	UpdatedAt             time.Time              `db:"updated_at" json:"updated_at"`
	UUID                  string                 `db:"uuid" json:"uuid"`
	ContactID             int                    `db:"contact_id" json:"contact_id"`
	InboxID               int                    `db:"inbox_id" json:"inbox_id"`
	ClosedAt              null.Time              `db:"closed_at" json:"closed_at"`
	ResolvedAt            null.Time              `db:"resolved_at" json:"resolved_at"`
	ReferenceNumber       string                 `db:"reference_number" json:"reference_number"`
	Priority              null.String            `db:"priority" json:"priority"`
	PriorityID            null.Int               `db:"priority_id" json:"priority_id"`
	Status                null.String            `db:"status" json:"status"`
	StatusID              null.Int               `db:"status_id" json:"status_id"`
	FirstReplyAt          null.Time              `db:"first_reply_at" json:"first_reply_at"`
	LastReplyAt           null.Time              `db:"last_reply_at" json:"last_reply_at"`
	AssignedUserID        null.Int               `db:"assigned_user_id" json:"assigned_user_id"`
	AssignedTeamID        null.Int               `db:"assigned_team_id" json:"assigned_team_id"`
	AssigneeLastSeenAt    null.Time              `db:"assignee_last_seen_at" json:"assignee_last_seen_at"`
	WaitingSince          null.Time              `db:"waiting_since" json:"waiting_since"`
	Subject               null.String            `db:"subject" json:"subject"`
	InboxMail             string                 `db:"inbox_mail" json:"inbox_mail"`
	InboxName             string                 `db:"inbox_name" json:"inbox_name"`
	InboxChannel          string                 `db:"inbox_channel" json:"inbox_channel"`
	Tags                  null.JSON              `db:"tags" json:"tags"`
	Meta                  pq.StringArray         `db:"meta" json:"meta"`
	CustomAttributes      json.RawMessage        `db:"custom_attributes" json:"custom_attributes"`
	LastMessageAt         null.Time              `db:"last_message_at" json:"last_message_at"`
	LastMessage           null.String            `db:"last_message" json:"last_message"`
	LastMessageSender     null.String            `db:"last_message_sender" json:"last_message_sender"`
	Contact               ConversationContact    `db:"contact" json:"contact"`
	SLAPolicyID           null.Int               `db:"sla_policy_id" json:"sla_policy_id"`
	SlaPolicyName         null.String            `db:"sla_policy_name" json:"sla_policy_name"`
	AppliedSLAID          null.Int               `db:"applied_sla_id" json:"applied_sla_id"`
	FirstResponseDueAt    null.Time              `db:"first_response_deadline_at" json:"first_response_deadline_at"`
	ResolutionDueAt       null.Time              `db:"resolution_deadline_at" json:"resolution_deadline_at"`
	NextResponseDueAt     null.Time              `db:"next_response_deadline_at" json:"next_response_deadline_at"`
	NextResponseMetAt     null.Time              `db:"next_response_met_at" json:"next_response_met_at"`
	PreviousConversations []PreviousConversation `db:"-" json:"previous_conversations"`
}

type ConversationContact struct {
	ID                     int             `db:"id" json:"id"`
	CreatedAt              time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt              time.Time       `db:"updated_at" json:"updated_at"`
	FirstName              string          `db:"first_name" json:"first_name"`
	LastName               string          `db:"last_name" json:"last_name"`
	Email                  null.String     `db:"email" json:"email"`
	Type                   string          `db:"type" json:"type"`
	AvailabilityStatus     string          `db:"availability_status" json:"availability_status"`
	AvatarURL              null.String     `db:"avatar_url" json:"avatar_url"`
	PhoneNumber            null.String     `db:"phone_number" json:"phone_number"`
	PhoneNumberCallingCode null.String     `db:"phone_number_calling_code" json:"phone_number_calling_code"`
	CustomAttributes       json.RawMessage `db:"custom_attributes" json:"custom_attributes"`
	Enabled                bool            `db:"enabled" json:"enabled"`
	LastActiveAt           null.Time       `db:"last_active_at" json:"last_active_at"`
	LastLoginAt            null.Time       `db:"last_login_at" json:"last_login_at"`
}

func (c *ConversationContact) FullName() string {
	return c.FirstName + " " + c.LastName
}

type PreviousConversation struct {
	ID            int                         `db:"id" json:"id"`
	CreatedAt     time.Time                   `db:"created_at" json:"created_at"`
	UpdatedAt     time.Time                   `db:"updated_at" json:"updated_at"`
	UUID          string                      `db:"uuid" json:"uuid"`
	Contact       PreviousConversationContact `db:"contact" json:"contact"`
	LastMessage   null.String                 `db:"last_message" json:"last_message"`
	LastMessageAt null.Time                   `db:"last_message_at" json:"last_message_at"`
}

type PreviousConversationContact struct {
	FirstName string      `db:"first_name" json:"first_name"`
	LastName  string      `db:"last_name" json:"last_name"`
	AvatarURL null.String `db:"avatar_url" json:"avatar_url"`
}

type ConversationParticipant struct {
	ID        string      `db:"id" json:"id"`
	FirstName string      `db:"first_name" json:"first_name"`
	LastName  string      `db:"last_name" json:"last_name"`
	AvatarURL null.String `db:"avatar_url" json:"avatar_url"`
}

type ConversationCounts struct {
	TotalAssigned         int `db:"total_assigned" json:"total_assigned"`
	UnresolvedCount       int `db:"unresolved_count" json:"unresolved_count"`
	AwaitingResponseCount int `db:"awaiting_response_count" json:"awaiting_response_count"`
	CreatedTodayCount     int `db:"created_today_count" json:"created_today_count"`
}

type NewConversationsStats struct {
	Date             string `db:"date" json:"date"`
	NewConversations int    `db:"new_conversations" json:"new_conversations"`
}

// Message represents a message in a conversation
type Message struct {
	Total            int                    `db:"total" json:"-"`
	ID               int                    `db:"id" json:"id"`
	CreatedAt        time.Time              `db:"created_at" json:"created_at"`
	UpdatedAt        time.Time              `db:"updated_at" json:"updated_at"`
	UUID             string                 `db:"uuid" json:"uuid"`
	Type             string                 `db:"type" json:"type"`
	Status           string                 `db:"status" json:"status"`
	ConversationID   int                    `db:"conversation_id" json:"conversation_id"`
	ConversationUUID string                 `db:"conversation_uuid" json:"conversation_uuid"`
	Content          string                 `db:"content" json:"content"`
	TextContent      string                 `db:"text_content" json:"text_content"`
	ContentType      string                 `db:"content_type" json:"content_type"`
	Private          bool                   `db:"private" json:"private"`
	SourceID         null.String            `db:"source_id" json:"-"`
	SenderID         int                    `db:"sender_id" json:"sender_id"`
	SenderType       string                 `db:"sender_type" json:"sender_type"`
	InboxID          int                    `db:"inbox_id" json:"-"`
	Meta             json.RawMessage        `db:"meta" json:"meta"`
	Attachments      attachment.Attachments `db:"attachments" json:"attachments"`
	From             string                 `db:"from"  json:"-"`
	Subject          string                 `db:"subject" json:"-"`
	Channel          string                 `db:"channel" json:"-"`
	To               pq.StringArray         `db:"to"  json:"-"`
	CC               pq.StringArray         `db:"cc" json:"-"`
	BCC              pq.StringArray         `db:"bcc" json:"-"`
	References       []string               `json:"-"`
	InReplyTo        string                 `json:"-"`
	Headers          textproto.MIMEHeader   `json:"-"`
	AltContent       string                 `json:"-"`
	Media            []mmodels.Media        `json:"-"`
	IsCSAT           bool                   `json:"-"`
}

// CensorCSATContent redacts the content of a CSAT message to prevent leaking the CSAT survey public link.
func (m *Message) CensorCSATContent() {
	var meta map[string]interface{}
	if err := json.Unmarshal([]byte(m.Meta), &meta); err != nil {
		return
	}
	if isCsat, _ := meta["is_csat"].(bool); isCsat {
		m.Content = "Please rate your experience with us"
		m.TextContent = m.Content
	}
}

// HasCSAT returns true if the message is a CSAT message.
func (m *Message) HasCSAT() bool {
	var meta map[string]interface{}
	if err := json.Unmarshal([]byte(m.Meta), &meta); err != nil {
		return false
	}
	isCsat, _ := meta["is_csat"].(bool)
	return isCsat
}

// IncomingMessage links a message with the contact information and inbox id.
type IncomingMessage struct {
	Message Message
	Contact umodels.User
	InboxID int
}

type Status struct {
	ID        int       `db:"id" json:"id"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	Name      string    `db:"name" json:"name"`
}

type Priority struct {
	ID        int       `db:"id" json:"id"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	Name      string    `db:"name" json:"name"`
}
