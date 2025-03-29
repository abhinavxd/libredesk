// Package view handles the management of conversation views.
package view

import (
	"database/sql"
	"embed"

	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/view/models"
	"github.com/jmoiron/sqlx"
	"github.com/knadh/go-i18n"
	"github.com/zerodha/logf"
)

var (
	//go:embed queries.sql
	efs embed.FS
)

// Manager manages views.
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
	GetView      *sqlx.Stmt `query:"get-view"`
	GetUserViews *sqlx.Stmt `query:"get-user-views"`
	InsertView   *sqlx.Stmt `query:"insert-view"`
	DeleteView   *sqlx.Stmt `query:"delete-view"`
	UpdateView   *sqlx.Stmt `query:"update-view"`
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

// Get returns a view by ID.
func (v *Manager) Get(id int) (models.View, error) {
	var view = models.View{}
	if err := v.q.GetView.Get(&view, id); err != nil {
		if err == sql.ErrNoRows {
			return view, envelope.NewError(envelope.NotFoundError, v.i18n.Ts("globals.messages.notFound", "name", "{globals.entities.view}"), nil)
		}
		v.lo.Error("error fetching view", "error", err)
		return view, envelope.NewError(envelope.GeneralError, v.i18n.Ts("globals.messages.errorFetching", "name", "{globals.entities.view}"), nil)
	}
	return view, nil
}

// GetUsersViews returns all views for a user.
func (v *Manager) GetUsersViews(userID int) ([]models.View, error) {
	views := make([]models.View, 0)
	if err := v.q.GetUserViews.Select(&views, userID); err != nil {
		v.lo.Error("error fetching views", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, v.i18n.Ts("globals.messages.errorFetching", "name", "{globals.entities.view}"), nil)
	}
	return views, nil
}

// Create creates a new view.
func (v *Manager) Create(name string, filter []byte, userID int) error {
	if _, err := v.q.InsertView.Exec(name, filter, userID); err != nil {
		v.lo.Error("error inserting view", "error", err)
		return envelope.NewError(envelope.GeneralError, v.i18n.Ts("globals.messages.errorCreating", "name", "{globals.entities.view}"), nil)
	}
	return nil
}

// Update updates a view by id.
func (v *Manager) Update(id int, name string, filter []byte) error {
	if _, err := v.q.UpdateView.Exec(id, name, filter); err != nil {
		v.lo.Error("error updating view", "error", err)
		return envelope.NewError(envelope.GeneralError, v.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.entities.view}"), nil)
	}
	return nil
}

// Delete deletes a view by ID.
func (v *Manager) Delete(id int) error {
	if _, err := v.q.DeleteView.Exec(id); err != nil {
		v.lo.Error("error deleting view", "error", err)
		return envelope.NewError(envelope.GeneralError, v.i18n.Ts("globals.messages.errorDeleting", "name", "{globals.entities.view}"), nil)
	}
	return nil
}
