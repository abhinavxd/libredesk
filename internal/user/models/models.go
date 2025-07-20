package models

import (
	"encoding/json"
	"slices"
	"time"

	rmodels "github.com/abhinavxd/libredesk/internal/role/models"
	tmodels "github.com/abhinavxd/libredesk/internal/team/models"
	"github.com/lib/pq"
	"github.com/volatiletech/null/v9"
)

const (
	UserModel = "user"

	SystemUserEmail = "System"

	// User types
	UserTypeAgent   = "agent"
	UserTypeContact = "contact"
	UserTypeVisitor = "visitor"

	// User availability statuses
	Online  = "online"
	Offline = "offline"
	// Away due to inactivity
	Away = "away"
	// Away due to manual setting from sidebar
	AwayManual         = "away_manual"
	AwayAndReassigning = "away_and_reassigning"
)

type User struct {
	ID                     int             `db:"id" json:"id"`
	CreatedAt              time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt              time.Time       `db:"updated_at" json:"updated_at"`
	FirstName              string          `db:"first_name" json:"first_name"`
	LastName               string          `db:"last_name" json:"last_name"`
	Email                  null.String     `db:"email" json:"email"`
	Type                   string          `db:"type" json:"type"`
	AvailabilityStatus     string          `db:"availability_status" json:"availability_status"`
	PhoneNumberCallingCode null.String     `db:"phone_number_calling_code" json:"phone_number_calling_code"`
	PhoneNumber            null.String     `db:"phone_number" json:"phone_number"`
	AvatarURL              null.String     `db:"avatar_url" json:"avatar_url"`
	Enabled                bool            `db:"enabled" json:"enabled"`
	Password               null.String     `db:"password" json:"-"`
	LastActiveAt           null.Time       `db:"last_active_at" json:"last_active_at"`
	LastLoginAt            null.Time       `db:"last_login_at" json:"last_login_at"`
	Roles                  pq.StringArray  `db:"roles" json:"roles"`
	Permissions            pq.StringArray  `db:"permissions" json:"permissions"`
	Meta                   pq.StringArray  `db:"meta" json:"meta"`
	CustomAttributes       json.RawMessage `db:"custom_attributes" json:"custom_attributes"`
	ExternalUserID         null.String     `db:"external_user_id" json:"external_user_id"`
	Teams                  tmodels.Teams   `db:"teams" json:"teams"`
	NewPassword            string          `db:"-" json:"new_password,omitempty"`
	SendWelcomeEmail       bool            `db:"-" json:"send_welcome_email,omitempty"`

	// API Key fields
	APIKey           null.String `db:"api_key" json:"api_key"`
	APIKeyLastUsedAt null.Time   `db:"api_key_last_used_at" json:"api_key_last_used_at"`
	APISecret        null.String `db:"api_secret" json:"-"`

	Total int `json:"total,omitempty"`
}

// ChatUser is a user with limited fields for live chat.
type ChatUser struct {
	ID                 int         `db:"id" json:"id"`
	FirstName          string      `db:"first_name" json:"first_name"`
	LastName           string      `db:"last_name" json:"last_name"`
	AvatarURL          null.String `db:"avatar_url" json:"avatar_url"`
	AvailabilityStatus string      `db:"availability_status" json:"availability_status"`
	Type               string      `db:"type" json:"type"`
	ActiveAt           null.Time   `db:"active_at" json:"active_at"`
}

type Note struct {
	ID        int         `db:"id" json:"id"`
	CreatedAt time.Time   `db:"created_at" json:"created_at"`
	UpdatedAt time.Time   `db:"updated_at" json:"updated_at"`
	ContactID int         `db:"contact_id" json:"contact_id"`
	Note      string      `db:"note" json:"note"`
	UserID    int         `db:"user_id" json:"user_id"`
	FirstName string      `db:"first_name" json:"first_name"`
	LastName  string      `db:"last_name" json:"last_name"`
	AvatarURL null.String `db:"avatar_url" json:"avatar_url"`
}

func (u *User) FullName() string {
	return u.FirstName + " " + u.LastName
}

func (u *User) HasAdminRole() bool {
	return slices.Contains(u.Roles, rmodels.RoleAdmin)
}

func (u *User) IsSystemUser() bool {
	return u.Email.String == SystemUserEmail
}
