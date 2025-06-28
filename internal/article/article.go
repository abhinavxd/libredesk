// Package article handles the management of articles.
package article

import (
	"embed"

	"github.com/abhinavxd/libredesk/internal/article/models"
	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/jmoiron/sqlx"
	"github.com/knadh/go-i18n"
	"github.com/zerodha/logf"
)

var (
	//go:embed queries.sql
	efs embed.FS
)

type Manager struct {
	q    queries
	lo   *logf.Logger
	i18n *i18n.I18n
}

// Opts contains options for initializing the Manager.
type Opts struct {
	DB   *sqlx.DB
	Lo   *logf.Logger
	I18n *i18n.I18n
}

// queries contains prepared SQL queries.
type queries struct {
	InsertArticle         *sqlx.Stmt `query:"insert-article"`
	DeleteArticle         *sqlx.Stmt `query:"delete-article"`
	UpdateArticle         *sqlx.Stmt `query:"update-article"`
	GetAllArticles        *sqlx.Stmt `query:"get-all-articles"`
	GetArticleByID        *sqlx.Stmt `query:"get-article-by-id"`
	GetArticlesBySectionID *sqlx.Stmt `query:"get-articles-by-section-id"`
	PublishArticle        *sqlx.Stmt `query:"publish-article"`
	UnpublishArticle      *sqlx.Stmt `query:"unpublish-article"`
}

// New creates and returns a new instance of the Manager.
func New(opts Opts) (*Manager, error) {
	var q queries

	if err := dbutil.ScanSQLFile("queries.sql", &q, opts.DB, efs); err != nil {
		return nil, err
	}

	return &Manager{
		q:    q,
		lo:   opts.Lo,
		i18n: opts.I18n,
	}, nil
}

// GetAll retrieves all articles with section and category information.
func (m *Manager) GetAll() ([]models.ArticleWithDetails, error) {
	var articles = make([]models.ArticleWithDetails, 0)
	if err := m.q.GetAllArticles.Select(&articles); err != nil {
		m.lo.Error("error fetching articles", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorFetching", "name", m.i18n.P("globals.terms.article")), nil)
	}
	return articles, nil
}

// Create creates a new article.
func (m *Manager) Create(title, content string, sectionID int) (int, error) {
	var id int
	if err := m.q.InsertArticle.Get(&id, title, content, sectionID); err != nil {
		m.lo.Error("error inserting article", "error", err)
		return 0, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorCreating", "name", "{globals.terms.article}"), nil)
	}
	return id, nil
}

// Delete deletes an article by ID.
func (m *Manager) Delete(id int) error {
	if _, err := m.q.DeleteArticle.Exec(id); err != nil {
		m.lo.Error("error deleting article", "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorDeleting", "name", "{globals.terms.article}"), nil)
	}
	return nil
}

// Update updates an article by id.
func (m *Manager) Update(id int, title, content string, sectionID int) error {
	if _, err := m.q.UpdateArticle.Exec(id, title, content, sectionID); err != nil {
		m.lo.Error("error updating article", "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.terms.article}"), nil)
	}
	return nil
}

// GetByID retrieves an article by its ID.
func (m *Manager) GetByID(id int) (*models.Article, error) {
	var article models.Article
	if err := m.q.GetArticleByID.Get(&article, id); err != nil {
		m.lo.Error("error fetching article", "error", err, "id", id)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorFetching", "name", m.i18n.P("globals.terms.article")), nil)
	}
	return &article, nil
}

// GetBySectionID retrieves all articles for a specific section.
func (m *Manager) GetBySectionID(sectionID int) ([]models.Article, error) {
	var articles = make([]models.Article, 0)
	if err := m.q.GetArticlesBySectionID.Select(&articles, sectionID); err != nil {
		m.lo.Error("error fetching articles by section", "error", err, "section_id", sectionID)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorFetching", "name", m.i18n.P("globals.terms.article")), nil)
	}
	return articles, nil
}

// Publish publishes an article by setting is_published to true and setting published_at.
func (m *Manager) Publish(id int) error {
	if _, err := m.q.PublishArticle.Exec(id); err != nil {
		m.lo.Error("error publishing article", "error", err, "id", id)
		return envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.terms.article}"), nil)
	}
	return nil
}

// Unpublish unpublishes an article by setting is_published to false and clearing published_at.
func (m *Manager) Unpublish(id int) error {
	if _, err := m.q.UnpublishArticle.Exec(id); err != nil {
		m.lo.Error("error unpublishing article", "error", err, "id", id)
		return envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.terms.article}"), nil)
	}
	return nil
}