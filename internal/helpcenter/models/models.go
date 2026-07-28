package models

import (
	"encoding/json"
	"time"
)

const (
	ArticleStatusDraft     = "draft"
	ArticleStatusPublished = "published"
	ArticleStatusArchived  = "archived"
)

type HelpCenter struct {
	ID             int             `db:"id" json:"id"`
	CreatedAt      time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt      time.Time       `db:"updated_at" json:"updated_at"`
	Name           string          `db:"name" json:"name"`
	Slug           string          `db:"slug" json:"slug"`
	PageTitle      string          `db:"page_title" json:"page_title"`
	HeaderText     string          `db:"header_text" json:"header_text"`
	LogoURL        string          `db:"logo_url" json:"logo_url"`
	Color          string          `db:"color" json:"color"`
	NavLinks       json.RawMessage `db:"nav_links" json:"nav_links"`
	CustomCSS      string          `db:"custom_css" json:"custom_css"`
	CustomJS       string          `db:"custom_js" json:"custom_js"`
	ViewCount      int             `db:"view_count" json:"view_count"`
	DefaultLocale  string          `db:"default_locale" json:"default_locale"`
	AllowedLocales json.RawMessage `db:"allowed_locales" json:"allowed_locales"`
	IsActive       bool            `db:"is_active" json:"is_active"`
	Theme          json.RawMessage `db:"theme" json:"theme"`
}

// Theme holds the customizable branding for a help center's public pages.
type Theme struct {
	Favicon     string       `json:"favicon"`
	Tagline     string       `json:"tagline"`
	Header      HeaderTheme  `json:"header"`
	Footer      FooterTheme  `json:"footer"`
	FooterLinks []NavLink    `json:"footer_links"`
	SocialLinks []SocialLink `json:"social_links"`
	Article     ArticleTheme `json:"article"`
}

type HeaderTheme struct {
	BackgroundType  string `json:"background_type"` // "solid" | "gradient"
	BackgroundColor string `json:"background_color"`
	GradientFrom    string `json:"gradient_from"`
	GradientTo      string `json:"gradient_to"`
	TextColor       string `json:"text_color"`
}

type FooterTheme struct {
	BackgroundColor string `json:"background_color"`
	TextColor       string `json:"text_color"`
	Tagline         string `json:"tagline"`
}

type SocialLink struct {
	Platform string `json:"platform"` // twitter, github, linkedin, facebook, instagram, youtube, website
	URL      string `json:"url"`
}

// ArticleTheme uses hide-flags so the zero value (empty theme) keeps the
// default of showing the table of contents and related articles.
type ArticleTheme struct {
	HideToc     bool `json:"hide_toc"`
	HideRelated bool `json:"hide_related"`
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
	Description  string    `db:"description" json:"description"`
	SortOrder    int       `db:"sort_order" json:"sort_order"`
	IsPublished  bool      `db:"is_published" json:"is_published"`
}

type Article struct {
	ID              int       `db:"id" json:"id"`
	CreatedAt       time.Time `db:"created_at" json:"created_at"`
	UpdatedAt       time.Time `db:"updated_at" json:"updated_at"`
	CollectionID    int       `db:"collection_id" json:"collection_id"`
	AuthorID        *int64    `db:"author_id" json:"author_id"`
	AuthorName      *string   `db:"author_name" json:"author_name"`
	Slug            string    `db:"slug" json:"slug"`
	Locale          string    `db:"locale" json:"locale"`
	Title           string    `db:"title" json:"title"`
	Content         string    `db:"content" json:"content"`
	Excerpt         string    `db:"excerpt" json:"excerpt"`
	MetaTitle       string    `db:"meta_title" json:"meta_title"`
	MetaDescription string    `db:"meta_description" json:"meta_description"`
	MetaImageURL    string    `db:"meta_image_url" json:"meta_image_url"`
	SortOrder       int       `db:"sort_order" json:"sort_order"`
	Status          string    `db:"status" json:"status"`
	ViewCount       int       `db:"view_count" json:"view_count"`
	AIEnabled       bool      `db:"ai_enabled" json:"ai_enabled"`
	HelpfulCount    int       `db:"helpful_count" json:"helpful_count"`
	NotHelpfulCount int       `db:"not_helpful_count" json:"not_helpful_count"`
}

// NavLink is a single header navigation link on the public help center pages.
type NavLink struct {
	Label string `json:"label"`
	URL   string `json:"url"`
}

type TreeCollection struct {
	Collection
	Articles     []Article        `json:"articles"`
	Children     []TreeCollection `json:"children"`
	ArticleCount int              `json:"article_count"`
}

type TreeResponse struct {
	HelpCenter HelpCenter       `json:"help_center"`
	Tree       []TreeCollection `json:"tree"`
}

// SearchTermStat is an aggregated public search term for the admin insights panel.
type SearchTermStat struct {
	Query      string `db:"query" json:"query"`
	Count      int    `db:"count" json:"count"`
	NoResults  int    `db:"no_results" json:"no_results"`
	LastSearch string `db:"last_search" json:"last_search"`
}

// Insights bundles the help center analytics shown to admins.
type Insights struct {
	TopSearches    []SearchTermStat `json:"top_searches"`
	NoResultSearch []SearchTermStat `json:"no_result_searches"`
}
