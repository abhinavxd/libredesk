// Package tag handles the management of tags.
package tag

import (
	"embed"

	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/tag/models"
	"github.com/jmoiron/sqlx"
	"github.com/knadh/go-i18n"
	"github.com/zerodha/logf"
)

var (
	//go:embed queries.sql
	efs embed.FS
)

// Manager manages tags.
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
	GetAllTags *sqlx.Stmt `query:"get-all-tags"`
	InsertTag  *sqlx.Stmt `query:"insert-tag"`
	DeleteTag  *sqlx.Stmt `query:"delete-tag"`
	UpdateTag  *sqlx.Stmt `query:"update-tag"`
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

// GetAll retrieves all tags.
func (t *Manager) GetAll() ([]models.Tag, error) {
	var tags = make([]models.Tag, 0)
	if err := t.q.GetAllTags.Select(&tags); err != nil {
		t.lo.Error("error fetching tags", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, t.i18n.Ts("globals.messages.errorFetching", "name", t.i18n.P("globals.entities.tag")), nil)
	}
	return tags, nil
}

// Create creates a new tag.
func (t *Manager) Create(name string) error {
	if _, err := t.q.InsertTag.Exec(name); err != nil {
		if dbutil.IsUniqueViolationError(err) {
			return envelope.NewError(envelope.ConflictError, t.i18n.Ts("globals.messages.mismatch", "name", "{globals.entities.tag}"), nil)
		}
		t.lo.Error("error inserting tag", "error", err)
		return envelope.NewError(envelope.GeneralError, t.i18n.Ts("globals.messages.errorCreating", "name", "{globals.entities.tag}"), nil)
	}
	return nil
}

// Delete deletes a tag by ID.
func (t *Manager) Delete(id int) error {
	if _, err := t.q.DeleteTag.Exec(id); err != nil {
		t.lo.Error("error deleting tag", "error", err)
		return envelope.NewError(envelope.GeneralError, t.i18n.Ts("globals.messages.errorDeleting", "name", "{globals.entities.tag}"), nil)
	}
	return nil
}

// Update updates a tag by id.
func (t *Manager) Update(id int, name string) error {
	if _, err := t.q.UpdateTag.Exec(id, name); err != nil {
		t.lo.Error("error updating tag", "error", err)
		return envelope.NewError(envelope.GeneralError, t.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.entities.tag}"), nil)
	}
	return nil
}
