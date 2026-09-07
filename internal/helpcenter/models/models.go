package models

import (
	"encoding/json"
	"time"
)

const (
	ArticleStatusDraft     = "draft"
	ArticleStatusPublished = "published"

	TemplateDocs    = "docs"
	TemplateClassic = "classic"
)

type HelpCenter struct {
	ID              int             `db:"id" json:"id"`
	CreatedAt       time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt       time.Time       `db:"updated_at" json:"updated_at"`
	Name            string          `db:"name" json:"name"`
	Slug            string          `db:"slug" json:"slug"`
	PageTitle       string          `db:"page_title" json:"page_title"`
	MetaDescription string          `db:"meta_description" json:"meta_description"`
	CustomCSS       string          `db:"custom_css" json:"custom_css"`
	CustomJS        string          `db:"custom_js" json:"custom_js"`
	DefaultLocale   string          `db:"default_locale" json:"default_locale"`
	AllowedLocales  json.RawMessage `db:"allowed_locales" json:"allowed_locales"`
	IsActive        bool            `db:"is_active" json:"is_active"`
	Theme           json.RawMessage `db:"theme" json:"theme"`
	CustomDomain    string          `db:"custom_domain" json:"custom_domain"`
	Template        string          `db:"template" json:"template"`
}

// Theme holds the customizable branding for a help center's public pages.
type Theme struct {
	Color        string            `json:"color"`
	LogoURL      string            `json:"logo_url"`
	NavLinks     []NavLink         `json:"nav_links"`
	Favicon      string            `json:"favicon"`
	Tagline      string            `json:"tagline"`
	Header       HeaderTheme       `json:"header"`
	Footer       FooterTheme       `json:"footer"`
	FooterLinks  []NavLink         `json:"footer_links"`
	SocialLinks  []SocialLink      `json:"social_links"`
	Article      ArticleTheme      `json:"article"`
	Layout       LayoutTheme       `json:"layout"`
	Cards        CardTheme         `json:"cards"`
	Announcement AnnouncementTheme `json:"announcement"`
}

// AnnouncementTheme is the dismissible banner shown above the header on every public page.
type AnnouncementTheme struct {
	Text      string `json:"text"`
	LinkLabel string `json:"link_label"`
	LinkURL   string `json:"link_url"`
}

type HeaderTheme struct {
	Heading         string `json:"heading"`
	BackgroundType  string `json:"background_type"` // "solid" | "gradient" | "image"
	BackgroundColor string `json:"background_color"`
	GradientFrom    string `json:"gradient_from"`
	GradientTo      string `json:"gradient_to"`
	BackgroundImage string `json:"background_image"`
	TextColor       string `json:"text_color"`
}

type LayoutTheme struct {
	Collections          string `json:"collections"` // "grid" (default) | "list"
	Columns              int    `json:"columns"`     // grid columns: 2 (default) | 3
	ShowPopularArticles  bool   `json:"show_popular_articles"`
	PopularArticlesLabel string `json:"popular_articles_label"`
}

// CardTheme uses hide-flags for what renders today and show-flags for opt-ins, so an
// empty theme keeps the current look.
type CardTheme struct {
	HideDescription bool   `json:"hide_description"`
	HideCount       bool   `json:"hide_count"`
	ShowAuthors     bool   `json:"show_authors"`
	ShowIconTile    bool   `json:"show_icon_tile"`
	IconPosition    string `json:"icon_position"` // "inline" (default) | "top" | "center"
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

// ArticleTheme uses hide-flags so an empty theme shows everything by default.
type ArticleTheme struct {
	HideToc     bool `json:"hide_toc"`
	HideRelated bool `json:"hide_related"`
	ShowAuthor  bool `json:"show_author"`
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
	Icon         string    `db:"icon" json:"icon"`
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
	AuthorAvatar    *string   `db:"author_avatar" json:"author_avatar"`
	CreatedBy       *int64    `db:"created_by" json:"created_by"`
	CreatedByName   *string   `db:"created_by_name" json:"created_by_name"`
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
	// EmbeddedFingerprint is internal AI-index state; mapped so RETURNING * scans in
	// safe-mode transactions don't fail, but never exposed in API responses.
	EmbeddedFingerprint string `db:"embedded_fingerprint" json:"-"`
	SearchTSV           string `db:"search_tsv" json:"-"`
	HelpfulCount    int    `db:"helpful_count" json:"helpful_count"`
	NotHelpfulCount int    `db:"not_helpful_count" json:"not_helpful_count"`
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
	Authors      []ArticleAuthor  `json:"authors"`
	AuthorCount  int              `json:"author_count"`
}

// ArticleAuthor is a distinct article author shown on collection cards.
type ArticleAuthor struct {
	Name   string `json:"name"`
	Avatar string `json:"avatar"`
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

// Locale is a selectable help center language.
type Locale struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

// Insights bundles the help center analytics shown to admins.
type Insights struct {
	TopSearches    []SearchTermStat `json:"top_searches"`
	NoResultSearch []SearchTermStat `json:"no_result_searches"`
}

// DefaultTheme enables the show-flags for elements that must keep rendering when a stored theme predates the flag.
func DefaultTheme() Theme {
	return Theme{
		Color:   "#1f93ff",
		Layout:  LayoutTheme{ShowPopularArticles: true},
		Cards:   CardTheme{ShowIconTile: true},
		Article: ArticleTheme{ShowAuthor: true},
	}
}
