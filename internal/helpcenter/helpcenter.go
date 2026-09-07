// Package helpcenter manages help centers, collections, and articles.
package helpcenter

import (
	"database/sql"
	"embed"
	"encoding/json"
	"fmt"
	"maps"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

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
	maxCardAuthors     = 3
	maxSearchQueryLen  = 200
	maxSlugLen         = 200
	maxNameLen         = 200
	maxPageTitleLen    = 200
	maxMetaDescLen     = 500

	// minSearchQueryLen mirrors the typeahead's floor; shorter terms miss the trigram index and seq-scan.
	minSearchQueryLen = 2

	// searchLogRetentionDays bounds both what insights read and what the cleaner keeps.
	searchLogRetentionDays = 90
)

var (
	//go:embed queries.sql
	efs embed.FS

	// reservedSlugs collide with public help center routes.
	reservedSlugs = []string{"articles", "search", "api", "sitemap.xml"}

	headerBackgroundTypes = []string{"solid", "gradient", "image"}

	helpCenterTemplates = []string{models.TemplateDocs, models.TemplateClassic}

	cardIconPositions = []string{"inline", "top", "center"}

	// First entry is the fallback for an unknown platform.
	socialPlatforms = []string{"website", "twitter", "github", "linkedin", "facebook", "instagram", "youtube"}

	hexColorRe = regexp.MustCompile(`^#(?:[0-9a-fA-F]{3,4}|[0-9a-fA-F]{6}|[0-9a-fA-F]{8})$`)

	// assetURLRe excludes quotes, parens, whitespace, angle brackets and CSS punctuation, else a value escapes url(), <style> or an attribute.
	assetURLRe = regexp.MustCompile(`^(?:https?://[^"'()\s\\<>;{}]+|/[^"'()\s\\<>;{}]*)$`)

	// iconNameRe matches lucide icon slugs, e.g. "rocket", "user-check".
	iconNameRe = regexp.MustCompile(`^[a-z0-9-]{1,64}$`)

	// slugRe matches the charset stringutil.GenerateSlug emits; anything else breaks /hc/ URLs.
	slugRe = regexp.MustCompile(fmt.Sprintf(`^[a-z0-9_-]{1,%d}$`, maxSlugLen))

	ilikeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

	youtubeEmbedRe = regexp.MustCompile(`^https://(www\.)?(youtube\.com|youtube-nocookie\.com)/embed/[\w-]+`)

	articleButtonClassRe = regexp.MustCompile(`^hc-button$`)

	textAlignRe = regexp.MustCompile(`^(left|center|right|justify)$`)

	blankLineRe = regexp.MustCompile(`\n{2,}`)

	// articleSanitizer strips unsafe HTML from article content since it renders raw on public pages.
	articleSanitizer = buildArticleSanitizer()

	// inlineTextSanitizer strips unsafe HTML from theme text fields that render raw on public pages.
	inlineTextSanitizer = buildInlineTextSanitizer()
)

// ArticleIndexer syncs article content into the AI embedding index.
type ArticleIndexer interface {
	ReindexHelpArticle(articleID int)
	RemoveHelpArticleEmbeddings(articleID int) error
}

// collectionGetter fetches a collection, optionally inside a transaction that locks the row.
type collectionGetter func(id int) (models.Collection, error)

type HelpCenterRequest struct {
	Name            string          `json:"name"`
	Slug            string          `json:"slug"`
	PageTitle       string          `json:"page_title"`
	MetaDescription string          `json:"meta_description"`
	CustomCSS       string          `json:"custom_css"`
	CustomJS        string          `json:"custom_js"`
	DefaultLocale   string          `json:"default_locale"`
	AllowedLocales  json.RawMessage `json:"allowed_locales"`
	Theme           json.RawMessage `json:"theme"`
	CustomDomain    string          `json:"custom_domain"`
	Template        string          `json:"template"`
}

type CollectionRequest struct {
	Slug        string `json:"slug"`
	ParentID    *int   `json:"parent_id"`
	Locale      string `json:"locale"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
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
	AuthorID        *int64 `json:"author_id"`
	CreatedBy       *int64 `json:"-"`
}

type Manager struct {
	db      *sqlx.DB
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
	GetAllHelpCenters    *sqlx.Stmt `query:"get-all-help-centers"`
	GetActiveHelpCenters *sqlx.Stmt `query:"get-active-help-centers"`
	GetHelpCenterByID    *sqlx.Stmt `query:"get-help-center-by-id"`
	GetHelpCenterBySlug  *sqlx.Stmt `query:"get-help-center-by-slug"`
	InsertHelpCenter     *sqlx.Stmt `query:"insert-help-center"`
	UpdateHelpCenter     *sqlx.Stmt `query:"update-help-center"`
	ToggleHelpCenter     *sqlx.Stmt `query:"toggle-help-center-active"`
	DeleteHelpCenter     *sqlx.Stmt `query:"delete-help-center"`
	GetLocalesInUse      *sqlx.Stmt `query:"get-help-center-locales-in-use"`

	GetCollectionsByHelpCenter *sqlx.Stmt `query:"get-collections-by-help-center"`
	GetCollectionByID          *sqlx.Stmt `query:"get-collection-by-id"`
	GetCollectionByIDForUpdate *sqlx.Stmt `query:"get-collection-by-id-for-update"`
	GetCollectionSubtreeDepth  *sqlx.Stmt `query:"get-collection-subtree-depth"`
	CollectionHasContent       *sqlx.Stmt `query:"collection-has-content"`
	LockHelpCenter             *sqlx.Stmt `query:"lock-help-center"`
	LockHelpCenterByCollection *sqlx.Stmt `query:"lock-help-center-by-collection"`
	GetSubtreeArticleIDs       *sqlx.Stmt `query:"get-article-ids-in-collection-subtree"`
	GetHelpCenterArticleIDs    *sqlx.Stmt `query:"get-article-ids-in-help-center"`
	UpdateCollectionSortOrder  *sqlx.Stmt `query:"update-collection-sort-order"`
	CollectionSlugExists       *sqlx.Stmt `query:"collection-slug-exists-in-help-center"`
	InsertCollection           *sqlx.Stmt `query:"insert-collection"`
	UpdateCollection           *sqlx.Stmt `query:"update-collection"`
	ToggleCollectionPublished  *sqlx.Stmt `query:"toggle-collection-published"`
	DeleteCollection           *sqlx.Stmt `query:"delete-collection"`

	GetArticleByID                *sqlx.Stmt `query:"get-article-by-id"`
	InsertArticle                 *sqlx.Stmt `query:"insert-article"`
	UpdateArticle                 *sqlx.Stmt `query:"update-article"`
	ArticleSlugExistsInHelpCenter *sqlx.Stmt `query:"article-slug-exists-in-help-center"`
	OtherArticleSlugExists        *sqlx.Stmt `query:"other-article-slug-exists-in-help-center"`
	MoveArticleToCollection       *sqlx.Stmt `query:"move-article-to-collection"`
	UpdateArticleSortOrder        *sqlx.Stmt `query:"update-article-sort-order"`
	UpdateArticleStatus           *sqlx.Stmt `query:"update-article-status"`
	DeleteArticle                 *sqlx.Stmt `query:"delete-article"`
	UserIsAuthorAssignable        *sqlx.Stmt `query:"user-is-author-assignable"`

	GetHelpCenterTreeData            *sqlx.Stmt `query:"get-help-center-tree-data"`
	GetPublicTreeData                *sqlx.Stmt `query:"get-public-tree-data"`
	GetPublishedArticleBySlug        *sqlx.Stmt `query:"get-published-article-by-slug"`
	GetPublishedArticleLocales       *sqlx.Stmt `query:"get-published-article-locales"`
	GetPublishedCollectionLocales    *sqlx.Stmt `query:"get-published-collection-locales"`
	GetPublishedArticles             *sqlx.Stmt `query:"get-published-articles"`
	GetPublishedArticlesByCollection *sqlx.Stmt `query:"get-published-articles-by-collection"`
	SearchPublishedArticles          *sqlx.Stmt `query:"search-published-articles"`
	IncrementArticleViewCount        *sqlx.Stmt `query:"increment-article-view-count"`
	IncrementPublishedArticleView    *sqlx.Stmt `query:"increment-published-article-view-count"`

	InsertArticleFeedback    *sqlx.Stmt `query:"insert-article-feedback"`
	InsertSearchQuery        *sqlx.Stmt `query:"insert-search-query"`
	GetTopSearchTerms        *sqlx.Stmt `query:"get-top-search-terms"`
	DeleteStaleSearchQueries *sqlx.Stmt `query:"delete-stale-search-queries"`
	GetNoResultSearchTerms   *sqlx.Stmt `query:"get-no-result-search-terms"`
}

// New creates and returns a new instance of the Manager.
func New(opts Opts) (*Manager, error) {
	var q queries
	if err := dbutil.ScanSQLFile("queries.sql", &q, opts.DB, efs); err != nil {
		return nil, err
	}
	return &Manager{
		db:      opts.DB,
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

// GetActiveHelpCenters retrieves the help centers whose public pages are live.
func (m *Manager) GetActiveHelpCenters() ([]models.HelpCenter, error) {
	var helpCenters = make([]models.HelpCenter, 0)
	if err := m.q.GetActiveHelpCenters.Select(&helpCenters); err != nil {
		m.lo.Error("error fetching active help centers", "error", err)
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
	req, err := m.normalizeHelpCenterRequest(req)
	if err != nil {
		return hc, err
	}
	if err := m.validateHelpCenterSlug(req.Slug); err != nil {
		return hc, err
	}
	if err := m.validateHelpCenterLengths(req); err != nil {
		return hc, err
	}
	if err := m.validateLocales(req.DefaultLocale, req.AllowedLocales); err != nil {
		return hc, err
	}
	if err := m.validateCustomDomain(req.CustomDomain, 0); err != nil {
		return hc, err
	}
	if err := m.q.InsertHelpCenter.Get(&hc, req.Name, req.Slug, req.PageTitle, req.MetaDescription, req.CustomCSS, req.CustomJS, req.DefaultLocale, req.AllowedLocales, req.Theme, req.CustomDomain, req.Template); err != nil {
		if dbutil.IsUniqueViolationError(err) {
			return hc, envelope.NewError(envelope.ConflictError, m.i18n.T("globals.messages.errorAlreadyExists"), nil)
		}
		m.lo.Error("error creating help center", "error", err)
		return hc, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return hc, nil
}

// DraftHelpCenter overlays an unsaved request onto the stored help center and returns it without writing.
func (m *Manager) DraftHelpCenter(id int, req HelpCenterRequest) (models.HelpCenter, error) {
	hc, err := m.GetHelpCenterByID(id)
	if err != nil {
		return hc, err
	}
	req, err = m.normalizeHelpCenterRequest(req)
	if err != nil {
		return hc, err
	}
	hc.Name = req.Name
	hc.PageTitle = req.PageTitle
	hc.MetaDescription = req.MetaDescription
	hc.CustomCSS = req.CustomCSS
	hc.CustomJS = req.CustomJS
	hc.DefaultLocale = req.DefaultLocale
	hc.AllowedLocales = req.AllowedLocales
	hc.Theme = req.Theme
	hc.CustomDomain = req.CustomDomain
	hc.Template = req.Template
	return hc, nil
}

// UpdateHelpCenter updates a help center.
func (m *Manager) UpdateHelpCenter(id int, req HelpCenterRequest) (models.HelpCenter, error) {
	var hc models.HelpCenter
	req, err := m.normalizeHelpCenterRequest(req)
	if err != nil {
		return hc, err
	}
	if err := m.validateHelpCenterSlug(req.Slug); err != nil {
		return hc, err
	}
	if err := m.validateHelpCenterLengths(req); err != nil {
		return hc, err
	}
	if err := m.validateLocales(req.DefaultLocale, req.AllowedLocales); err != nil {
		return hc, err
	}
	if err := m.validateLocalesRetained(id, req.AllowedLocales); err != nil {
		return hc, err
	}
	if err := m.validateCustomDomain(req.CustomDomain, id); err != nil {
		return hc, err
	}
	if err := m.q.UpdateHelpCenter.Get(&hc, id, req.Name, req.Slug, req.PageTitle, req.MetaDescription, req.CustomCSS, req.CustomJS, req.DefaultLocale, req.AllowedLocales, req.Theme, req.CustomDomain, req.Template); err != nil {
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
	// A paused help center 404s publicly, so its articles must leave the AI index too.
	m.reindexHelpCenterArticles(id)
	return hc, nil
}

func (m *Manager) reindexHelpCenterArticles(helpCenterID int) {
	if m.indexer == nil {
		return
	}
	var ids []int
	if err := m.q.GetHelpCenterArticleIDs.Select(&ids, helpCenterID); err != nil {
		m.lo.Error("error fetching help center articles", "error", err, "help_center_id", helpCenterID)
		return
	}
	for _, id := range ids {
		m.indexer.ReindexHelpArticle(id)
	}
}

// DeleteHelpCenter deletes a help center by ID.
func (m *Manager) DeleteHelpCenter(id int) error {
	// Read before the delete: the cascade takes the collections and articles with it.
	articleIDs := m.articleIDsFor(m.q.GetHelpCenterArticleIDs, id)
	if _, err := m.q.DeleteHelpCenter.Exec(id); err != nil {
		m.lo.Error("error deleting help center", "error", err, "id", id)
		return envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	m.removeArticleEmbeddings(articleIDs)
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

// CreateCollection creates a new collection; the help center row lock keeps a concurrent re-parent from pushing the new node past maxCollectionDepth.
func (m *Manager) CreateCollection(helpCenterID int, req CollectionRequest) (models.Collection, error) {
	var collection models.Collection
	if err := m.validateSlug(req.Slug); err != nil {
		return collection, err
	}
	if req.Locale == "" {
		req.Locale = defaultLocale
	}
	if err := m.validateHelpCenterServesLocale(helpCenterID, req.Locale); err != nil {
		return collection, err
	}
	req.Icon = sanitizeIconName(req.Icon)

	tx, err := m.db.Beginx()
	if err != nil {
		m.lo.Error("error starting transaction", "error", err)
		return collection, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	defer tx.Rollback()
	if err := m.lockHelpCenter(tx, helpCenterID); err != nil {
		return collection, err
	}
	if req.ParentID != nil {
		if err := m.validateCollectionParent(tx, *req.ParentID, 0, helpCenterID, req.Locale); err != nil {
			return collection, err
		}
	}
	slug, err := m.uniqueCollectionSlug(tx, helpCenterID, req.Slug, req.Locale)
	if err != nil {
		return collection, err
	}
	req.Slug = slug
	if err := tx.Stmtx(m.q.InsertCollection).Get(&collection, helpCenterID, req.Slug, req.ParentID, req.Locale, req.Name, req.Description, req.Icon, req.SortOrder, req.IsPublished); err != nil {
		if dbutil.IsUniqueViolationError(err) {
			return collection, envelope.NewError(envelope.ConflictError, m.i18n.T("globals.messages.errorAlreadyExists"), nil)
		}
		m.lo.Error("error creating collection", "error", err)
		return collection, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	if err := tx.Commit(); err != nil {
		m.lo.Error("error committing collection create", "error", err)
		return collection, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return collection, nil
}

// UpdateCollection updates a collection; the help center row lock keeps two simultaneous tree mutations from forming a cycle or overshooting maxCollectionDepth.
func (m *Manager) UpdateCollection(id int, req CollectionRequest) (models.Collection, error) {
	var collection models.Collection
	if err := m.validateSlug(req.Slug); err != nil {
		return collection, err
	}
	if req.Locale == "" {
		req.Locale = defaultLocale
	}
	req.Icon = sanitizeIconName(req.Icon)

	tx, err := m.db.Beginx()
	if err != nil {
		m.lo.Error("error starting transaction", "error", err)
		return collection, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	defer tx.Rollback()

	if err := m.lockHelpCenterByCollection(tx, id); err != nil {
		return collection, err
	}
	get := m.lockedCollectionGetter(tx)
	existing, err := get(id)
	if err != nil {
		return collection, err
	}
	if err := m.validateHelpCenterServesLocale(existing.HelpCenterID, req.Locale); err != nil {
		return collection, err
	}
	if req.Locale != existing.Locale {
		var hasContent bool
		if err := tx.Stmtx(m.q.CollectionHasContent).Get(&hasContent, id); err != nil {
			m.lo.Error("error checking collection content", "error", err, "id", id)
			return collection, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
		}
		if hasContent {
			return collection, envelope.NewError(envelope.InputError, m.i18n.T("helpCenter.localeChangeHasContent"), nil)
		}
	}
	if req.ParentID != nil {
		if err := m.validateCollectionParent(tx, *req.ParentID, id, existing.HelpCenterID, req.Locale); err != nil {
			return collection, err
		}
	}
	if err := tx.Stmtx(m.q.UpdateCollection).Get(&collection, id, req.Slug, req.ParentID, req.Locale, req.Name, req.Description, req.Icon, req.SortOrder, req.IsPublished); err != nil {
		if dbutil.IsUniqueViolationError(err) {
			return collection, envelope.NewError(envelope.ConflictError, m.i18n.T("globals.messages.errorAlreadyExists"), nil)
		}
		m.lo.Error("error updating collection", "error", err, "id", id)
		return collection, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	if err := tx.Commit(); err != nil {
		m.lo.Error("error committing collection update", "error", err, "id", id)
		return collection, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	// A hidden or re-parented collection can take its articles out of reach, and the AI only
	// cites reachable ones.
	m.reindexSubtreeArticles(id)
	return collection, nil
}

// UpdateCollectionSortOrders sets the sort order of the given collections in a help center.
func (m *Manager) UpdateCollectionSortOrders(helpCenterID int, orders map[int]int) error {
	tx, err := m.db.Beginx()
	if err != nil {
		m.lo.Error("error starting transaction", "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	defer tx.Rollback()
	for id, order := range orders {
		if _, err := tx.Stmtx(m.q.UpdateCollectionSortOrder).Exec(id, helpCenterID, order); err != nil {
			m.lo.Error("error updating collection sort order", "error", err, "id", id)
			return envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
		}
	}
	if err := tx.Commit(); err != nil {
		m.lo.Error("error committing collection sort orders", "error", err, "help_center_id", helpCenterID)
		return envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return nil
}

// ToggleCollectionPublished toggles the published status of a collection.
func (m *Manager) ToggleCollectionPublished(id int) (models.Collection, error) {
	var collection models.Collection
	if err := m.q.ToggleCollectionPublished.Get(&collection, id); err != nil {
		m.lo.Error("error toggling collection published status", "error", err, "id", id)
		return collection, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	// Publishing state decides whether the articles below are reachable, and only reachable
	// articles may be indexed for the AI agent.
	m.reindexSubtreeArticles(id)
	return collection, nil
}

func (m *Manager) reindexSubtreeArticles(collectionID int) {
	if m.indexer == nil {
		return
	}
	var ids []int
	if err := m.q.GetSubtreeArticleIDs.Select(&ids, collectionID); err != nil {
		m.lo.Error("error fetching collection subtree articles", "error", err, "collection_id", collectionID)
		return
	}
	for _, id := range ids {
		m.indexer.ReindexHelpArticle(id)
	}
}

// DeleteCollection deletes a collection by ID.
func (m *Manager) DeleteCollection(id int) error {
	// Read before the delete: the cascade takes the articles with it.
	articleIDs := m.articleIDsFor(m.q.GetSubtreeArticleIDs, id)
	if _, err := m.q.DeleteCollection.Exec(id); err != nil {
		m.lo.Error("error deleting collection", "error", err, "id", id)
		return envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	m.removeArticleEmbeddings(articleIDs)
	return nil
}

func (m *Manager) articleIDsFor(stmt *sqlx.Stmt, arg int) []int {
	if m.indexer == nil {
		return nil
	}
	var ids []int
	if err := stmt.Select(&ids, arg); err != nil {
		m.lo.Error("error fetching articles for embedding cleanup", "error", err, "arg", arg)
		return nil
	}
	return ids
}

func (m *Manager) removeArticleEmbeddings(ids []int) {
	if m.indexer == nil {
		return
	}
	for _, id := range ids {
		if err := m.indexer.RemoveHelpArticleEmbeddings(id); err != nil {
			m.lo.Error("error removing article embeddings", "error", err, "id", id)
		}
	}
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
	if err := m.validateArticleCollectionLocale(collectionID, req.Locale); err != nil {
		return article, err
	}
	if err := m.validateArticleAuthor(req.AuthorID); err != nil {
		return article, err
	}

	// Slug uniqueness is per help center but the DB index is per collection, so the
	// check and insert lock the help center row to serialize concurrent writers.
	tx, err := m.db.Beginx()
	if err != nil {
		m.lo.Error("error starting transaction", "error", err)
		return article, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	defer tx.Rollback()
	if err := m.lockHelpCenterByCollection(tx, collectionID); err != nil {
		return article, err
	}
	slug, err := m.uniqueArticleSlug(tx, collectionID, req.Slug, req.Locale)
	if err != nil {
		return article, err
	}
	req.Slug = slug
	req.Content = articleSanitizer.Sanitize(req.Content)
	req.Excerpt = strings.TrimSpace(req.Excerpt)
	if err := tx.Stmtx(m.q.InsertArticle).Get(&article, collectionID, req.AuthorID, req.CreatedBy, req.Slug, req.Locale, req.Title, req.Content, req.Excerpt, req.MetaTitle, req.MetaDescription, req.MetaImageURL, req.SortOrder, req.Status, req.AIEnabled); err != nil {
		if dbutil.IsUniqueViolationError(err) {
			return article, envelope.NewError(envelope.ConflictError, m.i18n.T("globals.messages.errorAlreadyExists"), nil)
		}
		m.lo.Error("error creating article", "error", err)
		return article, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	if err := tx.Commit(); err != nil {
		m.lo.Error("error committing article create", "error", err)
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
	existing, err := m.GetArticleByID(id)
	if err != nil {
		return article, err
	}
	collectionID := existing.CollectionID
	if req.CollectionID != nil {
		collectionID = *req.CollectionID
	}
	if err := m.validateArticleCollectionLocale(collectionID, req.Locale); err != nil {
		return article, err
	}
	// A soft-deleted author must not block saving an article that already has them.
	if req.AuthorID == nil || existing.AuthorID == nil || *existing.AuthorID != *req.AuthorID {
		if err := m.validateArticleAuthor(req.AuthorID); err != nil {
			return article, err
		}
	}
	req.Content = articleSanitizer.Sanitize(req.Content)
	req.Excerpt = strings.TrimSpace(req.Excerpt)

	// The slug check and write hold the same help center row lock as CreateArticle.
	tx, err := m.db.Beginx()
	if err != nil {
		m.lo.Error("error starting transaction", "error", err)
		return article, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	defer tx.Rollback()
	if err := m.lockHelpCenterByCollection(tx, collectionID); err != nil {
		return article, err
	}
	var slugTaken bool
	if err := tx.Stmtx(m.q.OtherArticleSlugExists).Get(&slugTaken, collectionID, req.Slug, req.Locale, id); err != nil {
		m.lo.Error("error checking article slug uniqueness", "error", err, "id", id, "slug", req.Slug)
		return article, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	if slugTaken {
		return article, envelope.NewError(envelope.ConflictError, m.i18n.T("globals.messages.errorAlreadyExists"), nil)
	}
	if err := tx.Stmtx(m.q.UpdateArticle).Get(&article, id, req.Slug, req.Locale, req.Title, req.Content, req.SortOrder, req.Status, req.AIEnabled, req.CollectionID, req.Excerpt, req.MetaTitle, req.MetaDescription, req.MetaImageURL, req.AuthorID); err != nil {
		if dbutil.IsUniqueViolationError(err) {
			return article, envelope.NewError(envelope.ConflictError, m.i18n.T("globals.messages.errorAlreadyExists"), nil)
		}
		m.lo.Error("error updating article", "error", err, "id", id)
		return article, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	if err := tx.Commit(); err != nil {
		m.lo.Error("error committing article update", "error", err, "id", id)
		return article, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	m.reindexArticle(article.ID)
	return article, nil
}

// MoveArticle moves an article to another collection, rejecting a slug that is already
// taken in the target collection's help center.
func (m *Manager) MoveArticle(id, collectionID int) (models.Article, error) {
	var article models.Article
	// The checks, slug check and write hold the same help center row lock as CreateArticle.
	tx, err := m.db.Beginx()
	if err != nil {
		m.lo.Error("error starting transaction", "error", err)
		return article, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	defer tx.Rollback()
	if err := m.lockHelpCenterByCollection(tx, collectionID); err != nil {
		return article, err
	}
	var existing models.Article
	if err := tx.Stmtx(m.q.GetArticleByID).Get(&existing, id); err != nil {
		if err == sql.ErrNoRows {
			return article, envelope.NewError(envelope.NotFoundError, m.i18n.T("globals.messages.notFound"), nil)
		}
		m.lo.Error("error fetching article", "error", err, "id", id)
		return article, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	get := m.lockedCollectionGetter(tx)
	source, err := get(existing.CollectionID)
	if err != nil {
		return article, err
	}
	target, err := get(collectionID)
	if err != nil {
		return article, err
	}
	if target.HelpCenterID != source.HelpCenterID {
		return article, envelope.NewError(envelope.InputError, m.i18n.T("helpCenter.invalidCollection"), nil)
	}
	if target.Locale != existing.Locale {
		return article, envelope.NewError(envelope.InputError, m.i18n.T("helpCenter.collectionLocaleMismatch"), nil)
	}
	var slugTaken bool
	if err := tx.Stmtx(m.q.OtherArticleSlugExists).Get(&slugTaken, collectionID, existing.Slug, existing.Locale, id); err != nil {
		m.lo.Error("error checking article slug uniqueness", "error", err, "id", id)
		return article, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	if slugTaken {
		return article, envelope.NewError(envelope.ConflictError, m.i18n.T("globals.messages.errorAlreadyExists"), nil)
	}
	if err := tx.Stmtx(m.q.MoveArticleToCollection).Get(&article, id, collectionID); err != nil {
		m.lo.Error("error moving article", "error", err, "id", id, "collection_id", collectionID)
		return article, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	if err := tx.Commit(); err != nil {
		m.lo.Error("error committing article move", "error", err, "id", id)
		return article, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	m.reindexArticle(article.ID)
	return article, nil
}

// UpdateArticleSortOrders sets the sort order of the given articles in a collection.
func (m *Manager) UpdateArticleSortOrders(collectionID int, orders map[int]int) error {
	tx, err := m.db.Beginx()
	if err != nil {
		m.lo.Error("error starting transaction", "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	defer tx.Rollback()
	for id, order := range orders {
		if _, err := tx.Stmtx(m.q.UpdateArticleSortOrder).Exec(id, collectionID, order); err != nil {
			m.lo.Error("error updating article sort order", "error", err, "id", id)
			return envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
		}
	}
	if err := tx.Commit(); err != nil {
		m.lo.Error("error committing article sort orders", "error", err, "collection_id", collectionID)
		return envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return nil
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

// GetPublicTree returns the published-only tree for a help center, filtered to locale (empty = all).
func (m *Manager) GetPublicTree(helpCenter models.HelpCenter, locale string) (models.TreeResponse, error) {
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

// GetPublishedArticle retrieves a published article by help center slug and article slug, restricted to locale (empty = any).
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

// GetPublishedArticlesByCollection returns published articles in a collection, excluding one article, filtered to locale (empty = all).
func (m *Manager) GetPublishedArticlesByCollection(collectionID, excludeArticleID int, locale string, limit int) ([]models.Article, error) {
	var articles = make([]models.Article, 0)
	if err := m.q.GetPublishedArticlesByCollection.Select(&articles, collectionID, excludeArticleID, locale, limit); err != nil {
		m.lo.Error("error fetching collection articles", "error", err, "collection_id", collectionID)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return articles, nil
}

// GetPublishedArticleLocales returns the locales a published article with the given slug exists in.
func (m *Manager) GetPublishedArticleLocales(helpCenterSlug, articleSlug string) ([]string, error) {
	var locales = make([]string, 0)
	if err := m.q.GetPublishedArticleLocales.Select(&locales, helpCenterSlug, articleSlug); err != nil {
		m.lo.Error("error fetching article locales", "error", err, "help_center_slug", helpCenterSlug, "article_slug", articleSlug)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return locales, nil
}

// GetPublishedCollectionLocales returns the locales a published collection with the given slug exists in.
func (m *Manager) GetPublishedCollectionLocales(helpCenterSlug, collectionSlug string) ([]string, error) {
	var locales = make([]string, 0)
	if err := m.q.GetPublishedCollectionLocales.Select(&locales, helpCenterSlug, collectionSlug); err != nil {
		m.lo.Error("error fetching collection locales", "error", err, "help_center_slug", helpCenterSlug, "collection_slug", collectionSlug)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return locales, nil
}

// SearchPublishedArticles searches published articles in a help center, content trimmed to an excerpt, filtered to locale.
func (m *Manager) SearchPublishedArticles(helpCenterSlug, query, locale string, limit int) ([]models.Article, error) {
	var articles = make([]models.Article, 0)
	query = strings.TrimSpace(query)
	if utf8.RuneCountInString(query) < minSearchQueryLen {
		return articles, nil
	}
	query = truncateRunes(query, maxSearchQueryLen)
	tsQuery := prefixTSQuery(query)
	query = ilikeEscaper.Replace(query)
	if err := m.q.SearchPublishedArticles.Select(&articles, helpCenterSlug, query, limit, locale, tsQuery); err != nil {
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

func (m *Manager) IncrementPublishedArticleView(helpCenterSlug, articleSlug, locale string) {
	if _, err := m.q.IncrementPublishedArticleView.Exec(helpCenterSlug, articleSlug, locale); err != nil {
		m.lo.Error("error incrementing article view count", "error", err, "slug", articleSlug)
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

// LogSearch records a public search term and how many results it returned. Terms under
// the search floor never ran, so logging them would pollute the zero-result insights.
func (m *Manager) LogSearch(helpCenterID int, query string, resultsCount int) {
	query = truncateRunes(strings.TrimSpace(query), maxSearchQueryLen)
	if utf8.RuneCountInString(query) < minSearchQueryLen {
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
	if err := m.q.GetTopSearchTerms.Select(&insights.TopSearches, helpCenterID, limit, searchLogRetentionDays); err != nil {
		m.lo.Error("error fetching top search terms", "error", err, "help_center_id", helpCenterID)
		return insights, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	if err := m.q.GetNoResultSearchTerms.Select(&insights.NoResultSearch, helpCenterID, limit, searchLogRetentionDays); err != nil {
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
			icon         *string
			sortOrder    int
			isPublished  *bool
			collectionID *int
			title        *string
			content      *string
			status       *string
			viewCount    *int
			aiEnabled    *bool
			authorName   *string
			authorAvatar *string
		)
		if err := rows.Scan(&itemType, &id, &createdAt, &updatedAt, &helpCenterID, &slug, &parentID, &locale, &name, &description, &icon, &sortOrder, &isPublished, &collectionID, &title, &content, &status, &viewCount, &aiEnabled, &authorName, &authorAvatar); err != nil {
			m.lo.Error("error scanning tree data", "error", err)
			return nil, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
		}

		switch itemType {
		case "collection":
			desc := ""
			if description != nil {
				desc = *description
			}
			ic := ""
			if icon != nil {
				ic = *icon
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
					Icon:         ic,
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
			article.AuthorName = authorName
			article.AuthorAvatar = authorAvatar
			collection.Articles = append(collection.Articles, article)
		}
	}
	if err := rows.Err(); err != nil {
		m.lo.Error("error iterating tree data", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}

	var buildTree func(parentID *int, depth int) []models.TreeCollection
	buildTree = func(parentID *int, depth int) []models.TreeCollection {
		children := make([]models.TreeCollection, 0)
		if depth > maxCollectionDepth {
			return children
		}
		for _, id := range rootOrder {
			col := collections[id]
			matches := (col.ParentID == nil && parentID == nil) ||
				(col.ParentID != nil && parentID != nil && *col.ParentID == *parentID)
			if matches {
				col.Children = buildTree(&col.ID, depth+1)
				children = append(children, *col)
			}
		}
		return children
	}
	tree := buildTree(nil, 1)

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
	fillAuthors(tree)

	return tree, nil
}

// fillAuthors aggregates the distinct article authors of each collection and its
// descendants, keeping at most maxCardAuthors for the avatar stack.
func fillAuthors(cols []models.TreeCollection) {
	var collect func(col *models.TreeCollection) map[string]models.ArticleAuthor
	collect = func(col *models.TreeCollection) map[string]models.ArticleAuthor {
		authors := map[string]models.ArticleAuthor{}
		for _, a := range col.Articles {
			if a.AuthorName == nil || strings.TrimSpace(*a.AuthorName) == "" {
				continue
			}
			author := models.ArticleAuthor{Name: strings.TrimSpace(*a.AuthorName)}
			if a.AuthorAvatar != nil {
				author.Avatar = *a.AuthorAvatar
			}
			authors[author.Name] = author
		}
		for i := range col.Children {
			for name, author := range collect(&col.Children[i]) {
				authors[name] = author
			}
		}
		col.AuthorCount = len(authors)
		col.Authors = col.Authors[:0]
		for _, name := range slices.Sorted(maps.Keys(authors)) {
			if len(col.Authors) == maxCardAuthors {
				break
			}
			col.Authors = append(col.Authors, authors[name])
		}
		return authors
	}
	for i := range cols {
		collect(&cols[i])
	}
}

// reindexArticle asks the AI indexer to re-sync an article's embeddings.
func (m *Manager) reindexArticle(id int) {
	if m.indexer != nil {
		m.indexer.ReindexHelpArticle(id)
	}
}

// lockedCollectionGetter returns a getter that reads collections through tx with FOR UPDATE.
func (m *Manager) lockedCollectionGetter(tx *sqlx.Tx) collectionGetter {
	return func(id int) (models.Collection, error) {
		var collection models.Collection
		if err := tx.Stmtx(m.q.GetCollectionByIDForUpdate).Get(&collection, id); err != nil {
			if err == sql.ErrNoRows {
				return collection, envelope.NewError(envelope.NotFoundError, m.i18n.T("globals.messages.notFound"), nil)
			}
			m.lo.Error("error locking collection", "error", err, "id", id)
			return collection, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
		}
		return collection, nil
	}
}

// validateCollectionParent rejects parents that belong to another help center, use another
// language, would make the collection its own ancestor, or would push the collection's own
// descendants past maxCollectionDepth. Runs inside tx with the help center row locked.
func (m *Manager) validateCollectionParent(tx *sqlx.Tx, parentID, selfID, helpCenterID int, locale string) error {
	get := m.lockedCollectionGetter(tx)
	depth := 2
	currentID := parentID
	for {
		if currentID == selfID {
			return envelope.NewError(envelope.InputError, m.i18n.T("helpCenter.invalidParent"), nil)
		}
		parent, err := get(currentID)
		if err != nil {
			return err
		}
		if parent.HelpCenterID != helpCenterID {
			return envelope.NewError(envelope.InputError, m.i18n.T("helpCenter.invalidParent"), nil)
		}
		if parent.Locale != locale {
			return envelope.NewError(envelope.InputError, m.i18n.T("helpCenter.parentLocaleMismatch"), nil)
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
	if selfID == 0 {
		return nil
	}
	subtreeDepth, err := m.collectionSubtreeDepth(tx, selfID)
	if err != nil {
		return err
	}
	if depth+subtreeDepth-1 > maxCollectionDepth {
		return envelope.NewError(envelope.InputError, m.i18n.T("helpCenter.maxDepthReached"), nil)
	}
	return nil
}

// collectionSubtreeDepth returns how many levels the collection spans, itself counting as one.
func (m *Manager) collectionSubtreeDepth(tx *sqlx.Tx, id int) (int, error) {
	var depth int
	if err := tx.Stmtx(m.q.GetCollectionSubtreeDepth).Get(&depth, id); err != nil {
		m.lo.Error("error fetching collection subtree depth", "error", err, "id", id)
		return 0, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return depth, nil
}

// lockHelpCenter locks the help center row that serializes tree and article slug mutations.
func (m *Manager) lockHelpCenter(tx *sqlx.Tx, id int) error {
	var hcID int
	if err := tx.Stmtx(m.q.LockHelpCenter).Get(&hcID, id); err != nil {
		if err == sql.ErrNoRows {
			return envelope.NewError(envelope.NotFoundError, m.i18n.T("globals.messages.notFound"), nil)
		}
		m.lo.Error("error locking help center", "error", err, "id", id)
		return envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return nil
}

// lockHelpCenterByCollection locks the collection's help center row, see lockHelpCenter.
func (m *Manager) lockHelpCenterByCollection(tx *sqlx.Tx, collectionID int) error {
	var hcID int
	if err := tx.Stmtx(m.q.LockHelpCenterByCollection).Get(&hcID, collectionID); err != nil {
		if err == sql.ErrNoRows {
			return envelope.NewError(envelope.NotFoundError, m.i18n.T("globals.messages.notFound"), nil)
		}
		m.lo.Error("error locking help center", "error", err, "collection_id", collectionID)
		return envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return nil
}

// validateArticleCollectionLocale rejects a collection in a different language than the article.
func (m *Manager) validateArticleCollectionLocale(collectionID int, locale string) error {
	collection, err := m.GetCollectionByID(collectionID)
	if err != nil {
		return err
	}
	if collection.Locale != locale {
		return envelope.NewError(envelope.InputError, m.i18n.T("helpCenter.collectionLocaleMismatch"), nil)
	}
	return nil
}

// validateArticleAuthor rejects an author that isn't an agent or AI assistant.
func (m *Manager) validateArticleAuthor(authorID *int64) error {
	if authorID == nil {
		return nil
	}
	var ok bool
	if err := m.q.UserIsAuthorAssignable.Get(&ok, *authorID); err != nil {
		m.lo.Error("error checking article author", "error", err, "author_id", *authorID)
		return envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	if !ok {
		return envelope.NewError(envelope.InputError, m.i18n.T("helpCenter.invalidAuthor"), nil)
	}
	return nil
}

// uniqueCollectionSlug appends a numeric suffix until the slug is unique within the help center.
func (m *Manager) uniqueCollectionSlug(tx *sqlx.Tx, helpCenterID int, slug, locale string) (string, error) {
	candidate := slug
	for i := 2; ; i++ {
		var exists bool
		if err := tx.Stmtx(m.q.CollectionSlugExists).Get(&exists, helpCenterID, candidate, locale); err != nil {
			m.lo.Error("error checking collection slug uniqueness", "error", err, "slug", candidate)
			return "", envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
		}
		if !exists {
			return candidate, nil
		}
		candidate = withSuffix(slug, fmt.Sprintf("-%d", i))
	}
}

// uniqueArticleSlug appends a numeric suffix until the slug is unique within the collection's help center.
func (m *Manager) uniqueArticleSlug(tx *sqlx.Tx, collectionID int, slug, locale string) (string, error) {
	candidate := slug
	for i := 2; ; i++ {
		var exists bool
		if err := tx.Stmtx(m.q.ArticleSlugExistsInHelpCenter).Get(&exists, collectionID, candidate, locale); err != nil {
			m.lo.Error("error checking article slug uniqueness", "error", err, "slug", candidate)
			return "", envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
		}
		if !exists {
			return candidate, nil
		}
		candidate = withSuffix(slug, fmt.Sprintf("-%d", i))
	}
}

// validateSlug rejects slugs whose charset would break /hc/ URLs.
func (m *Manager) validateSlug(slug string) error {
	if !slugRe.MatchString(slug) {
		return envelope.NewError(envelope.InputError, m.i18n.T("helpCenter.invalidSlug"), nil)
	}
	return nil
}

// validateHelpCenterSlug rejects malformed slugs and slugs that collide with public help center routes.
func (m *Manager) validateHelpCenterSlug(slug string) error {
	if !slugRe.MatchString(slug) || slices.Contains(reservedSlugs, slug) {
		return envelope.NewError(envelope.InputError, m.i18n.T("helpCenter.invalidSlug"), nil)
	}
	return nil
}

func (m *Manager) validateHelpCenterLengths(req HelpCenterRequest) error {
	for _, f := range []struct {
		name  string
		value string
		max   int
	}{
		{"name", req.Name, maxNameLen},
		{"page_title", req.PageTitle, maxPageTitleLen},
		{"meta_description", req.MetaDescription, maxMetaDescLen},
	} {
		if utf8.RuneCountInString(f.value) > f.max {
			return envelope.NewError(envelope.InputError, m.i18n.Ts("globals.messages.fieldTooLong", "field", f.name, "max", strconv.Itoa(f.max)), nil)
		}
	}
	return nil
}

// validateLocales rejects language codes outside the supported set.
func (m *Manager) validateLocales(defaultLocale string, allowed json.RawMessage) error {
	for _, l := range append(parseLocales(allowed), defaultLocale) {
		if _, ok := supportedLocales[strings.TrimSpace(l)]; !ok {
			return envelope.NewError(envelope.InputError, m.i18n.T("helpCenter.invalidLocale"), nil)
		}
	}
	return nil
}

// Only currently-allowed locales are checked, so a pre-existing orphan locale can still be saved.
func (m *Manager) validateLocalesRetained(helpCenterID int, allowed json.RawMessage) error {
	hc, err := m.GetHelpCenterByID(helpCenterID)
	if err != nil {
		return err
	}
	kept := parseLocales(allowed)
	var dropped []string
	for _, l := range parseLocales(hc.AllowedLocales) {
		if !slices.Contains(kept, l) {
			dropped = append(dropped, l)
		}
	}
	if len(dropped) == 0 {
		return nil
	}
	var inUse []string
	if err := m.q.GetLocalesInUse.Select(&inUse, helpCenterID); err != nil {
		m.lo.Error("error fetching help center locales in use", "error", err, "help_center_id", helpCenterID)
		return envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	for _, l := range dropped {
		if slices.Contains(inUse, l) {
			return envelope.NewError(envelope.InputError, m.i18n.Ts("helpCenter.localeInUse", "locale", l), nil)
		}
	}
	return nil
}

// validateCustomDomain rejects malformed domains and hostnames already claimed by another help center.
func (m *Manager) validateCustomDomain(domain string, excludeID int) error {
	if domain == "" {
		return nil
	}
	u, err := url.Parse(domain)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" ||
		u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" {
		return envelope.NewError(envelope.InputError, m.i18n.T("helpCenter.invalidCustomDomain"), nil)
	}
	helpCenters, err := m.GetAllHelpCenters()
	if err != nil {
		return err
	}
	for _, hc := range helpCenters {
		if hc.ID == excludeID || hc.CustomDomain == "" {
			continue
		}
		other, err := url.Parse(hc.CustomDomain)
		if err != nil {
			continue
		}
		if strings.EqualFold(other.Hostname(), u.Hostname()) {
			return envelope.NewError(envelope.ConflictError, m.i18n.T("helpCenter.customDomainInUse"), nil)
		}
	}
	return nil
}

// validateHelpCenterServesLocale rejects a locale the help center doesn't list.
func (m *Manager) validateHelpCenterServesLocale(helpCenterID int, locale string) error {
	hc, err := m.GetHelpCenterByID(helpCenterID)
	if err != nil {
		return err
	}
	if !slices.Contains(parseLocales(hc.AllowedLocales), locale) {
		return envelope.NewError(envelope.InputError, m.i18n.T("helpCenter.localeNotSupported"), nil)
	}
	return nil
}

// RenderInlineMarkdown renders theme text fields as inline HTML, stripping anything unsafe.
func RenderInlineMarkdown(s string) string {
	if strings.TrimSpace(s) == "" {
		return ""
	}
	html := stringutil.Markdown2HTML(blankLineRe.ReplaceAllString(s, "\n"))
	return strings.TrimSpace(inlineTextSanitizer.Sanitize(html))
}

// GenerateSlug derives a slug from a title, bounded to what validateSlug accepts.
func GenerateSlug(title string) string {
	return withSuffix(stringutil.GenerateSlug(title), "")
}

// withSuffix truncates a slug so the suffix still fits inside maxSlugLen.
func withSuffix(slug, suffix string) string {
	return strings.Trim(truncateRunes(slug, maxSlugLen-len(suffix)), "-") + suffix
}

func (m *Manager) normalizeHelpCenterRequest(req HelpCenterRequest) (HelpCenterRequest, error) {
	if req.DefaultLocale == "" {
		req.DefaultLocale = defaultLocale
	}
	if !slices.Contains(helpCenterTemplates, req.Template) {
		req.Template = models.TemplateClassic
	}
	req.CustomDomain = strings.TrimRight(strings.TrimSpace(req.CustomDomain), "/")

	locales := normalizeLocales(parseLocales(req.AllowedLocales), req.DefaultLocale)
	if b, err := json.Marshal(locales); err == nil {
		req.AllowedLocales = b
	}
	theme, err := normalizeTheme(req.Theme)
	if err != nil {
		m.lo.Error("error normalizing help center theme", "error", err)
		return req, envelope.NewError(envelope.InputError, m.i18n.T("helpCenter.invalidTheme"), nil)
	}
	req.Theme = theme
	return req, nil
}

// normalizeTheme drops theme values that aren't safe to inject into CSS, and rejects a theme it can't read.
func normalizeTheme(raw json.RawMessage) (json.RawMessage, error) {
	if len(raw) == 0 {
		return json.RawMessage("{}"), nil
	}
	t := models.DefaultTheme()
	if err := json.Unmarshal(raw, &t); err != nil {
		return nil, err
	}
	t.Color = sanitizeHexColor(t.Color)
	if t.Color == "" {
		t.Color = defaultAccentColor
	}
	t.LogoURL = sanitizeAssetURL(t.LogoURL)
	t.NavLinks = sanitizeNavLinks(t.NavLinks)
	t.Header.Heading = strings.TrimSpace(t.Header.Heading)
	t.Header.BackgroundColor = sanitizeHexColor(t.Header.BackgroundColor)
	t.Header.GradientFrom = sanitizeHexColor(t.Header.GradientFrom)
	t.Header.GradientTo = sanitizeHexColor(t.Header.GradientTo)
	t.Header.BackgroundImage = sanitizeAssetURL(t.Header.BackgroundImage)
	t.Header.TextColor = sanitizeHexColor(t.Header.TextColor)
	t.Footer.BackgroundColor = sanitizeHexColor(t.Footer.BackgroundColor)
	t.Footer.TextColor = sanitizeHexColor(t.Footer.TextColor)
	t.Favicon = sanitizeAssetURL(t.Favicon)
	t.FooterLinks = sanitizeNavLinks(t.FooterLinks)
	t.SocialLinks = sanitizeSocialLinks(t.SocialLinks)
	t.Tagline = strings.TrimSpace(t.Tagline)
	t.Footer.Tagline = strings.TrimSpace(t.Footer.Tagline)
	t.Announcement.Text = strings.TrimSpace(t.Announcement.Text)
	t.Announcement.LinkLabel = strings.TrimSpace(t.Announcement.LinkLabel)
	t.Announcement.LinkURL = sanitizeAssetURL(t.Announcement.LinkURL)
	if t.Announcement.Text == "" {
		t.Announcement = models.AnnouncementTheme{}
	}
	if !slices.Contains(headerBackgroundTypes, t.Header.BackgroundType) {
		t.Header.BackgroundType = ""
	}
	if t.Layout.Collections != "list" {
		t.Layout.Collections = ""
	}
	if t.Layout.Columns != 2 && t.Layout.Columns != 3 {
		t.Layout.Columns = 0
	}
	if !slices.Contains(cardIconPositions, t.Cards.IconPosition) {
		t.Cards.IconPosition = cardIconPositions[0]
	}
	b, err := json.Marshal(t)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func sanitizeHexColor(c string) string {
	if !hexColorRe.MatchString(c) {
		return ""
	}
	return c
}

// A leading "//" is protocol-relative and points off-site while reading as root-relative.
func sanitizeAssetURL(u string) string {
	u = strings.TrimSpace(u)
	if !assetURLRe.MatchString(u) || strings.HasPrefix(u, "//") {
		return ""
	}
	return u
}

// sanitizeNavLinks drops link URLs that aren't absolute or root-relative; a bare host resolves inside the help center.
func sanitizeNavLinks(links []models.NavLink) []models.NavLink {
	out := make([]models.NavLink, 0, len(links))
	for _, l := range links {
		l.URL = sanitizeAssetURL(l.URL)
		l.Label = strings.TrimSpace(l.Label)
		if l.URL == "" || l.Label == "" {
			continue
		}
		out = append(out, l)
	}
	return out
}

func sanitizeSocialLinks(links []models.SocialLink) []models.SocialLink {
	out := make([]models.SocialLink, 0, len(links))
	for _, l := range links {
		l.URL = sanitizeAssetURL(l.URL)
		if l.URL == "" {
			continue
		}
		if !slices.Contains(socialPlatforms, l.Platform) {
			l.Platform = socialPlatforms[0]
		}
		out = append(out, l)
	}
	return out
}

func sanitizeIconName(n string) string {
	n = strings.TrimSpace(n)
	if !iconNameRe.MatchString(n) {
		return ""
	}
	return n
}

func parseLocales(raw json.RawMessage) []string {
	locales := []string{}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &locales)
	}
	return locales
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

func isValidArticleStatus(status string) bool {
	return status == models.ArticleStatusDraft || status == models.ArticleStatusPublished
}

func truncateRunes(s string, limit int) string {
	runes := []rune(s)
	if len(runes) <= limit {
		return s
	}
	return string(runes[:limit])
}

// buildArticleSanitizer returns the HTML sanitization policy for article content.
func buildArticleSanitizer() *bluemonday.Policy {
	p := bluemonday.UGCPolicy()
	// Links off the help center open in a new tab; bluemonday adds rel="noopener" with it.
	p.AddTargetBlankToFullyQualifiedLinks(true)
	p.RequireNoFollowOnFullyQualifiedLinks(true)
	p.AllowAttrs("class").OnElements("img", "pre", "code", "div", "span", "p")
	p.AllowAttrs("class").Matching(articleButtonClassRe).OnElements("a")
	// Collapsible sections render as native <details>/<summary>.
	p.AllowElements("details", "summary")
	p.AllowAttrs("class").OnElements("details", "summary")
	p.AllowStyles("text-align").Matching(textAlignRe).OnElements("p", "h1", "h2", "h3", "h4", "div")
	p.AllowAttrs("width", "height").OnElements("img")
	p.AllowStyles("width", "height", "max-width").OnElements("img")
	p.AllowStyles("border", "width", "margin", "table-layout", "border-collapse", "border-radius",
		"box-sizing", "min-width", "padding", "vertical-align", "background-color", "color",
		"font-weight", "text-align").OnElements("table", "td", "th")
	// YouTube embeds as rendered by the tiptap Youtube extension.
	p.AllowAttrs("data-youtube-video").OnElements("div")
	p.AllowAttrs("src").Matching(youtubeEmbedRe).OnElements("iframe")
	p.AllowAttrs("width", "height", "allowfullscreen", "frameborder", "allow", "referrerpolicy", "start", "title").OnElements("iframe")
	p.AllowStyles("text-align").Matching(textAlignRe).OnElements("iframe")
	// UGCPolicy binds alt to a charset that drops the whole attribute on a ? or :. A later
	// unrestricted policy wins, since any matching policy passes.
	p.AllowAttrs("alt").OnElements("img")
	return p
}

// prefixTSQuery builds an AND-ed prefix tsquery from the reader's search terms.
func prefixTSQuery(query string) string {
	var terms []string
	for _, word := range strings.FieldsFunc(query, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	}) {
		terms = append(terms, word+":*")
	}
	return strings.Join(terms, " & ")
}

// buildInlineTextSanitizer returns the HTML sanitization policy for short theme text fields.
func buildInlineTextSanitizer() *bluemonday.Policy {
	p := bluemonday.NewPolicy()
	p.AllowStandardURLs()
	p.AllowAttrs("href").OnElements("a")
	p.AddTargetBlankToFullyQualifiedLinks(true)
	p.RequireNoFollowOnFullyQualifiedLinks(true)
	p.AllowElements("b", "strong", "i", "em", "u", "s", "del", "ins", "mark", "small", "sub", "sup", "br", "span", "code")
	return p
}
