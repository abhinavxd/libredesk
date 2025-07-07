package models

import (
	"github.com/volatiletech/null/v9"
	"time"
)

type Section struct {
	ID         int       `db:"id" json:"id"`
	CreatedAt  time.Time `db:"created_at" json:"created_at"`
	UpdatedAt  time.Time `db:"updated_at" json:"updated_at"`
	Name       string    `db:"name" json:"name"`
	CategoryID null.Int  `db:"category_id" json:"category_id"`
}

type SectionWithCategory struct {
	Section
	CategoryName *string `db:"category_name" json:"category_name"`
}
