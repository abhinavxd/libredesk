// Package helpcenter manages help centers, collections, and articles.
package helpcenter

import (
	"database/sql"
	"embed"
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/helpcenter/models"
	"github.com/abhinavxd/libredesk/internal/stringutil"
	"github.com/jmoiron/sqlx"
	"github.com/knadh/go-i18n"
	"github.com/microcosm-cc/bluemonday"
	"github.com/zerodha/logf"
)

const (
	maxCollectionDepth = 3
	defaultLocale      = "en"
	defaultAccentColor = "#1f93ff"
	excerptLimit       = 160
)

var (
	//go:embed queries.sql
	efs embed.FS

	// reservedSlugs collide with public help center routes.
	reservedSlugs = []string{"articles", "search", "api", "sitemap.xml"}

	hexColorRe = regexp.MustCompile(`^#(?:[0-9a-fA-F]{3,4}|[0-9a-fA-F]{6}|[0-9a-fA-F]{8})$`)

	ilikeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

	youtubeEmbedRe = regexp.MustCompile(`^https://(www\.)?(youtube\.com|youtube-nocookie\.com)/embed/[\w-]+`)

	articleButtonClassRe = regexp.MustCompile(`^hc-button$`)

	textAlignRe = regexp.MustCompile(`^(left|center|right|justify)$`)

	// articleSanitizer strips unsafe HTML from article content since it renders raw on public pages.
	articleSanitizer = buildArticleSanitizer()
)

// ArticleIndexer syncs article content into the AI embedding index.
type ArticleIndexer interface {
	ReindexHelpArticle(articleID int)
	RemoveHelpArticleEmbeddings(articleID int) error
}

type HelpCenterRequest struct {
	Name           string          `json:"name"`
	Slug           string          `json:"slug"`
	PageTitle      string          `json:"page_title"`
	HeaderText     string          `json:"header_text"`
	LogoURL        string          `json:"logo_url"`
	Color          string          `json:"color"`
	NavLinks       json.RawMessage `json:"nav_links"`
	CustomCSS      string          `json:"custom_css"`
	CustomJS       string          `json:"custom_js"`
	DefaultLocale  string          `json:"default_locale"`
	AllowedLocales json.RawMessage `json:"allowed_locales"`
	Theme          json.RawMessage `json:"theme"`
}

type CollectionRequest struct {
	Slug        string `json:"slug"`
	ParentID    *int   `json:"parent_id"`
	Locale      string `json:"locale"`
	Name        string `json:"name"`
	Description string `json:"description"`
	SortOrder   int    `json:"sort_order"`
	IsPublished bool   `json:"is_published"`
}

type ArticleRequest struct {
	Slug            string `json:"slug"`
	Locale          string `json:"locale"`
	Title           string `json:"title"`
	Content         string `json:"content"`
	Excerpt         string `json:"excerpt"`
	MetaTitle       string `json:"meta_title"`
	MetaDescription string `json:"meta_description"`
	MetaImageURL    string `json:"meta_image_url"`
	SortOrder       int    `json:"sort_order"`
	Status          string `json:"status"`
	AIEnabled       bool   `json:"ai_enabled"`
	CollectionID    *int   `json:"collection_id,omitempty"`
	AuthorID        *int64 `json:"-"`
}

type Manager struct {
	q       queries
	lo      *logf.Logger
	i18n    *i18n.I18n
	indexer ArticleIndexer
}

// Opts contains options for initializing the Manager.
type Opts struct {
	DB      *sqlx.DB
	Lo      *logf.Logger
	I18n    *i18n.I18n
	Indexer ArticleIndexer
}

// queries contains prepared SQL queries.
type queries struct {
	GetAllHelpCenters   *sqlx.Stmt `query:"get-all-help-centers"`
	GetHelpCenterByID   *sqlx.Stmt `query:"get-help-center-by-id"`
	GetHelpCenterBySlug *sqlx.Stmt `query:"get-help-center-by-slug"`
	InsertHelpCenter    *sqlx.Stmt `query:"insert-help-center"`
	UpdateHelpCenter    *sqlx.Stmt `query:"update-help-center"`
	ToggleHelpCenter    *sqlx.Stmt `query:"toggle-help-center-active"`
	DeleteHelpCenter    *sqlx.Stmt `query:"delete-help-center"`

	GetCollectionsByHelpCenter *sqlx.Stmt `query:"get-collections-by-help-center"`
	GetCollectionByID          *sqlx.Stmt `query:"get-collection-by-id"`
	InsertCollection           *sqlx.Stmt `query:"insert-collection"`
	UpdateCollection           *sqlx.Stmt `query:"update-collection"`
	ToggleCollectionPublished  *sqlx.Stmt `query:"toggle-collection-published"`
	DeleteCollection           *sqlx.Stmt `query:"delete-collection"`

	GetArticlesByCollection       *sqlx.Stmt `query:"get-articles-by-collection"`
	GetArticleByID                *sqlx.Stmt `query:"get-article-by-id"`
	InsertArticle                 *sqlx.Stmt `query:"insert-article"`
	UpdateArticle                 *sqlx.Stmt `query:"update-article"`
	ArticleSlugExistsInHelpCenter *sqlx.Stmt `query:"article-slug-exists-in-help-center"`
	UpdateArticleStatus           *sqlx.Stmt `query:"update-article-status"`
	DeleteArticle                 *sqlx.Stmt `query:"delete-article"`

	GetHelpCenterTreeData            *sqlx.Stmt `query:"get-help-center-tree-data"`
	GetPublicTreeData                *sqlx.Stmt `query:"get-public-tree-data"`
	GetPublishedArticleBySlug        *sqlx.Stmt `query:"get-published-article-by-slug"`
	GetPublishedArticles             *sqlx.Stmt `query:"get-published-articles"`
	GetPublishedArticlesByCollection *sqlx.Stmt `query:"get-published-articles-by-collection"`
	SearchPublishedArticles          *sqlx.Stmt `query:"search-published-articles"`
	IncrementArticleViewCount        *sqlx.Stmt `query:"increment-article-view-count"`
	IncrementHelpCenterViewCount     *sqlx.Stmt `query:"increment-help-center-view-count"`

	InsertArticleFeedback  *sqlx.Stmt `query:"insert-article-feedback"`
	InsertSearchQuery      *sqlx.Stmt `query:"insert-search-query"`
	GetTopSearchTerms      *sqlx.Stmt `query:"get-top-search-terms"`
	GetNoResultSearchTerms *sqlx.Stmt `query:"get-no-result-search-terms"`
}

// New creates and returns a new instance of the Manager.
func New(opts Opts) (*Manager, error) {
	var q queries
	if err := dbutil.ScanSQLFile("queries.sql", &q, opts.DB, efs); err != nil {
		return nil, err
	}
	return &Manager{
		q:       q,
		lo:      opts.Lo,
		i18n:    opts.I18n,
		indexer: opts.Indexer,
	}, nil
}

// GetAllHelpCenters retrieves all help centers.
func (m *Manager) GetAllHelpCenters() ([]models.HelpCenter, error) {
	var helpCenters = make([]models.HelpCenter, 0)
	if err := m.q.GetAllHelpCenters.Select(&helpCenters); err != nil {
		m.lo.Error("error fetching help centers", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return helpCenters, nil
}

// GetHelpCenterByID retrieves a help center by ID.
func (m *Manager) GetHelpCenterByID(id int) (models.HelpCenter, error) {
	var hc models.HelpCenter
	if err := m.q.GetHelpCenterByID.Get(&hc, id); err != nil {
		if err == sql.ErrNoRows {
			return hc, envelope.NewError(envelope.NotFoundError, m.i18n.T("globals.messages.notFound"), nil)
		}
		m.lo.Error("error fetching help center", "error", err, "id", id)
		return hc, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return hc, nil
}

// GetHelpCenterBySlug retrieves a help center by slug.
func (m *Manager) GetHelpCenterBySlug(slug string) (models.HelpCenter, error) {
	var hc models.HelpCenter
	if err := m.q.GetHelpCenterBySlug.Get(&hc, slug); err != nil {
		if err == sql.ErrNoRows {
			return hc, envelope.NewError(envelope.NotFoundError, m.i18n.T("globals.messages.notFound"), nil)
		}
		m.lo.Error("error fetching help center by slug", "error", err, "slug", slug)
		return hc, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return hc, nil
}

// CreateHelpCenter creates a new help center.
func (m *Manager) CreateHelpCenter(req HelpCenterRequest) (models.HelpCenter, error) {
	var hc models.HelpCenter
	req = normalizeHelpCenterRequest(req)
	if err := m.validateHelpCenterSlug(req.Slug); err != nil {
		return hc, err
	}
	if err := m.validateColor(req.Color); err != nil {
		return hc, err
	}
	if err := m.q.InsertHelpCenter.Get(&hc, req.Name, req.Slug, req.PageTitle, req.HeaderText, req.LogoURL, req.Color, req.NavLinks, req.CustomCSS, req.CustomJS, req.DefaultLocale, req.AllowedLocales, req.Theme); err != nil {
		if dbutil.IsUniqueViolationError(err) {
			return hc, envelope.NewError(envelope.ConflictError, m.i18n.T("globals.messages.errorAlreadyExists"), nil)
		}
		m.lo.Error("error creating help center", "error", err)
		return hc, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return hc, nil
}

// UpdateHelpCenter updates a help center.
func (m *Manager) UpdateHelpCenter(id int, req HelpCenterRequest) (models.HelpCenter, error) {
	var hc models.HelpCenter
	req = normalizeHelpCenterRequest(req)
	if err := m.validateHelpCenterSlug(req.Slug); err != nil {
		return hc, err
	}
	if err := m.validateColor(req.Color); err != nil {
		return hc, err
	}
	if err := m.q.UpdateHelpCenter.Get(&hc, id, req.Name, req.Slug, req.PageTitle, req.HeaderText, req.LogoURL, req.Color, req.NavLinks, req.CustomCSS, req.CustomJS, req.DefaultLocale, req.AllowedLocales, req.Theme); err != nil {
		if dbutil.IsUniqueViolationError(err) {
			return hc, envelope.NewError(envelope.ConflictError, m.i18n.T("globals.messages.errorAlreadyExists"), nil)
		}
		m.lo.Error("error updating help center", "error", err, "id", id)
		return hc, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return hc, nil
}

// ToggleHelpCenterActive flips a help center between live and paused.
func (m *Manager) ToggleHelpCenterActive(id int) (models.HelpCenter, error) {
	var hc models.HelpCenter
	if err := m.q.ToggleHelpCenter.Get(&hc, id); err != nil {
		m.lo.Error("error toggling help center active status", "error", err, "id", id)
		return hc, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return hc, nil
}

// DeleteHelpCenter deletes a help center by ID.
func (m *Manager) DeleteHelpCenter(id int) error {
	if _, err := m.q.DeleteHelpCenter.Exec(id); err != nil {
		m.lo.Error("error deleting help center", "error", err, "id", id)
		return envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return nil
}

// GetCollectionsByHelpCenter retrieves all collections for a help center.
func (m *Manager) GetCollectionsByHelpCenter(helpCenterID int) ([]models.Collection, error) {
	var collections = make([]models.Collection, 0)
	if err := m.q.GetCollectionsByHelpCenter.Select(&collections, helpCenterID); err != nil {
		m.lo.Error("error fetching collections", "error", err, "help_center_id", helpCenterID)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return collections, nil
}

// GetCollectionByID retrieves a collection by ID.
func (m *Manager) GetCollectionByID(id int) (models.Collection, error) {
	var collection models.Collection
	if err := m.q.GetCollectionByID.Get(&collection, id); err != nil {
		if err == sql.ErrNoRows {
			return collection, envelope.NewError(envelope.NotFoundError, m.i18n.T("globals.messages.notFound"), nil)
		}
		m.lo.Error("error fetching collection", "error", err, "id", id)
		return collection, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return collection, nil
}

// CreateCollection creates a new collection.
func (m *Manager) CreateCollection(helpCenterID int, req CollectionRequest) (models.Collection, error) {
	var collection models.Collection
	if err := m.validateSlug(req.Slug); err != nil {
		return collection, err
	}
	if req.ParentID != nil {
		if err := m.validateCollectionParent(*req.ParentID, 0, helpCenterID); err != nil {
			return collection, err
		}
	}
	if req.Locale == "" {
		req.Locale = defaultLocale
	}
	if err := m.q.InsertCollection.Get(&collection, helpCenterID, req.Slug, req.ParentID, req.Locale, req.Name, req.Description, req.SortOrder, req.IsPublished); err != nil {
		if dbutil.IsUniqueViolationError(err) {
			return collection, envelope.NewError(envelope.ConflictError, m.i18n.T("globals.messages.errorAlreadyExists"), nil)
		}
		m.lo.Error("error creating collection", "error", err)
		return collection, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return collection, nil
}

// UpdateCollection updates a collection.
func (m *Manager) UpdateCollection(id int, req CollectionRequest) (models.Collection, error) {
	var collection models.Collection
	if err := m.validateSlug(req.Slug); err != nil {
		return collection, err
	}
	if req.ParentID != nil {
		existing, err := m.GetCollectionByID(id)
		if err != nil {
			return collection, err
		}
		if err := m.validateCollectionParent(*req.ParentID, id, existing.HelpCenterID); err != nil {
			return collection, err
		}
	}
	if req.Locale == "" {
		req.Locale = defaultLocale
	}
	if err := m.q.UpdateCollection.Get(&collection, id, req.Slug, req.ParentID, req.Locale, req.Name, req.Description, req.SortOrder, req.IsPublished); err != nil {
		if dbutil.IsUniqueViolationError(err) {
			return collection, envelope.NewError(envelope.ConflictError, m.i18n.T("globals.messages.errorAlreadyExists"), nil)
		}
		m.lo.Error("error updating collection", "error", err, "id", id)
		return collection, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return collection, nil
}

// ToggleCollectionPublished toggles the published status of a collection.
func (m *Manager) ToggleCollectionPublished(id int) (models.Collection, error) {
	var collection models.Collection
	if err := m.q.ToggleCollectionPublished.Get(&collection, id); err != nil {
		m.lo.Error("error toggling collection published status", "error", err, "id", id)
		return collection, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return collection, nil
}

// DeleteCollection deletes a collection by ID.
func (m *Manager) DeleteCollection(id int) error {
	if _, err := m.q.DeleteCollection.Exec(id); err != nil {
		m.lo.Error("error deleting collection", "error", err, "id", id)
		return envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return nil
}

// GetArticlesByCollection retrieves all articles for a collection.
func (m *Manager) GetArticlesByCollection(collectionID int) ([]models.Article, error) {
	var articles = make([]models.Article, 0)
	if err := m.q.GetArticlesByCollection.Select(&articles, collectionID); err != nil {
		m.lo.Error("error fetching articles", "error", err, "collection_id", collectionID)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return articles, nil
}

// GetArticleByID retrieves an article by ID.
func (m *Manager) GetArticleByID(id int) (models.Article, error) {
	var article models.Article
	if err := m.q.GetArticleByID.Get(&article, id); err != nil {
		if err == sql.ErrNoRows {
			return article, envelope.NewError(envelope.NotFoundError, m.i18n.T("globals.messages.notFound"), nil)
		}
		m.lo.Error("error fetching article", "error", err, "id", id)
		return article, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return article, nil
}

// CreateArticle creates a new article.
func (m *Manager) CreateArticle(collectionID int, req ArticleRequest) (models.Article, error) {
	var article models.Article
	if !isValidArticleStatus(req.Status) {
		return article, envelope.NewError(envelope.InputError, m.i18n.T("helpCenter.invalidStatus"), nil)
	}
	if err := m.validateSlug(req.Slug); err != nil {
		return article, err
	}
	if req.Locale == "" {
		req.Locale = defaultLocale
	}
	slug, err := m.uniqueArticleSlug(collectionID, req.Slug, req.Locale)
	if err != nil {
		return article, err
	}
	req.Slug = slug
	req.Content = articleSanitizer.Sanitize(req.Content)
	req.Excerpt = resolveExcerpt(req.Excerpt, req.Content)
	if err := m.q.InsertArticle.Get(&article, collectionID, req.AuthorID, req.Slug, req.Locale, req.Title, req.Content, req.Excerpt, req.MetaTitle, req.MetaDescription, req.MetaImageURL, req.SortOrder, req.Status, req.AIEnabled); err != nil {
		if dbutil.IsUniqueViolationError(err) {
			return article, envelope.NewError(envelope.ConflictError, m.i18n.T("globals.messages.errorAlreadyExists"), nil)
		}
		m.lo.Error("error creating article", "error", err)
		return article, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	m.reindexArticle(article.ID)
	return article, nil
}

// UpdateArticle updates an article, optionally moving it to another collection.
func (m *Manager) UpdateArticle(id int, req ArticleRequest) (models.Article, error) {
	var article models.Article
	if !isValidArticleStatus(req.Status) {
		return article, envelope.NewError(envelope.InputError, m.i18n.T("helpCenter.invalidStatus"), nil)
	}
	if err := m.validateSlug(req.Slug); err != nil {
		return article, err
	}
	if req.Locale == "" {
		req.Locale = defaultLocale
	}
	req.Content = articleSanitizer.Sanitize(req.Content)
	req.Excerpt = resolveExcerpt(req.Excerpt, req.Content)
	if err := m.q.UpdateArticle.Get(&article, id, req.Slug, req.Locale, req.Title, req.Content, req.SortOrder, req.Status, req.AIEnabled, req.CollectionID, req.Excerpt, req.MetaTitle, req.MetaDescription, req.MetaImageURL); err != nil {
		if dbutil.IsUniqueViolationError(err) {
			return article, envelope.NewError(envelope.ConflictError, m.i18n.T("globals.messages.errorAlreadyExists"), nil)
		}
		m.lo.Error("error updating article", "error", err, "id", id)
		return article, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	m.reindexArticle(article.ID)
	return article, nil
}

// UpdateArticleStatus updates the status of an article.
func (m *Manager) UpdateArticleStatus(id int, status string) (models.Article, error) {
	var article models.Article
	if !isValidArticleStatus(status) {
		return article, envelope.NewError(envelope.InputError, m.i18n.T("helpCenter.invalidStatus"), nil)
	}
	if err := m.q.UpdateArticleStatus.Get(&article, id, status); err != nil {
		m.lo.Error("error updating article status", "error", err, "id", id)
		return article, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	m.reindexArticle(article.ID)
	return article, nil
}

// DeleteArticle deletes an article by ID.
func (m *Manager) DeleteArticle(id int) error {
	if _, err := m.q.DeleteArticle.Exec(id); err != nil {
		m.lo.Error("error deleting article", "error", err, "id", id)
		return envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	if m.indexer != nil {
		if err := m.indexer.RemoveHelpArticleEmbeddings(id); err != nil {
			m.lo.Error("error removing article embeddings", "error", err, "id", id)
		}
	}
	return nil
}

// GetHelpCenterTree returns the complete tree structure for a help center, filtered to locale (empty = all).
func (m *Manager) GetHelpCenterTree(helpCenterID int, locale string) (models.TreeResponse, error) {
	helpCenter, err := m.GetHelpCenterByID(helpCenterID)
	if err != nil {
		return models.TreeResponse{}, err
	}
	rows, err := m.q.GetHelpCenterTreeData.Query(helpCenterID, locale)
	if err != nil {
		m.lo.Error("error fetching tree data", "error", err, "help_center_id", helpCenterID)
		return models.TreeResponse{}, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	defer rows.Close()

	tree, err := m.scanTree(rows)
	if err != nil {
		return models.TreeResponse{}, err
	}
	return models.TreeResponse{HelpCenter: helpCenter, Tree: tree}, nil
}

// GetPublicTree returns the published-only tree for a help center by slug, filtered to locale (empty = all).
func (m *Manager) GetPublicTree(slug, locale string) (models.TreeResponse, error) {
	helpCenter, err := m.GetHelpCenterBySlug(slug)
	if err != nil {
		return models.TreeResponse{}, err
	}
	rows, err := m.q.GetPublicTreeData.Query(helpCenter.ID, locale)
	if err != nil {
		m.lo.Error("error fetching public tree data", "error", err, "help_center_id", helpCenter.ID)
		return models.TreeResponse{}, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	defer rows.Close()

	tree, err := m.scanTree(rows)
	if err != nil {
		return models.TreeResponse{}, err
	}
	return models.TreeResponse{HelpCenter: helpCenter, Tree: tree}, nil
}

// GetPublishedArticle retrieves a published article by help center slug and article slug, preferring locale (empty = any).
func (m *Manager) GetPublishedArticle(helpCenterSlug, articleSlug, locale string) (models.Article, error) {
	var article models.Article
	if err := m.q.GetPublishedArticleBySlug.Get(&article, helpCenterSlug, articleSlug, locale); err != nil {
		if err == sql.ErrNoRows {
			return article, envelope.NewError(envelope.NotFoundError, m.i18n.T("globals.messages.notFound"), nil)
		}
		m.lo.Error("error fetching published article", "error", err, "help_center_slug", helpCenterSlug, "article_slug", articleSlug)
		return article, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return article, nil
}

// GetPopularArticles returns the most viewed published articles for a help center, filtered to locale (empty = all).
func (m *Manager) GetPopularArticles(helpCenterSlug, locale string, limit int) ([]models.Article, error) {
	var articles = make([]models.Article, 0)
	if err := m.q.GetPublishedArticles.Select(&articles, helpCenterSlug, locale, limit); err != nil {
		m.lo.Error("error fetching popular articles", "error", err, "help_center_slug", helpCenterSlug)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return articles, nil
}

// GetPublishedArticlesByCollection returns published articles in a collection, excluding one article.
func (m *Manager) GetPublishedArticlesByCollection(collectionID, excludeArticleID, limit int) ([]models.Article, error) {
	var articles = make([]models.Article, 0)
	if err := m.q.GetPublishedArticlesByCollection.Select(&articles, collectionID, excludeArticleID, limit); err != nil {
		m.lo.Error("error fetching collection articles", "error", err, "collection_id", collectionID)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return articles, nil
}

// SearchPublishedArticles searches published articles in a help center, content trimmed to an excerpt, filtered to locale (empty = all).
func (m *Manager) SearchPublishedArticles(helpCenterSlug, query, locale string, limit int) ([]models.Article, error) {
	var articles = make([]models.Article, 0)
	query = ilikeEscaper.Replace(query)
	if err := m.q.SearchPublishedArticles.Select(&articles, helpCenterSlug, query, limit, locale); err != nil {
		m.lo.Error("error searching published articles", "error", err, "help_center_slug", helpCenterSlug)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return articles, nil
}

// IncrementArticleViewCount increments the view count of an article.
func (m *Manager) IncrementArticleViewCount(id int) {
	if _, err := m.q.IncrementArticleViewCount.Exec(id); err != nil {
		m.lo.Error("error incrementing article view count", "error", err, "id", id)
	}
}

// IncrementHelpCenterViewCount increments the view count of a help center.
func (m *Manager) IncrementHelpCenterViewCount(id int) {
	if _, err := m.q.IncrementHelpCenterViewCount.Exec(id); err != nil {
		m.lo.Error("error incrementing help center view count", "error", err, "id", id)
	}
}

// RecordArticleFeedback stores a reader's helpful/not-helpful vote for a published article.
func (m *Manager) RecordArticleFeedback(articleID int, isHelpful bool) error {
	if _, err := m.q.InsertArticleFeedback.Exec(articleID, isHelpful); err != nil {
		m.lo.Error("error recording article feedback", "error", err, "article_id", articleID)
		return envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return nil
}

// LogSearch records a public search term and how many results it returned.
func (m *Manager) LogSearch(helpCenterID int, query string, resultsCount int) {
	query = strings.TrimSpace(query)
	if query == "" {
		return
	}
	if _, err := m.q.InsertSearchQuery.Exec(helpCenterID, query, resultsCount); err != nil {
		m.lo.Error("error logging search query", "error", err, "help_center_id", helpCenterID)
	}
}

// GetInsights returns the top and zero-result search terms for a help center.
func (m *Manager) GetInsights(helpCenterID, limit int) (models.Insights, error) {
	var insights models.Insights
	insights.TopSearches = make([]models.SearchTermStat, 0)
	insights.NoResultSearch = make([]models.SearchTermStat, 0)
	if err := m.q.GetTopSearchTerms.Select(&insights.TopSearches, helpCenterID, limit); err != nil {
		m.lo.Error("error fetching top search terms", "error", err, "help_center_id", helpCenterID)
		return insights, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	if err := m.q.GetNoResultSearchTerms.Select(&insights.NoResultSearch, helpCenterID, limit); err != nil {
		m.lo.Error("error fetching no-result search terms", "error", err, "help_center_id", helpCenterID)
		return insights, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return insights, nil
}

// scanTree scans combined collection/article rows and assembles the nested tree.
func (m *Manager) scanTree(rows *sql.Rows) ([]models.TreeCollection, error) {
	collections := make(map[int]*models.TreeCollection)
	var rootOrder []int

	for rows.Next() {
		var (
			itemType     string
			id           int
			createdAt    time.Time
			updatedAt    time.Time
			helpCenterID int
			slug         string
			parentID     *int
			locale       string
			name         string
			description  *string
			sortOrder    int
			isPublished  *bool
			collectionID *int
			title        *string
			content      *string
			status       *string
			viewCount    *int
			aiEnabled    *bool
		)
		if err := rows.Scan(&itemType, &id, &createdAt, &updatedAt, &helpCenterID, &slug, &parentID, &locale, &name, &description, &sortOrder, &isPublished, &collectionID, &title, &content, &status, &viewCount, &aiEnabled); err != nil {
			m.lo.Error("error scanning tree data", "error", err)
			return nil, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
		}

		switch itemType {
		case "collection":
			desc := ""
			if description != nil {
				desc = *description
			}
			collections[id] = &models.TreeCollection{
				Collection: models.Collection{
					ID:           id,
					CreatedAt:    createdAt,
					UpdatedAt:    updatedAt,
					HelpCenterID: helpCenterID,
					Slug:         slug,
					ParentID:     parentID,
					Locale:       locale,
					Name:         name,
					Description:  desc,
					SortOrder:    sortOrder,
					IsPublished:  isPublished != nil && *isPublished,
				},
				Articles: make([]models.Article, 0),
				Children: make([]models.TreeCollection, 0),
			}
			rootOrder = append(rootOrder, id)
		case "article":
			if collectionID == nil {
				continue
			}
			collection, exists := collections[*collectionID]
			if !exists {
				continue
			}
			article := models.Article{
				ID:           id,
				CreatedAt:    createdAt,
				UpdatedAt:    updatedAt,
				CollectionID: *collectionID,
				Slug:         slug,
				Locale:       locale,
				SortOrder:    sortOrder,
			}
			if title != nil {
				article.Title = *title
			}
			if content != nil {
				article.Content = *content
			}
			if status != nil {
				article.Status = *status
			}
			if viewCount != nil {
				article.ViewCount = *viewCount
			}
			if aiEnabled != nil {
				article.AIEnabled = *aiEnabled
			}
			collection.Articles = append(collection.Articles, article)
		}
	}
	if err := rows.Err(); err != nil {
		m.lo.Error("error iterating tree data", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}

	var buildTree func(parentID *int) []models.TreeCollection
	buildTree = func(parentID *int) []models.TreeCollection {
		children := make([]models.TreeCollection, 0)
		for _, id := range rootOrder {
			col := collections[id]
			matches := (col.ParentID == nil && parentID == nil) ||
				(col.ParentID != nil && parentID != nil && *col.ParentID == *parentID)
			if matches {
				col.Children = buildTree(&col.ID)
				children = append(children, *col)
			}
		}
		return children
	}
	tree := buildTree(nil)

	var fillCounts func(cols []models.TreeCollection) int
	fillCounts = func(cols []models.TreeCollection) int {
		total := 0
		for i := range cols {
			cols[i].ArticleCount = len(cols[i].Articles) + fillCounts(cols[i].Children)
			total += cols[i].ArticleCount
		}
		return total
	}
	fillCounts(tree)

	return tree, nil
}

// reindexArticle asks the AI indexer to re-sync an article's embeddings.
func (m *Manager) reindexArticle(id int) {
	if m.indexer != nil {
		m.indexer.ReindexHelpArticle(id)
	}
}

// validateCollectionParent rejects parents that nest deeper than maxCollectionDepth,
// belong to another help center, or would make the collection its own ancestor.
func (m *Manager) validateCollectionParent(parentID, selfID, helpCenterID int) error {
	depth := 2
	currentID := parentID
	for {
		if currentID == selfID {
			return envelope.NewError(envelope.InputError, m.i18n.T("helpCenter.invalidParent"), nil)
		}
		parent, err := m.GetCollectionByID(currentID)
		if err != nil {
			return err
		}
		if parent.HelpCenterID != helpCenterID {
			return envelope.NewError(envelope.InputError, m.i18n.T("helpCenter.invalidParent"), nil)
		}
		if parent.ParentID == nil {
			break
		}
		currentID = *parent.ParentID
		depth++
		if depth > maxCollectionDepth {
			return envelope.NewError(envelope.InputError, m.i18n.T("helpCenter.maxDepthReached"), nil)
		}
	}
	return nil
}

// uniqueArticleSlug appends a numeric suffix until the slug is unique within the collection's help center.
func (m *Manager) uniqueArticleSlug(collectionID int, slug, locale string) (string, error) {
	candidate := slug
	for i := 2; ; i++ {
		var exists bool
		if err := m.q.ArticleSlugExistsInHelpCenter.Get(&exists, collectionID, candidate, locale); err != nil {
			m.lo.Error("error checking article slug uniqueness", "error", err, "slug", candidate)
			return "", envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
		}
		if !exists {
			return candidate, nil
		}
		candidate = fmt.Sprintf("%s-%d", slug, i)
	}
}

// normalizeHelpCenterRequest fills defaults for optional fields and keeps the language config consistent.
func normalizeHelpCenterRequest(req HelpCenterRequest) HelpCenterRequest {
	if req.DefaultLocale == "" {
		req.DefaultLocale = defaultLocale
	}
	if req.Color == "" {
		req.Color = defaultAccentColor
	}
	if len(req.NavLinks) == 0 {
		req.NavLinks = json.RawMessage("[]")
	}

	locales := []string{}
	if len(req.AllowedLocales) > 0 {
		_ = json.Unmarshal(req.AllowedLocales, &locales)
	}
	locales = normalizeLocales(locales, req.DefaultLocale)
	if b, err := json.Marshal(locales); err == nil {
		req.AllowedLocales = b
	}
	req.Theme = normalizeTheme(req.Theme)
	return req
}

// normalizeTheme drops any theme color that isn't a valid hex code before it can reach
// the injected CSS. Invalid JSON collapses to '{}'.
func normalizeTheme(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return json.RawMessage("{}")
	}
	var t models.Theme
	if err := json.Unmarshal(raw, &t); err != nil {
		return json.RawMessage("{}")
	}
	t.Header.BackgroundColor = sanitizeHexColor(t.Header.BackgroundColor)
	t.Header.GradientFrom = sanitizeHexColor(t.Header.GradientFrom)
	t.Header.GradientTo = sanitizeHexColor(t.Header.GradientTo)
	t.Header.TextColor = sanitizeHexColor(t.Header.TextColor)
	t.Footer.BackgroundColor = sanitizeHexColor(t.Footer.BackgroundColor)
	t.Footer.TextColor = sanitizeHexColor(t.Footer.TextColor)
	if t.Header.BackgroundType != "gradient" && t.Header.BackgroundType != "solid" {
		t.Header.BackgroundType = ""
	}
	b, err := json.Marshal(t)
	if err != nil {
		return json.RawMessage("{}")
	}
	return b
}

func sanitizeHexColor(c string) string {
	if !hexColorRe.MatchString(c) {
		return ""
	}
	return c
}

// normalizeLocales trims/dedupes locale codes and guarantees the default locale is present and first.
func normalizeLocales(locales []string, defaultLocale string) []string {
	seen := map[string]bool{}
	out := []string{defaultLocale}
	seen[defaultLocale] = true
	for _, l := range locales {
		l = strings.TrimSpace(l)
		if l == "" || seen[l] {
			continue
		}
		seen[l] = true
		out = append(out, l)
	}
	return out
}

// validateSlug rejects empty slugs.
func (m *Manager) validateSlug(slug string) error {
	if slug == "" {
		return envelope.NewError(envelope.InputError, m.i18n.T("helpCenter.invalidSlug"), nil)
	}
	return nil
}

// validateHelpCenterSlug rejects empty slugs and slugs that collide with public help center routes.
func (m *Manager) validateHelpCenterSlug(slug string) error {
	if slug == "" || slices.Contains(reservedSlugs, slug) {
		return envelope.NewError(envelope.InputError, m.i18n.T("helpCenter.invalidSlug"), nil)
	}
	return nil
}

// validateColor rejects accent colors that are not hex color codes.
func (m *Manager) validateColor(color string) error {
	if !hexColorRe.MatchString(color) {
		return envelope.NewError(envelope.InputError, m.i18n.T("helpCenter.invalidColor"), nil)
	}
	return nil
}

// isValidArticleStatus checks if the given status is valid.
func isValidArticleStatus(status string) bool {
	return status == models.ArticleStatusDraft || status == models.ArticleStatusPublished || status == models.ArticleStatusArchived
}

// resolveExcerpt returns the given excerpt, or a plain-text excerpt derived from content when empty.
func resolveExcerpt(excerpt, htmlContent string) string {
	if strings.TrimSpace(excerpt) != "" {
		return strings.TrimSpace(excerpt)
	}
	text := strings.Join(strings.Fields(stringutil.HTML2Text(htmlContent)), " ")
	runes := []rune(text)
	if len(runes) <= excerptLimit {
		return text
	}
	text = string(runes[:excerptLimit])
	if i := strings.LastIndex(text, " "); i > 0 {
		text = text[:i]
	}
	return text
}

// buildArticleSanitizer returns the HTML sanitization policy for article content.
func buildArticleSanitizer() *bluemonday.Policy {
	p := bluemonday.UGCPolicy()
	p.AllowAttrs("class").OnElements("img", "pre", "code", "div", "span", "p")
	p.AllowAttrs("class").Matching(articleButtonClassRe).OnElements("a")
	// Collapsible sections render as native <details>/<summary>.
	p.AllowElements("details", "summary")
	p.AllowAttrs("class").OnElements("details", "summary")
	p.AllowStyles("text-align").Matching(textAlignRe).OnElements("p", "h1", "h2", "h3", "h4")
	p.AllowAttrs("width", "height").OnElements("img")
	p.AllowStyles("width", "height", "max-width").OnElements("img")
	p.AllowStyles("border", "width", "margin", "table-layout", "border-collapse", "border-radius",
		"box-sizing", "min-width", "padding", "vertical-align", "background-color", "color",
		"font-weight", "text-align", "position").OnElements("table", "td", "th")
	// YouTube embeds as rendered by the tiptap Youtube extension.
	p.AllowAttrs("data-youtube-video").OnElements("div")
	p.AllowAttrs("src").Matching(youtubeEmbedRe).OnElements("iframe")
	p.AllowAttrs("width", "height", "allowfullscreen", "frameborder", "allow", "referrerpolicy", "start").OnElements("iframe")
	return p
}
