// Package helpcenter handles the management of help centers, collections, and articles.
package helpcenter

import (
	"embed"
	"encoding/json"
	"fmt"
	"time"

	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/helpcenter/models"
	"github.com/abhinavxd/libredesk/internal/stringutil"
	"github.com/jmoiron/sqlx"
	"github.com/knadh/go-i18n"
	"github.com/pgvector/pgvector-go"
	"github.com/zerodha/logf"
)

var (
	//go:embed queries.sql
	efs embed.FS
)

// Request structs for help center operations
type HelpCenterCreateRequest struct {
	Name          string `json:"name"`
	Slug          string `json:"slug"`
	PageTitle     string `json:"page_title"`
	DefaultLocale string `json:"default_locale"`
}

type HelpCenterUpdateRequest struct {
	Name          string `json:"name"`
	Slug          string `json:"slug"`
	PageTitle     string `json:"page_title"`
	DefaultLocale string `json:"default_locale"`
}

type CollectionCreateRequest struct {
	Slug        string  `json:"slug"`
	ParentID    *int    `json:"parent_id"`
	Locale      string  `json:"locale"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
	SortOrder   int     `json:"sort_order"`
	IsPublished bool    `json:"is_published"`
}

type CollectionUpdateRequest struct {
	Slug        string  `json:"slug"`
	ParentID    *int    `json:"parent_id"`
	Locale      string  `json:"locale"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
	SortOrder   int     `json:"sort_order"`
	IsPublished bool    `json:"is_published"`
}

type ArticleCreateRequest struct {
	Slug      string `json:"slug"`
	Locale    string `json:"locale"`
	Title     string `json:"title"`
	Content   string `json:"content"`
	SortOrder int    `json:"sort_order"`
	Status    string `json:"status"`
	AIEnabled bool   `json:"ai_enabled"`
}

type ArticleUpdateRequest struct {
	Slug         string `json:"slug"`
	Locale       string `json:"locale"`
	Title        string `json:"title"`
	Content      string `json:"content"`
	SortOrder    int    `json:"sort_order"`
	Status       string `json:"status"`
	AIEnabled    bool   `json:"ai_enabled"`
	CollectionID *int   `json:"collection_id,omitempty"`
}

type UpdateStatusRequest struct {
	Status string `json:"status"`
}

type AIStore interface {
	GetEmbeddings(text string) ([]float32, error)
	GetChunkConfig() stringutil.ChunkConfig
}

type Manager struct {
	q    queries
	db   *sqlx.DB
	lo   *logf.Logger
	i18n *i18n.I18n
	ai   AIStore
}

// Opts contains options for initializing the Manager.
type Opts struct {
	DB   *sqlx.DB
	Lo   *logf.Logger
	I18n *i18n.I18n
}

// queries contains prepared SQL queries.
type queries struct {
	// Help Centers
	GetAllHelpCenters   *sqlx.Stmt `query:"get-all-help-centers"`
	GetHelpCenterByID   *sqlx.Stmt `query:"get-help-center-by-id"`
	GetHelpCenterBySlug *sqlx.Stmt `query:"get-help-center-by-slug"`
	InsertHelpCenter    *sqlx.Stmt `query:"insert-help-center"`
	UpdateHelpCenter    *sqlx.Stmt `query:"update-help-center"`
	DeleteHelpCenter    *sqlx.Stmt `query:"delete-help-center"`

	// Collections
	GetCollectionsByHelpCenter          *sqlx.Stmt `query:"get-collections-by-help-center"`
	GetCollectionsByHelpCenterAndLocale *sqlx.Stmt `query:"get-collections-by-help-center-and-locale"`
	GetCollectionByID                   *sqlx.Stmt `query:"get-collection-by-id"`
	InsertCollection                    *sqlx.Stmt `query:"insert-collection"`
	UpdateCollection                    *sqlx.Stmt `query:"update-collection"`
	ToggleCollectionPublished           *sqlx.Stmt `query:"toggle-collection-published"`
	DeleteCollection                    *sqlx.Stmt `query:"delete-collection"`

	// Articles
	GetArticlesByCollection          *sqlx.Stmt `query:"get-articles-by-collection"`
	GetArticlesByCollectionAndLocale *sqlx.Stmt `query:"get-articles-by-collection-and-locale"`
	GetArticleByID                   *sqlx.Stmt `query:"get-article-by-id"`
	InsertArticle                    *sqlx.Stmt `query:"insert-article"`
	UpdateArticle                    *sqlx.Stmt `query:"update-article"`
	MoveArticle                      *sqlx.Stmt `query:"move-article"`
	UpdateArticleStatus              *sqlx.Stmt `query:"update-article-status"`
	DeleteArticle                    *sqlx.Stmt `query:"delete-article"`
	SearchArticlesByVector           *sqlx.Stmt `query:"search-articles-by-vector"`
	UpdateArticleEmbedding           *sqlx.Stmt `query:"update-article-embedding"`
	SearchKnowledgeBase              *sqlx.Stmt `query:"search-knowledge-base"`
	DeleteEmbeddingsBySource         *sqlx.Stmt `query:"delete-embeddings-by-source"`
	InsertEmbedding                  *sqlx.Stmt `query:"insert-embedding"`
	GetHelpCenterTreeData            *sqlx.Stmt `query:"get-help-center-tree-data"`
}

// New creates and returns a new instance of the Manager.
func New(opts Opts) (*Manager, error) {
	var q queries

	if err := dbutil.ScanSQLFile("queries.sql", &q, opts.DB, efs); err != nil {
		return nil, err
	}

	return &Manager{
		q:    q,
		db:   opts.DB,
		lo:   opts.Lo,
		i18n: opts.I18n,
	}, nil
}

func (m *Manager) SetAIStore(ai AIStore) {
	m.ai = ai
}

// Help Centers

// GetAllHelpCenters retrieves all help centers.
func (m *Manager) GetAllHelpCenters() ([]models.HelpCenter, error) {
	var helpCenters = make([]models.HelpCenter, 0)
	if err := m.q.GetAllHelpCenters.Select(&helpCenters); err != nil {
		m.lo.Error("error fetching help centers", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorFetching", "name", "help centers"), nil)
	}
	return helpCenters, nil
}

// GetHelpCenterByID retrieves a help center by ID.
func (m *Manager) GetHelpCenterByID(id int) (models.HelpCenter, error) {
	var hc models.HelpCenter
	if err := m.q.GetHelpCenterByID.Get(&hc, id); err != nil {
		m.lo.Error("error fetching help center", "error", err, "id", id)
		return hc, envelope.NewError(envelope.NotFoundError, m.i18n.Ts("globals.messages.notFound", "name", "help center"), nil)
	}
	return hc, nil
}

// GetHelpCenterBySlug retrieves a help center by slug.
func (m *Manager) GetHelpCenterBySlug(slug string) (models.HelpCenter, error) {
	var hc models.HelpCenter
	if err := m.q.GetHelpCenterBySlug.Get(&hc, slug); err != nil {
		m.lo.Error("error fetching help center by slug", "error", err, "slug", slug)
		return hc, envelope.NewError(envelope.NotFoundError, m.i18n.Ts("globals.messages.notFound", "name", "help center"), nil)
	}
	return hc, nil
}

// CreateHelpCenter creates a new help center.
func (m *Manager) CreateHelpCenter(req HelpCenterCreateRequest) (models.HelpCenter, error) {
	// Set default locale to 'en' if not provided
	defaultLocale := req.DefaultLocale
	if defaultLocale == "" {
		defaultLocale = "en"
	}

	var hc models.HelpCenter
	if err := m.q.InsertHelpCenter.Get(&hc, req.Name, req.Slug, req.PageTitle, defaultLocale); err != nil {
		if dbutil.IsUniqueViolationError(err) {
			return hc, envelope.NewError(envelope.ConflictError, m.i18n.Ts("globals.messages.errorAlreadyExists", "name", "help center slug"), nil)
		}
		m.lo.Error("error creating help center", "error", err)
		return hc, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorCreating", "name", "help center"), nil)
	}
	return hc, nil
}

// UpdateHelpCenter updates a help center.
func (m *Manager) UpdateHelpCenter(id int, req HelpCenterUpdateRequest) (models.HelpCenter, error) {
	// Set default locale to 'en' if not provided
	defaultLocale := req.DefaultLocale
	if defaultLocale == "" {
		defaultLocale = "en"
	}

	var hc models.HelpCenter
	if err := m.q.UpdateHelpCenter.Get(&hc, id, req.Name, req.Slug, req.PageTitle, defaultLocale); err != nil {
		if dbutil.IsUniqueViolationError(err) {
			return hc, envelope.NewError(envelope.ConflictError, m.i18n.Ts("globals.messages.errorAlreadyExists", "name", "help center slug"), nil)
		}
		m.lo.Error("error updating help center", "error", err, "id", id)
		return hc, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorUpdating", "name", "help center"), nil)
	}
	return hc, nil
}

// DeleteHelpCenter deletes a help center by ID.
func (m *Manager) DeleteHelpCenter(id int) error {
	if _, err := m.q.DeleteHelpCenter.Exec(id); err != nil {
		m.lo.Error("error deleting help center", "error", err, "id", id)
		return envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorDeleting", "name", "help center"), nil)
	}
	return nil
}

// Collections

// GetCollectionsByHelpCenter retrieves all collections for a help center.
func (m *Manager) GetCollectionsByHelpCenter(helpCenterID int) ([]models.Collection, error) {
	var collections = make([]models.Collection, 0)
	if err := m.q.GetCollectionsByHelpCenter.Select(&collections, helpCenterID); err != nil {
		m.lo.Error("error fetching collections", "error", err, "help_center_id", helpCenterID)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorFetching", "name", "collections"), nil)
	}
	return collections, nil
}

// GetCollectionsByHelpCenterAndLocale retrieves collections for a help center and locale.
func (m *Manager) GetCollectionsByHelpCenterAndLocale(helpCenterID int, locale string) ([]models.Collection, error) {
	var collections = make([]models.Collection, 0)
	if err := m.q.GetCollectionsByHelpCenterAndLocale.Select(&collections, helpCenterID, locale); err != nil {
		m.lo.Error("error fetching collections by locale", "error", err, "help_center_id", helpCenterID, "locale", locale)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorFetching", "name", "collections"), nil)
	}
	return collections, nil
}

// GetCollectionByID retrieves a collection by ID.
func (m *Manager) GetCollectionByID(id int) (models.Collection, error) {
	var collection models.Collection
	if err := m.q.GetCollectionByID.Get(&collection, id); err != nil {
		m.lo.Error("error fetching collection", "error", err, "id", id)
		return collection, envelope.NewError(envelope.NotFoundError, m.i18n.Ts("globals.messages.notFound", "name", "collection"), nil)
	}
	return collection, nil
}

// CreateCollection creates a new collection.
func (m *Manager) CreateCollection(helpCenterID int, req CollectionCreateRequest) (models.Collection, error) {
	// Validate depth if parent_id is provided
	if req.ParentID != nil {
		if err := m.validateCollectionDepth(*req.ParentID); err != nil {
			return models.Collection{}, err
		}
	}

	var collection models.Collection
	if err := m.q.InsertCollection.Get(&collection, helpCenterID, req.Slug, req.ParentID, req.Locale, req.Name, req.Description, req.SortOrder, req.IsPublished); err != nil {
		if dbutil.IsUniqueViolationError(err) {
			return collection, envelope.NewError(envelope.ConflictError, m.i18n.Ts("globals.messages.errorAlreadyExists", "name", "collection slug"), nil)
		}
		m.lo.Error("error creating collection", "error", err)
		return collection, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorCreating", "name", "collection"), nil)
	}
	return collection, nil
}

// UpdateCollection updates a collection.
func (m *Manager) UpdateCollection(id int, req CollectionUpdateRequest) (models.Collection, error) {
	// Validate depth if parent_id is provided and changing
	if req.ParentID != nil {
		if err := m.validateCollectionDepth(*req.ParentID); err != nil {
			return models.Collection{}, err
		}
	}

	var collection models.Collection
	if err := m.q.UpdateCollection.Get(&collection, id, req.Slug, req.ParentID, req.Locale, req.Name, req.Description, req.SortOrder, req.IsPublished); err != nil {
		if dbutil.IsUniqueViolationError(err) {
			return collection, envelope.NewError(envelope.ConflictError, m.i18n.Ts("globals.messages.errorAlreadyExists", "name", "collection slug"), nil)
		}
		m.lo.Error("error updating collection", "error", err, "id", id)
		return collection, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorUpdating", "name", "collection"), nil)
	}
	return collection, nil
}

// ToggleCollectionPublished toggles the published status of a collection.
func (m *Manager) ToggleCollectionPublished(id int) (models.Collection, error) {
	var collection models.Collection
	if err := m.q.ToggleCollectionPublished.Get(&collection, id); err != nil {
		m.lo.Error("error toggling collection published status", "error", err, "id", id)
		return collection, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorUpdating", "name", "collection"), nil)
	}
	return collection, nil
}

// DeleteCollection deletes a collection by ID.
func (m *Manager) DeleteCollection(id int) error {
	if _, err := m.q.DeleteCollection.Exec(id); err != nil {
		m.lo.Error("error deleting collection", "error", err, "id", id)
		return envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorDeleting", "name", "collection"), nil)
	}
	return nil
}

// ReorderCollections updates the sort order of multiple collections.
func (m *Manager) ReorderCollections(orders map[int]int) error {
	tx, err := m.db.Begin()
	if err != nil {
		return envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorUpdating", "name", "collections"), nil)
	}
	defer tx.Rollback()

	for id, order := range orders {
		if _, err := tx.Exec("UPDATE collections SET sort_order = $1, updated_at = NOW() WHERE id = $2", order, id); err != nil {
			m.lo.Error("error updating collection sort order", "error", err, "id", id, "order", order)
			return envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorUpdating", "name", "collections"), nil)
		}
	}

	if err := tx.Commit(); err != nil {
		m.lo.Error("error committing collection reorder transaction", "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorUpdating", "name", "collections"), nil)
	}

	return nil
}

// Articles

// GetArticlesByCollection retrieves all articles for a collection.
func (m *Manager) GetArticlesByCollection(collectionID int) ([]models.Article, error) {
	var articles = make([]models.Article, 0)
	if err := m.q.GetArticlesByCollection.Select(&articles, collectionID); err != nil {
		m.lo.Error("error fetching articles", "error", err, "collection_id", collectionID)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorFetching", "name", "articles"), nil)
	}
	return articles, nil
}

// GetArticlesByCollectionAndLocale retrieves articles for a collection and locale.
func (m *Manager) GetArticlesByCollectionAndLocale(collectionID int, locale string) ([]models.Article, error) {
	var articles = make([]models.Article, 0)
	if err := m.q.GetArticlesByCollectionAndLocale.Select(&articles, collectionID, locale); err != nil {
		m.lo.Error("error fetching articles by locale", "error", err, "collection_id", collectionID, "locale", locale)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorFetching", "name", "articles"), nil)
	}
	return articles, nil
}

// GetArticleByID retrieves an article by ID.
func (m *Manager) GetArticleByID(id int) (models.Article, error) {
	var article models.Article
	if err := m.q.GetArticleByID.Get(&article, id); err != nil {
		m.lo.Error("error fetching article", "error", err, "id", id)
		return article, envelope.NewError(envelope.NotFoundError, m.i18n.Ts("globals.messages.notFound", "name", "article"), nil)
	}
	return article, nil
}

// CreateArticle creates a new article.
func (m *Manager) CreateArticle(collectionID int, req ArticleCreateRequest) (models.Article, error) {
	// Validate status
	if !isValidArticleStatus(req.Status) {
		return models.Article{}, envelope.NewError(envelope.InputError, "Invalid article status", nil)
	}

	var article models.Article
	if err := m.q.InsertArticle.Get(&article, collectionID, req.Slug, req.Locale, req.Title, req.Content, req.SortOrder, req.Status, req.AIEnabled); err != nil {
		if dbutil.IsUniqueViolationError(err) {
			return article, envelope.NewError(envelope.ConflictError, m.i18n.Ts("globals.messages.errorAlreadyExists", "name", "article slug"), nil)
		}
		m.lo.Error("error creating article", "error", err)
		return article, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorCreating", "name", "article"), nil)
	}

	// Generate and save embeddings in a goroutine only if AI enabled
	if req.AIEnabled {
		go m.generateAndSaveEmbedding(article.ID, req.Title, req.Content)
	}

	return article, nil
}

// UpdateArticle updates an article.
func (m *Manager) UpdateArticle(id int, req ArticleUpdateRequest) (models.Article, error) {
	// Validate status
	if !isValidArticleStatus(req.Status) {
		return models.Article{}, envelope.NewError(envelope.InputError, "Invalid article status", nil)
	}

	var article models.Article

	// Check if collection_id is being changed
	if req.CollectionID != nil {
		// If collection is being moved, use MoveArticle first
		if err := m.q.MoveArticle.Get(&article, id, *req.CollectionID, req.SortOrder); err != nil {
			m.lo.Error("error moving article to new collection", "error", err, "id", id, "collection_id", *req.CollectionID)
			return article, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorUpdating", "name", "article"), nil)
		}
	}

	// Update the article with other fields
	if err := m.q.UpdateArticle.Get(&article, id, req.Slug, req.Locale, req.Title, req.Content, req.SortOrder, req.Status, req.AIEnabled); err != nil {
		if dbutil.IsUniqueViolationError(err) {
			return article, envelope.NewError(envelope.ConflictError, m.i18n.Ts("globals.messages.errorAlreadyExists", "name", "article slug"), nil)
		}
		m.lo.Error("error updating article", "error", err, "id", id)
		return article, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorUpdating", "name", "article"), nil)
	}

	// Handle embeddings based on AI enabled status
	if req.AIEnabled {
		// Generate and save embeddings in a goroutine
		go m.generateAndSaveEmbedding(article.ID, req.Title, req.Content)
	} else {
		// Remove embeddings if AI is disabled
		go m.removeEmbeddings(article.ID)
	}

	return article, nil
}

// UpdateArticleStatus updates the status of an article.
func (m *Manager) UpdateArticleStatus(id int, status string) (models.Article, error) {
	// Validate status
	if !isValidArticleStatus(status) {
		return models.Article{}, envelope.NewError(envelope.InputError, "Invalid article status", nil)
	}

	var article models.Article
	if err := m.q.UpdateArticleStatus.Get(&article, id, status); err != nil {
		m.lo.Error("error updating article status", "error", err, "id", id)
		return article, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorUpdating", "name", "article"), nil)
	}
	return article, nil
}

// DeleteArticle deletes an article by ID.
func (m *Manager) DeleteArticle(id int) error {
	if _, err := m.q.DeleteArticle.Exec(id); err != nil {
		m.lo.Error("error deleting article", "error", err, "id", id)
		return envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorDeleting", "name", "article"), nil)
	}
	return nil
}

// ReorderArticles updates the sort order of multiple articles.
func (m *Manager) ReorderArticles(orders map[int]int) error {
	tx, err := m.db.Begin()
	if err != nil {
		return envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorUpdating", "name", "articles"), nil)
	}
	defer tx.Rollback()

	for id, order := range orders {
		if _, err := tx.Exec("UPDATE articles SET sort_order = $1, updated_at = NOW() WHERE id = $2", order, id); err != nil {
			m.lo.Error("error updating article sort order", "error", err, "id", id, "order", order)
			return envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorUpdating", "name", "articles"), nil)
		}
	}

	if err := tx.Commit(); err != nil {
		m.lo.Error("error committing article reorder transaction", "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorUpdating", "name", "articles"), nil)
	}

	return nil
}

// SearchArticlesByVector performs vector similarity search on articles.
func (m *Manager) SearchArticlesByVector(helpCenterID int, embedding []float32, locale string, limit int) ([]models.ArticleSearchResult, error) {
	var results = make([]models.ArticleSearchResult, 0)

	// Convert []float32 to pgvector.Vector for PostgreSQL
	vector := pgvector.NewVector(embedding)
	if err := m.q.SearchArticlesByVector.Select(&results, helpCenterID, vector, locale, limit); err != nil {
		m.lo.Error("error searching articles by vector", "error", err, "help_center_id", helpCenterID)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorFetching", "name", "articles"), nil)
	}

	return results, nil
}

// SearchArticles performs semantic search on articles by converting query to embeddings.
func (m *Manager) SearchArticles(helpCenterID int, query string, locale string, limit int) ([]models.ArticleSearchResult, error) {
	// Generate embeddings for the search query
	embedding, err := m.ai.GetEmbeddings(query)
	if err != nil {
		m.lo.Error("error generating embeddings for search query", "error", err, "query", query)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorFetching", "name", "articles"), nil)
	}

	// Perform vector search
	return m.SearchArticlesByVector(helpCenterID, embedding, locale, limit)
}

// UpdateArticleEmbedding updates the embedding for an article.
func (m *Manager) UpdateArticleEmbedding(id int, embedding []float32) error {
	// Convert []float32 to pgvector.Vector for PostgreSQL
	vector := pgvector.NewVector(embedding)
	if _, err := m.q.UpdateArticleEmbedding.Exec(id, vector); err != nil {
		m.lo.Error("error updating article embedding", "error", err, "id", id)
		return envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorUpdating", "name", "article"), nil)
	}
	return nil
}

// GetHelpCenterTree returns the complete tree structure for a help center
func (m *Manager) GetHelpCenterTree(helpCenterID int, locale string) (models.TreeResponse, error) {
	// Get the help center info first
	helpCenter, err := m.GetHelpCenterByID(helpCenterID)
	if err != nil {
		return models.TreeResponse{}, err
	}

	// Get all tree data
	rows, err := m.q.GetHelpCenterTreeData.Query(helpCenterID, locale)
	if err != nil {
		m.lo.Error("error fetching tree data", "error", err, "help_center_id", helpCenterID)
		return models.TreeResponse{}, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorFetching", "name", "tree data"), nil)
	}
	defer rows.Close()

	// Parse the combined data
	collections := make(map[int]*models.TreeCollection)
	var collectionOrder []int

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

		err := rows.Scan(&itemType, &id, &createdAt, &updatedAt, &helpCenterID, &slug, &parentID, &locale, &name, &description, &sortOrder, &isPublished, &collectionID, &title, &content, &status, &viewCount, &aiEnabled)
		if err != nil {
			m.lo.Error("error scanning tree data", "error", err)
			return models.TreeResponse{}, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorFetching", "name", "tree data"), nil)
		}

		if itemType == "collection" {
			collection := &models.TreeCollection{
				Collection: models.Collection{
					ID:           id,
					CreatedAt:    createdAt,
					UpdatedAt:    updatedAt,
					HelpCenterID: helpCenterID,
					Slug:         slug,
					ParentID:     parentID,
					Locale:       locale,
					Name:         name,
					Description:  description,
					SortOrder:    sortOrder,
					IsPublished:  *isPublished,
				},
				Articles: make([]models.Article, 0),
				Children: make([]models.TreeCollection, 0),
			}
			collections[id] = collection
			collectionOrder = append(collectionOrder, id)
		} else if itemType == "article" && collectionID != nil {
			article := models.Article{
				ID:           id,
				CreatedAt:    createdAt,
				UpdatedAt:    updatedAt,
				CollectionID: *collectionID,
				Slug:         slug,
				Locale:       locale,
				Title:        *title,
				Content:      *content,
				SortOrder:    sortOrder,
				Status:       *status,
				ViewCount:    *viewCount,
				AIEnabled:    *aiEnabled,
			}

			if collection, exists := collections[*collectionID]; exists {
				collection.Articles = append(collection.Articles, article)
			}
		}
	}

	// Build the tree structure
	var buildTree func(parentID *int) []models.TreeCollection
	buildTree = func(parentID *int) []models.TreeCollection {
		children := make([]models.TreeCollection, 0)
		for _, col := range collections {
			if (col.ParentID == nil && parentID == nil) || (col.ParentID != nil && parentID != nil && *col.ParentID == *parentID) {
				// Recursively build children
				col.Children = buildTree(&col.ID)
				children = append(children, *col)
			}
		}
		return children
	}

	tree := buildTree(nil)

	response := models.TreeResponse{
		HelpCenter: helpCenter,
		Tree:       tree,
	}

	return response, nil
}

// SearchKnowledgeBase searches knowledge base (articles + knowledge sources) with fallback threshold
func (m *Manager) SearchKnowledgeBase(helpCenterID int, query string, locale string, threshold float64, limit int) ([]models.KnowledgeBaseResult, error) {
	// Generate embeddings for the search query
	embedding, err := m.ai.GetEmbeddings(query)
	if err != nil {
		m.lo.Error("error generating embeddings for knowledge base search", "error", err, "query", query)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorFetching", "name", "knowledge base"), nil)
	}

	var results = make([]models.KnowledgeBaseResult, 0)
	// Convert []float32 to pgvector.Vector for PostgreSQL
	vector := pgvector.NewVector(embedding)

	if err := m.q.SearchKnowledgeBase.Select(&results, helpCenterID, vector, locale, threshold, limit); err != nil {
		m.lo.Error("error searching knowledge base", "error", err, "query", query, "help_center_id", helpCenterID)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorFetching", "name", "knowledge base"), nil)
	}

	return results, nil
}

// Helper functions

// validateCollectionDepth checks if a parent collection would violate the 3-level depth limit.
func (m *Manager) validateCollectionDepth(parentID int) error {
	// Traverse up the parent chain to determine depth
	depth := 1
	currentID := parentID
	for currentID != 0 {
		parent, err := m.GetCollectionByID(currentID)
		if err != nil {
			return err
		}
		if parent.ParentID == nil {
			break
		}
		currentID = *parent.ParentID
		depth++
		if depth > 3 {
			return envelope.NewError(envelope.InputError, "Collections can only be nested up to 3 levels deep", nil)
		}
	}
	fmt.Printf("Collection depth is valid: %d\n", depth)
	return nil
}

// isValidArticleStatus checks if the given status is valid.
func isValidArticleStatus(status string) bool {
	return status == models.ArticleStatusDraft ||
		status == models.ArticleStatusPublished
}

// generateAndSaveEmbedding generates embeddings for an article and saves them asynchronously.
func (m *Manager) generateAndSaveEmbedding(articleID int, title, content string) {
	// Chunk HTML content into semantically meaningful pieces using configured parameters
	chunks, err := stringutil.ChunkHTMLContent(title, content, m.ai.GetChunkConfig())
	if err != nil {
		m.lo.Error("error chunking HTML content", "error", err, "article_id", articleID)
		return
	}

	// First, remove any existing embeddings for this article
	if _, err := m.q.DeleteEmbeddingsBySource.Exec("help_article", articleID); err != nil {
		m.lo.Error("error removing existing embeddings", "error", err, "article_id", articleID)
		// Continue anyway to insert new embeddings
	}

	// Generate embeddings for each chunk
	successfulChunks := 0
	fmt.Printf("Found %d chunks to process\n", len(chunks))
	for _, chunk := range chunks {
		// TODO: Remove after debugging.
		fmt.Println("Processing chunk")
		fmt.Printf("Chunk -> %s\n", chunk.Text)
		fmt.Println("Processing END =====================================================================================")


		// Generate embeddings using AI store
		embedding, err := m.ai.GetEmbeddings(chunk.Text)
		if err != nil {
			m.lo.Error("error generating chunk embedding", "error", err, "article_id", articleID, "chunk", chunk.ChunkIndex)
			continue
		}

		// Prepare metadata for the chunk
		metadata := map[string]any{
			"chunk_index":  chunk.ChunkIndex,
			"total_chunks": chunk.TotalChunks,
			"has_heading":  chunk.HasHeading,
			"has_code":     chunk.HasCode,
			"has_table":    chunk.HasTable,
		}

		// Convert metadata to JSON
		metadataJSON, err := json.Marshal(metadata)
		if err != nil {
			m.lo.Error("error marshaling chunk metadata", "error", err, "article_id", articleID, "chunk", chunk.ChunkIndex)
			metadataJSON = []byte(`{}`)
		}

		// Convert []float32 to pgvector.Vector for PostgreSQL
		vector := pgvector.NewVector(embedding)

		// Insert chunk embedding record
		if _, err := m.q.InsertEmbedding.Exec("help_article", articleID, chunk.Text, vector, string(metadataJSON)); err != nil {
			m.lo.Error("error saving chunk embedding", "error", err, "article_id", articleID, "chunk", chunk.ChunkIndex)
			continue
		}

		successfulChunks++
	}

	m.lo.Info("article chunked and embeddings generated", "article_id", articleID, "total_chunks", len(chunks), "successful_chunks", successfulChunks)
}

// removeEmbeddings removes embeddings for an article asynchronously.
func (m *Manager) removeEmbeddings(articleID int) {
	if _, err := m.q.DeleteEmbeddingsBySource.Exec("help_article", articleID); err != nil {
		m.lo.Error("error removing embeddings", "error", err, "article_id", articleID)
		return
	}
	m.lo.Info("embeddings removed", "article_id", articleID)
}
