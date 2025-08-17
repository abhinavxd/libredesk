package models

import "time"

type HelpCenter struct {
	ID            int       `db:"id" json:"id"`
	CreatedAt     time.Time `db:"created_at" json:"created_at"`
	UpdatedAt     time.Time `db:"updated_at" json:"updated_at"`
	Name          string    `db:"name" json:"name"`
	Slug          string    `db:"slug" json:"slug"`
	PageTitle     string    `db:"page_title" json:"page_title"`
	ViewCount     int       `db:"view_count" json:"view_count"`
	DefaultLocale string    `db:"default_locale" json:"default_locale"`
}

type Collection struct {
	ID           int       `db:"id" json:"id"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time `db:"updated_at" json:"updated_at"`
	HelpCenterID int       `db:"help_center_id" json:"help_center_id"`
	Slug         string    `db:"slug" json:"slug"`
	ParentID     *int      `db:"parent_id" json:"parent_id"`
	Locale       string    `db:"locale" json:"locale"`
	Name         string    `db:"name" json:"name"`
	Description  *string   `db:"description" json:"description"`
	SortOrder    int       `db:"sort_order" json:"sort_order"`
	IsPublished  bool      `db:"is_published" json:"is_published"`
}

type Article struct {
	ID           int       `db:"id" json:"id"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time `db:"updated_at" json:"updated_at"`
	CollectionID int       `db:"collection_id" json:"collection_id"`
	Slug         string    `db:"slug" json:"slug"`
	Locale       string    `db:"locale" json:"locale"`
	Title        string    `db:"title" json:"title"`
	Content      string    `db:"content" json:"content"`
	SortOrder    int       `db:"sort_order" json:"sort_order"`
	Status       string    `db:"status" json:"status"`
	ViewCount    int       `db:"view_count" json:"view_count"`
	AIEnabled    bool      `db:"ai_enabled" json:"ai_enabled"`
}

// ArticleStatus constants
const (
	ArticleStatusDraft     = "draft"
	ArticleStatusPublished = "published"
)

// CollectionWithDepth is used for hierarchy validation
type CollectionWithDepth struct {
	Collection
	Depth int `json:"depth"`
}

// Tree structures for nested API response
type TreeCollection struct {
	Collection
	Articles []Article        `json:"articles"`
	Children []TreeCollection `json:"children"`
}

type TreeResponse struct {
	HelpCenter HelpCenter       `json:"help_center"`
	Tree       []TreeCollection `json:"tree"`
}

// ArticleSearchResult represents an article with similarity score for vector search
type ArticleSearchResult struct {
	Article
	Similarity float64 `db:"similarity" json:"similarity"`
}

// KnowledgeBaseResult represents a knowledge base search result
type KnowledgeBaseResult struct {
	SourceType   string  `db:"source_type" json:"source_type"`
	SourceID     int     `db:"source_id" json:"source_id"`
	Title        string  `db:"title" json:"title"`
	Content      string  `db:"content" json:"content"`
	HelpCenterID *int    `db:"help_center_id" json:"help_center_id"`
	Similarity   float64 `db:"similarity" json:"similarity"`
}
