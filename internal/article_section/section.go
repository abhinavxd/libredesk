// Package article_section handles the management of sections.
package article_section

import (
	"embed"

	"github.com/abhinavxd/libredesk/internal/article_section/models"
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
	GetAllSections         *sqlx.Stmt `query:"get-all-sections"`
	InsertSection          *sqlx.Stmt `query:"insert-section"`
	DeleteSection          *sqlx.Stmt `query:"delete-section"`
	UpdateSection          *sqlx.Stmt `query:"update-section"`
	GetSectionsWithCategory *sqlx.Stmt `query:"get-sections-with-category"`
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

// GetAll retrieves all sections.
func (t *Manager) GetAll() ([]models.Section, error) {
	var sections = make([]models.Section, 0)
	if err := t.q.GetAllSections.Select(&sections); err != nil {
		t.lo.Error("error fetching sections", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, t.i18n.Ts("globals.messages.errorFetching", "name", t.i18n.P("globals.terms.section")), nil)
	}
	return sections, nil
}

// Create creates a new section.
func (t *Manager) Create(section models.Section) error {
	if _, err := t.q.InsertSection.Exec(section.Name, section.CategoryID); err != nil {
		if dbutil.IsUniqueViolationError(err) {
			return envelope.NewError(envelope.ConflictError, t.i18n.Ts("globals.messages.errorAlreadyExists", "name", "{globals.terms.section}"), nil)
		}
		t.lo.Error("error inserting section", "error", err)
		return envelope.NewError(envelope.GeneralError, t.i18n.Ts("globals.messages.errorCreating", "name", "{globals.terms.section}"), nil)
	}
	return nil
}

// Delete deletes a section by ID.
func (t *Manager) Delete(id int) error {
	if _, err := t.q.DeleteSection.Exec(id); err != nil {
		t.lo.Error("error deleting section", "error", err)
		return envelope.NewError(envelope.GeneralError, t.i18n.Ts("globals.messages.errorDeleting", "name", "{globals.terms.section}"), nil)
	}
	return nil
}

// Update updates a section by id.
func (t *Manager) Update(id int, section models.Section) error {
	if _, err := t.q.UpdateSection.Exec(id, section.Name, section.CategoryID); err != nil {
		t.lo.Error("error updating section", "error", err)
		return envelope.NewError(envelope.GeneralError, t.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.terms.section}"), nil)
	}
	return nil
}

// GetSectionsWithCategory retrieves a all sections with its category information
func (t *Manager) GetSectionsWithCategory() ([]models.SectionWithCategory, error) {
    var sections = make([]models.SectionWithCategory, 0)
    if err := t.q.GetSectionsWithCategory.Select(&sections); err != nil {
        t.lo.Error("error fetching sections with categories", "error", err)
        return nil, envelope.NewError(envelope.GeneralError, t.i18n.Ts("globals.messages.errorFetching", "name", t.i18n.P("globals.terms.section")), nil)
    }
    return sections, nil
}
