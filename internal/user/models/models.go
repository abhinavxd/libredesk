package models

import (
	"time"

	tmodels "github.com/abhinavxd/libredesk/internal/team/models"
	"github.com/lib/pq"
	"github.com/volatiletech/null/v9"
)
type User struct {
	ID               int            `db:"id" json:"id"`
	CreatedAt        time.Time      `db:"created_at" json:"created_at"`
	UpdatedAt        time.Time      `db:"updated_at" json:"updated_at"`
	FirstName        string         `db:"first_name" json:"first_name"`
	LastName         string         `db:"last_name" json:"last_name"`
	Email            null.String    `db:"email" json:"email,omitempty"`
	Type             string         `db:"type" json:"type"`
	AvatarURL        null.String    `db:"avatar_url" json:"avatar_url"`
	Disabled         bool           `db:"disabled" json:"disabled"`
	Password         string         `db:"password" json:"-"`
	Roles            pq.StringArray `db:"roles" json:"roles,omitempty"`
	Permissions      pq.StringArray `db:"permissions" json:"permissions,omitempty"`
	Meta             pq.StringArray `db:"meta" json:"meta,omitempty"`
	CustomAttributes pq.StringArray `db:"custom_attributes" json:"custom_attributes,omitempty"`
	Teams            tmodels.Teams  `db:"teams" json:"teams,omitempty"`
	ContactChannelID int            `db:"contact_channel_id,omitempty"`
	NewPassword      string         `db:"-" json:"new_password,omitempty"`
	SendWelcomeEmail bool           `db:"-" json:"send_welcome_email,omitempty"`
	InboxID          int            `json:"-"`
	SourceChannel    null.String    `json:"-"`
	SourceChannelID  null.String    `json:"-"`
}

func (u *User) FullName() string {
	return u.FirstName + " " + u.LastName
}
