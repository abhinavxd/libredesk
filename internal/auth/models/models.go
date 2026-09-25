package models

// User represents an authenticated user.
type User struct {
	SessionVersion int    `db:"session_version" json:"-"`
	ID             int    `json:"id"`
	FirstName      string `json:"first_name"`
	LastName       string `json:"last_name"`
	Email          string `json:"email,omitempty"`
}
