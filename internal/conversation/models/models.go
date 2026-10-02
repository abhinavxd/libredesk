package models

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/jakedolan443/fernmail/internal/attachment"
	mmodels "github.com/jakedolan443/fernmail/internal/media/models"
	"github.com/jakedolan443/fernmail/internal/resourcepolicy"

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
	TeamAllConversations        = "team_all"
	MentionedConversations      = "mentioned"

	MessageIncoming = "incoming"
	MessageOutgoing = "outgoing"
	MessageActivity = "activity"

	SenderTypeAgent   = "agent"
	SenderTypeContact = "contact"

	MentionTypeAgent = "agent"
	MentionTypeTeam  = "team"

	MessageStatusPending  = "pending"
	MessageStatusSent     = "sent"
	MessageStatusFailed   = "failed"
	MessageStatusReceived = "received"

	ActivityStatusChange        = "status_change"
	ActivityPriorityChange      = "priority_change"
	ActivityAssignedUserChange  = "assigned_user_change"
	ActivityAssignedTeamChange  = "assigned_team_change"
	ActivityAssigneeUserRemoved = "assignee_user_removed"
	ActivitySelfAssign          = "self_assign"
	ActivitySelfUnassign        = "self_unassign"
	ActivitySLASet              = "sla_set"
	ActivityParticipantAdded    = "participant_added"

	ContentTypeText = "text"
	ContentTypeHTML = "html"
)

// ConversationListItem represents a conversation in list views
type ConversationListItem struct {
	InboxID               int                     `db:"inbox_id" json:"inbox_id"`
	AddressID             null.Int                `db:"address_id" json:"address_id"`
	Total                 int                     `db:"total" json:"-"`
	ID                    int                     `db:"id" json:"id"`
	CreatedAt             time.Time               `db:"created_at" json:"created_at"`
	UpdatedAt             time.Time               `db:"updated_at" json:"updated_at"`
	UUID                  string                  `db:"uuid" json:"uuid"`
	ReferenceNumber       string                  `db:"reference_number" json:"reference_number"`
	WaitingSince          null.Time               `db:"waiting_since" json:"waiting_since"`
	Contact               ConversationListContact `db:"contact" json:"correspondent"`
	InboxChannel          string                  `db:"inbox_channel" json:"inbox_channel"`
	InboxName             string                  `db:"inbox_name" json:"inbox_name"`
	SLAPolicyID           null.Int                `db:"sla_policy_id" json:"-"`
	FirstReplyAt          null.Time               `db:"first_reply_at" json:"first_reply_at"`
	LastReplyAt           null.Time               `db:"last_reply_at" json:"last_reply_at"`
	ResolvedAt            null.Time               `db:"resolved_at" json:"resolved_at"`
	Subject               null.String             `db:"subject" json:"subject"`
	LastMessage           null.String             `db:"last_message" json:"last_message"`
	LastMessageAt         null.Time               `db:"last_message_at" json:"last_message_at"`
	LastMessageSender     null.String             `db:"last_message_sender" json:"last_message_sender"`
	LastInteraction       null.String             `db:"last_interaction" json:"last_interaction"`
	LastInteractionAt     null.Time               `db:"last_interaction_at" json:"last_interaction_at"`
	LastInteractionSender null.String             `db:"last_interaction_sender" json:"last_interaction_sender"`
	NextSLADeadlineAt     null.Time               `db:"next_sla_deadline_at" json:"-"`
	PriorityID            null.Int                `db:"priority_id" json:"-"`
	AssignedUserID        null.Int                `db:"assigned_user_id" json:"-"`
	AssignedTeamID        null.Int                `db:"assigned_team_id" json:"-"`
	UnreadMessageCount    int                     `db:"unread_message_count" json:"unread_message_count"`
	Status                null.String             `db:"status" json:"status"`
	Priority              null.String             `db:"priority" json:"-"`
	FirstResponseDueAt    null.Time               `db:"first_response_deadline_at" json:"-"`
	ResolutionDueAt       null.Time               `db:"resolution_deadline_at" json:"-"`
	AppliedSLAID          null.Int                `db:"applied_sla_id" json:"-"`
	NextResponseDueAt     null.Time               `db:"next_response_deadline_at" json:"-"`
	NextResponseMetAt     null.Time               `db:"next_response_met_at" json:"-"`
	MentionedMessageUUID  null.String             `db:"mentioned_message_uuid" json:"mentioned_message_uuid"`
}

// ConversationListContact represents contact info in conversation list views
type ConversationListContact struct {
	CreatedAt time.Time   `db:"created_at" json:"created_at"`
	UpdatedAt time.Time   `db:"updated_at" json:"updated_at"`
	FirstName string      `db:"first_name" json:"first_name"`
	LastName  string      `db:"last_name" json:"last_name"`
	Email     null.String `db:"email" json:"email"`
	AvatarURL null.String `db:"avatar_url" json:"avatar_url"`
}

// ConversationRef is a lightweight (id, uuid) reference to a conversation.
type ConversationRef struct {
	ID   int    `db:"id" json:"id"`
	UUID string `db:"uuid" json:"uuid"`
}

type Conversation struct {
	ID                        int                    `db:"id" json:"id"`
	CreatedAt                 time.Time              `db:"created_at" json:"created_at"`
	UpdatedAt                 time.Time              `db:"updated_at" json:"updated_at"`
	UUID                      string                 `db:"uuid" json:"uuid"`
	ContactID                 int                    `db:"contact_id" json:"-"`
	InboxID                   int                    `db:"inbox_id" json:"inbox_id"`
	AddressID                 null.Int               `db:"address_id" json:"address_id"`
	ClosedAt                  null.Time              `db:"closed_at" json:"closed_at"`
	ResolvedAt                null.Time              `db:"resolved_at" json:"resolved_at"`
	ContactLastSeenAt         null.Time              `db:"contact_last_seen_at" json:"-"`
	ReferenceNumber           string                 `db:"reference_number" json:"reference_number"`
	Priority                  null.String            `db:"priority" json:"-"`
	PriorityID                null.Int               `db:"priority_id" json:"-"`
	Status                    null.String            `db:"status" json:"status"`
	StatusCategory            null.String            `db:"status_category" json:"status_category"`
	StatusID                  null.Int               `db:"status_id" json:"status_id"`
	FirstReplyAt              null.Time              `db:"first_reply_at" json:"first_reply_at"`
	LastReplyAt               null.Time              `db:"last_reply_at" json:"last_reply_at"`
	AssignedUserID            null.Int               `db:"assigned_user_id" json:"-"`
	AssignedTeamID            null.Int               `db:"assigned_team_id" json:"-"`
	WaitingSince              null.Time              `db:"waiting_since" json:"waiting_since"`
	SnoozedUntil              null.Time              `db:"snoozed_until" json:"snoozed_until"`
	Subject                   null.String            `db:"subject" json:"subject"`
	InboxMail                 string                 `db:"inbox_mail" json:"inbox_mail"`
	InboxReplyTo              string                 `db:"inbox_reply_to" json:"inbox_reply_to"`
	InboxName                 string                 `db:"inbox_name" json:"inbox_name"`
	InboxChannel              string                 `db:"inbox_channel" json:"inbox_channel"`
	Meta                      json.RawMessage        `db:"meta" json:"meta"`
	LatestIncomingRecipient   string                 `db:"latest_incoming_recipient" json:"-"`
	CustomAttributes          json.RawMessage        `db:"custom_attributes" json:"-"`
	LastMessageAt             null.Time              `db:"last_message_at" json:"last_message_at"`
	LastMessage               null.String            `db:"last_message" json:"last_message"`
	LastMessageSender         null.String            `db:"last_message_sender" json:"last_message_sender"`
	LastInteraction           null.String            `db:"last_interaction" json:"last_interaction"`
	LastInteractionAt         null.Time              `db:"last_interaction_at" json:"last_interaction_at"`
	LastInteractionSender     null.String            `db:"last_interaction_sender" json:"last_interaction_sender"`
	Contact                   ConversationContact    `db:"contact" json:"correspondent"`
	SLAPolicyID               null.Int               `db:"sla_policy_id" json:"-"`
	SlaPolicyName             null.String            `db:"sla_policy_name" json:"-"`
	AppliedSLAID              null.Int               `db:"applied_sla_id" json:"-"`
	NextSLADeadlineAt         null.Time              `db:"next_sla_deadline_at" json:"-"`
	FirstResponseDueAt        null.Time              `db:"first_response_deadline_at" json:"-"`
	ResolutionDueAt           null.Time              `db:"resolution_deadline_at" json:"-"`
	NextResponseDueAt         null.Time              `db:"next_response_deadline_at" json:"-"`
	NextResponseMetAt         null.Time              `db:"next_response_met_at" json:"-"`
	LastContinuityEmailSentAt null.Time              `db:"last_continuity_email_sent_at" json:"-"`
	CSATRating                null.Int               `db:"csat_rating" json:"-"`
	CSATFeedback              null.String            `db:"csat_feedback" json:"-"`
	CSATRespondedAt           null.Time              `db:"csat_responded_at" json:"-"`
	PreviousConversations     []PreviousConversation `db:"-" json:"-"`
}

type ConversationContact struct {
	ID        int         `db:"id" json:"id"`
	FirstName string      `db:"first_name" json:"first_name"`
	LastName  string      `db:"last_name" json:"last_name"`
	Email     null.String `db:"email" json:"email"`
	Type      string      `db:"type" json:"type"`
	AvatarURL null.String `db:"avatar_url" json:"avatar_url"`
}

func (c *ConversationContact) FullName() string {
	name := strings.TrimSpace(c.FirstName + " " + c.LastName)
	if name == "" {
		return c.Email.String
	}
	return name
}

type PreviousConversation struct {
	ID            int                         `db:"id" json:"id"`
	CreatedAt     time.Time                   `db:"created_at" json:"created_at"`
	UpdatedAt     time.Time                   `db:"updated_at" json:"updated_at"`
	UUID          string                      `db:"uuid" json:"uuid"`
	Subject       string                      `db:"subject" json:"subject"`
	Contact       PreviousConversationContact `db:"contact" json:"correspondent"`
	LastMessage   null.String                 `db:"last_message" json:"last_message"`
	LastMessageAt null.Time                   `db:"last_message_at" json:"last_message_at"`
}

type PreviousConversationContact struct {
	FirstName string      `db:"first_name" json:"first_name"`
	LastName  string      `db:"last_name" json:"last_name"`
	AvatarURL null.String `db:"avatar_url" json:"avatar_url"`
}

type ConversationParticipant struct {
	ID        int         `db:"id" json:"id"`
	FirstName string      `db:"first_name" json:"first_name"`
	LastName  string      `db:"last_name" json:"last_name"`
	AvatarURL null.String `db:"avatar_url" json:"avatar_url"`
}

type MessageAuthor struct {
	ID                 int         `db:"id" json:"id"`
	FirstName          string      `db:"first_name" json:"first_name"`
	LastName           string      `db:"last_name" json:"last_name"`
	Email              null.String `db:"email" json:"email"`
	AvatarURL          null.String `db:"avatar_url" json:"avatar_url"`
	AvailabilityStatus string      `db:"availability_status" json:"availability_status"`
	Type               string      `db:"type" json:"type"`
	LastActiveAt       null.Time   `db:"last_active_at" json:"last_active_at"`
}

func (a *MessageAuthor) FullName() string {
	if a.LastName == "" {
		return a.FirstName
	}
	return a.FirstName + " " + a.LastName
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
	UploadUserID      int                    `db:"-" json:"-"`
	ReplyToSourceID   string                 `db:"reply_to_source_id" json:"-"`
	Total             int                    `db:"total" json:"-"`
	ID                int                    `db:"id" json:"id"`
	CreatedAt         time.Time              `db:"created_at" json:"created_at"`
	UpdatedAt         time.Time              `db:"updated_at" json:"updated_at"`
	UUID              string                 `db:"uuid" json:"uuid"`
	Type              string                 `db:"type" json:"type"`
	Status            string                 `db:"status" json:"status"`
	ConversationID    int                    `db:"conversation_id" json:"conversation_id"`
	ConversationUUID  string                 `db:"conversation_uuid" json:"conversation_uuid"`
	Content           string                 `db:"content" json:"content"`
	TextContent       string                 `db:"text_content" json:"text_content"`
	ContentType       string                 `db:"content_type" json:"content_type"`
	Private           bool                   `db:"private" json:"private"`
	SourceID          null.String            `db:"source_id" json:"-"`
	SenderID          int                    `db:"sender_id" json:"sender_id"`
	SenderType        string                 `db:"sender_type" json:"sender_type"`
	InboxID           int                    `db:"inbox_id" json:"-"`
	Meta              json.RawMessage        `db:"meta" json:"meta"`
	Attachments       attachment.Attachments `db:"attachments" json:"attachments"`
	From              string                 `db:"from"  json:"-"`
	Subject           string                 `db:"subject" json:"-"`
	Channel           string                 `db:"channel" json:"-"`
	To                pq.StringArray         `db:"to"  json:"-"`
	CC                pq.StringArray         `db:"cc" json:"-"`
	BCC               pq.StringArray         `db:"bcc" json:"-"`
	MessageReceiverID int                    `db:"message_receiver_id" json:"-"`
	Media             []mmodels.Media        `json:"-"`
	Author            MessageAuthor          `db:"author" json:"author"`

	Display *resourcepolicy.Display `db:"-" json:"display,omitempty"`
}

// IsContinuityMessage returns true if the message is a continuity email.
func (m *Message) IsContinuityMessage() bool {
	var meta map[string]any
	if err := json.Unmarshal([]byte(m.Meta), &meta); err != nil {
		return false
	}
	isContinuity, _ := meta["continuity_email"].(bool)
	return isContinuity
}

// csatMeta unmarshals the message meta and returns the map and whether is_csat is true.
func (m *Message) csatMeta() (map[string]any, bool) {
	var meta map[string]any
	if err := json.Unmarshal([]byte(m.Meta), &meta); err != nil {
		return nil, false
	}
	isCsat, _ := meta["is_csat"].(bool)
	return meta, isCsat
}

// HasCSAT returns true if the message is a CSAT message.
func (m *Message) HasCSAT() bool {
	_, isCsat := m.csatMeta()
	return isCsat
}

// StripCSATUUID removes the csat_uuid from the message meta.
// Used to hide CSAT links from agent sessions while keeping them for API key callers.
func (m *Message) StripCSATUUID() {
	var meta map[string]any
	if err := json.Unmarshal([]byte(m.Meta), &meta); err != nil {
		return
	}
	delete(meta, "csat_uuid")
	if updatedMeta, err := json.Marshal(meta); err == nil {
		m.Meta = json.RawMessage(updatedMeta)
	}
}

// OutboundMessage contains fields needed for sending messages via inboxes.
type OutboundMessage struct {
	// Core message identifiers
	UUID             string
	ConversationUUID string

	// Sender info
	SenderID          int
	MessageReceiverID int

	// Content
	Content     string
	TextContent string
	ContentType string
	AltContent  string // Plain text alternative for HTML emails

	// Email-specific fields
	From     string
	To       []string
	CC       []string
	BCC      []string
	Subject  string
	SourceID string

	// Threading (email)
	References []string
	InReplyTo  string
	ReplyTo    string

	// Attachments
	Attachments attachment.Attachments

	// Metadata
	Meta      json.RawMessage
	CreatedAt time.Time
}

// ToOutbound converts a Message to an OutboundMessage for transport.
// Transport-only fields (References, InReplyTo, Headers, AltContent) must be set by caller.
func (m *Message) ToOutbound() OutboundMessage {
	return OutboundMessage{
		UUID:              m.UUID,
		ConversationUUID:  m.ConversationUUID,
		SenderID:          m.SenderID,
		MessageReceiverID: m.MessageReceiverID,
		Content:           m.Content,
		TextContent:       m.TextContent,
		ContentType:       m.ContentType,
		From:              m.From,
		To:                m.To,
		CC:                m.CC,
		BCC:               m.BCC,
		Subject:           m.Subject,
		SourceID:          m.SourceID.String,
		Attachments:       m.Attachments,
		Meta:              m.Meta,
		CreatedAt:         m.CreatedAt,
	}
}

type IncomingContact struct {
	ID        int
	FirstName string
	LastName  string
	Email     null.String
}

type IncomingMessage struct {
	AddressID   int
	MailboxKey  string
	UIDValidity uint32
	UID         uint32
	// Channel context
	Channel string
	InboxID int

	// Contact
	Contact IncomingContact

	// Message fields
	Subject     string
	SourceID    null.String
	Content     string
	ContentType string
	Meta        json.RawMessage
	Attachments attachment.Attachments
	EmailAlias  string

	// Email threading
	ConversationUUIDFromReplyTo string // UUID extracted from plus-addressed recipient (inbox+conv-{uuid}@domain)
	InReplyTo                   string
	References                  []string
}

// ToMessage converts IncomingMessage to a Message for DB insertion.
func (in *IncomingMessage) ToMessage(senderID, conversationID int, conversationUUID string) Message {
	return Message{
		Channel:          in.Channel,
		SenderType:       SenderTypeContact,
		Type:             MessageIncoming,
		Status:           MessageStatusReceived,
		InboxID:          in.InboxID,
		Subject:          in.Subject,
		SourceID:         in.SourceID,
		Content:          in.Content,
		ContentType:      in.ContentType,
		Meta:             in.Meta,
		Attachments:      in.Attachments,
		SenderID:         senderID,
		ConversationID:   conversationID,
		ConversationUUID: conversationUUID,
	}
}

type Status struct {
	ID        int       `db:"id" json:"id"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	Name      string    `db:"name" json:"name"`
	Category  string    `db:"category" json:"category"`
}

type Priority struct {
	ID        int       `db:"id" json:"id"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	Name      string    `db:"name" json:"name"`
}

// ConversationDraft represents a draft reply for a conversation.
type ConversationDraft struct {
	ID               int64           `db:"id" json:"id"`
	ConversationID   int64           `db:"conversation_id" json:"conversation_id"`
	ConversationUUID string          `db:"conversation_uuid" json:"conversation_uuid"`
	UserID           int64           `db:"user_id" json:"user_id"`
	Type             string          `db:"type" json:"type"`
	Content          string          `db:"content" json:"content"`
	CreatedAt        time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt        time.Time       `db:"updated_at" json:"updated_at"`
	Meta             json.RawMessage `db:"meta" json:"meta"`
}

// MentionInput represents a mention in a private note from frontend.
type MentionInput struct {
	Type string `json:"type"` // "agent" or "team"
	ID   int    `json:"id"`
}
