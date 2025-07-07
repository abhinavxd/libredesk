package models

import (
	"github.com/volatiletech/null/v9"
	"time"
)

type Article struct {
	ID          int         `db:"id" json:"id"`
	CreatedAt   time.Time   `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time   `db:"updated_at" json:"updated_at"`
	Title       string      `db:"title" json:"title"`
	Content     null.String `db:"content" json:"content"`
	SectionID   null.Int    `db:"section_id" json:"section_id"`
	PublishedAt null.Time   `db:"published_at" json:"published_at"`
	IsPublished bool        `db:"is_published" json:"is_published"`
}

type ArticleWithDetails struct {
	Article
	SectionName  null.String `db:"section_name" json:"section_name"`
	CategoryID   null.Int    `db:"category_id" json:"category_id"`
	CategoryName null.String `db:"category_name" json:"category_name"`
}