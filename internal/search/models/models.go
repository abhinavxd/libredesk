package models

import (
	"time"

	"github.com/lib/pq"
	"github.com/volatiletech/null/v9"
)

const (
	SortNewest       Sort = "newest"
	SortOldest       Sort = "oldest"
	SortStartedFirst Sort = "started_first"
	SortStartedLast  Sort = "started_last"
)

type Sort string

// ReadScope carries an agent's conversation read permissions for filtering search results.
type ReadScope struct {
	UserID         int
	TeamIDs        []int
	Read           bool
	ReadAll        bool
	ReadAssigned   bool
	ReadTeamAll    bool
	ReadTeamInbox  bool
	ReadUnassigned bool
}

type Query struct {
	Term     string
	Filters  string
	Cursor   string
	Sort     Sort
	PageSize int
}

type Contact struct {
	FirstName string      `db:"first_name" json:"first_name"`
	LastName  null.String `db:"last_name" json:"last_name"`
	Email     null.String `db:"email" json:"email"`
	AvatarURL null.String `db:"avatar_url" json:"avatar_url"`
}

type Assignee struct {
	FirstName null.String `db:"first_name" json:"first_name"`
	LastName  null.String `db:"last_name" json:"last_name"`
	AvatarURL null.String `db:"avatar_url" json:"avatar_url"`
}

type Sender struct {
	FirstName null.String `db:"first_name" json:"first_name"`
	LastName  null.String `db:"last_name" json:"last_name"`
}

type ConversationResult struct {
	ID              int            `db:"id" json:"-"`
	ReferenceMatch  bool           `db:"reference_match" json:"-"`
	CreatedAt       time.Time      `db:"created_at" json:"created_at"`
	UUID            string         `db:"uuid" json:"uuid"`
	ReferenceNumber string         `db:"reference_number" json:"reference_number"`
	Subject         null.String    `db:"subject" json:"subject"`
	LastMessage     null.String    `db:"last_message" json:"last_message"`
	LastMessageAt   null.Time      `db:"last_message_at" json:"last_message_at"`
	AssignedUserID  null.Int       `db:"assigned_user_id" json:"assigned_user_id"`
	AssignedTeamID  null.Int       `db:"assigned_team_id" json:"assigned_team_id"`
	Assignee        Assignee       `db:"assignee" json:"assignee"`
	TeamName        null.String    `db:"team_name" json:"team_name"`
	Status          null.String    `db:"status" json:"status"`
	Priority        null.String    `db:"priority" json:"priority"`
	InboxName       null.String    `db:"inbox_name" json:"inbox_name"`
	InboxChannel    null.String    `db:"inbox_channel" json:"inbox_channel"`
	Contact         Contact        `db:"contact" json:"contact"`
	Tags            pq.StringArray `db:"tags" json:"tags"`
}

type MessageResult struct {
	ID                          int         `db:"id" json:"-"`
	UUID                        string      `db:"uuid" json:"uuid"`
	CreatedAt                   time.Time   `db:"created_at" json:"created_at"`
	Type                        string      `db:"type" json:"type"`
	Private                     bool        `db:"private" json:"private"`
	TextContent                 string      `db:"text_content" json:"text_content"`
	Snippet                     null.String `db:"snippet" json:"snippet"`
	Sender                      Sender      `db:"sender" json:"sender"`
	ConversationCreatedAt       time.Time   `db:"conversation_created_at" json:"conversation_created_at"`
	ConversationUUID            string      `db:"conversation_uuid" json:"conversation_uuid"`
	ConversationReferenceNumber string      `db:"conversation_reference_number" json:"conversation_reference_number"`
	ConversationSubject         null.String `db:"conversation_subject" json:"conversation_subject"`
	ConversationStatus          null.String `db:"conversation_status" json:"conversation_status"`
	AssignedUserID              null.Int    `db:"assigned_user_id" json:"assigned_user_id"`
	AssignedTeamID              null.Int    `db:"assigned_team_id" json:"assigned_team_id"`
	Assignee                    Assignee    `db:"assignee" json:"assignee"`
	TeamName                    null.String `db:"team_name" json:"team_name"`
	Priority                    null.String `db:"priority" json:"priority"`
	InboxName                   null.String `db:"inbox_name" json:"inbox_name"`
	InboxChannel                null.String `db:"inbox_channel" json:"inbox_channel"`
	Contact                     Contact     `db:"contact" json:"contact"`
}

type ContactResult struct {
	ID             int         `db:"id" json:"id"`
	CreatedAt      time.Time   `db:"created_at" json:"created_at"`
	FirstName      string      `db:"first_name" json:"first_name"`
	LastName       string      `db:"last_name" json:"last_name"`
	Email          string      `db:"email" json:"email"`
	ExternalUserID null.String `db:"external_user_id" json:"external_user_id"`
}
