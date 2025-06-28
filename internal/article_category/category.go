// Package article_category handles the management of categories.
package article_category

import (
	"embed"

	"github.com/abhinavxd/libredesk/internal/article_category/models"
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
	GetAllCategories             *sqlx.Stmt `query:"get-all-categories"`
	GetAllCategoriesWithSections *sqlx.Stmt `query:"get-all-categories-with-sections"`
	InsertCategory               *sqlx.Stmt `query:"insert-category"`
	DeleteCategory               *sqlx.Stmt `query:"delete-category"`
	UpdateCategory               *sqlx.Stmt `query:"update-category"`
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

// GetAll retrieves all categories.
func (t *Manager) GetAll() ([]models.Category, error) {
	var categories = make([]models.Category, 0)
	if err := t.q.GetAllCategories.Select(&categories); err != nil {
		t.lo.Error("error fetching categories", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, t.i18n.Ts("globals.messages.errorFetching", "name", t.i18n.P("globals.terms.category")), nil)
	}
	return categories, nil
}

// Create creates a new category.
func (t *Manager) Create(name string) error {
	if _, err := t.q.InsertCategory.Exec(name); err != nil {
		if dbutil.IsUniqueViolationError(err) {
			return envelope.NewError(envelope.ConflictError, t.i18n.Ts("globals.messages.errorAlreadyExists", "name", "{globals.terms.category}"), nil)
		}
		t.lo.Error("error inserting category", "error", err)
		return envelope.NewError(envelope.GeneralError, t.i18n.Ts("globals.messages.errorCreating", "name", "{globals.terms.category}"), nil)
	}
	return nil
}

// Delete deletes a category by ID.
func (t *Manager) Delete(id int) error {
	if _, err := t.q.DeleteCategory.Exec(id); err != nil {
		t.lo.Error("error deleting category", "error", err)
		return envelope.NewError(envelope.GeneralError, t.i18n.Ts("globals.messages.errorDeleting", "name", "{globals.terms.category}"), nil)
	}
	return nil
}

// Update updates a category by id.
func (t *Manager) Update(id int, name string) error {
	if _, err := t.q.UpdateCategory.Exec(id, name); err != nil {
		t.lo.Error("error updating category", "error", err)
		return envelope.NewError(envelope.GeneralError, t.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.terms.category}"), nil)
	}
	return nil
}

// GetAllWithSections retrieves all categories with their sections.
func (t *Manager) GetAllWithSections() ([]models.CategoryWithSections, error) {
	var categories = make([]models.CategoryWithSections, 0)
	if err := t.q.GetAllCategoriesWithSections.Select(&categories); err != nil {
		t.lo.Error("error fetching categories with sections", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, t.i18n.Ts("globals.messages.errorFetching", "name", t.i18n.P("globals.terms.category")), nil)
	}
	return categories, nil
}
