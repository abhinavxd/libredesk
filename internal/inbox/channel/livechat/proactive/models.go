package proactive

import (
	"encoding/json"
	"time"

	amodels "github.com/abhinavxd/libredesk/internal/automation/models"
)

type Campaign struct {
	ID              string            `json:"id"`
	Name            string            `json:"name"`
	Enabled         bool              `json:"enabled"`
	Message         string            `json:"message"`
	SenderID        int               `json:"sender_id"`
	TeamID          int               `json:"team_id"`
	Audience        string            `json:"audience"`
	IncludeURLs     []string          `json:"include_urls"`
	ExcludeURLs     []string          `json:"exclude_urls"`
	Conditions      amodels.RuleGroup `json:"conditions"`
	Event           string            `json:"event"`
	DelaySeconds    int               `json:"delay_seconds"`
	BusinessHoursID int               `json:"business_hours_id"`
	BusinessHours   string            `json:"business_hours"`
	Desktop         bool              `json:"desktop"`
	Mobile          bool              `json:"mobile"`
	Repeat          string            `json:"repeat"`
	RepeatHours     int               `json:"repeat_hours"`
}

type Context struct {
	URL           string    `json:"url"`
	BrowserKey    string    `json:"browser_key"`
	SessionKey    string    `json:"session_key"`
	Event         string    `json:"event"`
	ActiveSeconds int       `json:"active_seconds"`
	Mobile        bool      `json:"mobile"`
	Visitor       bool      `json:"visitor"`
	Now           time.Time `json:"now"`
	ContactID     int       `json:"contact_id"`
}

type Delivery struct {
	ID               string          `db:"id" json:"id"`
	CampaignID       string          `db:"campaign_id" json:"campaign_id"`
	InboxID          int             `db:"inbox_id" json:"-"`
	BrowserKey       string          `db:"browser_key" json:"-"`
	SessionKey       string          `db:"session_key" json:"-"`
	ContactID        int             `db:"contact_id" json:"-"`
	Snapshot         json.RawMessage `db:"snapshot" json:"snapshot"`
	CreatedAt        time.Time       `db:"created_at" json:"-"`
	Displayed        bool            `db:"displayed" json:"-"`
	Opened           bool            `db:"opened" json:"-"`
	Dismissed        bool            `db:"dismissed" json:"-"`
	Replied          bool            `db:"replied" json:"-"`
	ConversationUUID string          `db:"conversation_uuid" json:"-"`
}

type Snapshot struct {
	Message  string `json:"message"`
	Sender   string `json:"sender"`
	Avatar   string `json:"avatar"`
	SenderID int    `json:"sender_id"`
	TeamID   int    `json:"team_id"`
}

type Stats struct {
	CampaignID string `db:"campaign_id" json:"campaign_id"`
	Displayed  int    `db:"displayed" json:"displayed"`
	Opened     int    `db:"opened" json:"opened"`
	Dismissed  int    `db:"dismissed" json:"dismissed"`
	Replied    int    `db:"replied" json:"replied"`
}
