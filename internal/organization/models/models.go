package models

import (
	"time"

	"github.com/volatiletech/null/v9"
)

// Organization represents a company or organization that contacts belong to.
type Organization struct {
	ID          string      `db:"id" json:"id"`
	CreatedAt   time.Time   `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time   `db:"updated_at" json:"updated_at"`
	Name        string      `db:"name" json:"name"`
	Website     null.String `db:"website" json:"website"`
	EmailDomain null.String `db:"email_domain" json:"email_domain"`
	Phone       null.String `db:"phone" json:"phone"`

	// Total is used for pagination count
	Total int `db:"total" json:"-"`
}

// OrganizationCompact represents a minimal organization for list views.
type OrganizationCompact struct {
	ID   string `db:"id" json:"id"`
	Name string `db:"name" json:"name"`

	// Total is used for pagination count
	Total int `db:"total" json:"-"`
}
