package models

import "time"
import "github.com/abhinavxd/libredesk/internal/article_section/models"

type Category struct {
	ID        int       `db:"id" json:"id"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
	Name      string    `db:"name" json:"name"`
}

type CategoryWithSections struct {
	ID        int              `json:"id" db:"id"`
	CreatedAt time.Time        `json:"created_at" db:"created_at"`
	UpdatedAt time.Time        `json:"updated_at" db:"updated_at"`
	Name      string           `json:"name" db:"name"`
	Sections  []models.Section `json:"sections" db:"sections"`
}
