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
	"github.com/lib/pq"
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
	GetView               *sqlx.Stmt `query:"get-view"`
	GetUserViews          *sqlx.Stmt `query:"get-user-views"`
	GetSharedViewsForUser *sqlx.Stmt `query:"get-shared-views-for-user"`
	GetAllSharedViews     *sqlx.Stmt `query:"get-all-shared-views"`
	InsertView            *sqlx.Stmt `query:"insert-view"`
	DeleteView            *sqlx.Stmt `query:"delete-view"`
	UpdateView            *sqlx.Stmt `query:"update-view"`
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
			return view, envelope.NewError(envelope.NotFoundError, v.i18n.Ts("globals.messages.notFound", "name", "{globals.terms.view}"), nil)
		}
		v.lo.Error("error fetching view", "error", err)
		return view, envelope.NewError(envelope.GeneralError, v.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.view}"), nil)
	}
	return view, nil
}

// GetUsersViews returns all personal views (visibility='user') for a user.
func (v *Manager) GetUsersViews(userID int) ([]models.View, error) {
	views := make([]models.View, 0)
	if err := v.q.GetUserViews.Select(&views, userID); err != nil {
		v.lo.Error("error fetching views", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, v.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.view}"), nil)
	}
	return views, nil
}

// GetSharedViewsForUser returns shared views accessible to a user based on their team memberships.
func (v *Manager) GetSharedViewsForUser(teamIDs []int) ([]models.View, error) {
	views := make([]models.View, 0)
	if err := v.q.GetSharedViewsForUser.Select(&views, pq.Array(teamIDs)); err != nil {
		v.lo.Error("error fetching shared views", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, v.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.view}"), nil)
	}
	return views, nil
}

// GetAllSharedViews returns all shared views (for admin management).
func (v *Manager) GetAllSharedViews() ([]models.View, error) {
	views := make([]models.View, 0)
	if err := v.q.GetAllSharedViews.Select(&views); err != nil {
		v.lo.Error("error fetching all shared views", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, v.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.view}"), nil)
	}
	return views, nil
}

// Create creates a new view (personal view with visibility='user').
func (v *Manager) Create(name string, filter []byte, userID int) (models.View, error) {
	var createdView models.View
	if err := v.q.InsertView.Get(&createdView, name, filter, models.VisibilityUser, userID, nil); err != nil {
		v.lo.Error("error inserting view", "error", err)
		return models.View{}, envelope.NewError(envelope.GeneralError, v.i18n.Ts("globals.messages.errorCreating", "name", "{globals.terms.view}"), nil)
	}
	return createdView, nil
}

// CreateSharedView creates a new shared view (admin only).
func (v *Manager) CreateSharedView(name string, filter []byte, visibility string, teamID *int) (models.View, error) {
	var createdView models.View
	if err := v.q.InsertView.Get(&createdView, name, filter, visibility, nil, teamID); err != nil {
		v.lo.Error("error inserting shared view", "error", err)
		return models.View{}, envelope.NewError(envelope.GeneralError, v.i18n.Ts("globals.messages.errorCreating", "name", "{globals.terms.view}"), nil)
	}
	return createdView, nil
}

// Update updates a personal view by id.
func (v *Manager) Update(id int, name string, filter []byte, userID int) (models.View, error) {
	var updatedView models.View
	if err := v.q.UpdateView.Get(&updatedView, id, name, filter, models.VisibilityUser, userID, nil); err != nil {
		v.lo.Error("error updating view", "error", err)
		return models.View{}, envelope.NewError(envelope.GeneralError, v.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.terms.view}"), nil)
	}
	return updatedView, nil
}

// UpdateSharedView updates a shared view.
func (v *Manager) UpdateSharedView(id int, name string, filter []byte, visibility string, teamID *int) (models.View, error) {
	var updatedView models.View
	if err := v.q.UpdateView.Get(&updatedView, id, name, filter, visibility, nil, teamID); err != nil {
		v.lo.Error("error updating shared view", "error", err)
		return models.View{}, envelope.NewError(envelope.GeneralError, v.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.terms.view}"), nil)
	}
	return updatedView, nil
}

// Delete deletes a view by ID.
func (v *Manager) Delete(id int) error {
	if _, err := v.q.DeleteView.Exec(id); err != nil {
		v.lo.Error("error deleting view", "error", err)
		return envelope.NewError(envelope.GeneralError, v.i18n.Ts("globals.messages.errorDeleting", "name", "{globals.terms.view}"), nil)
	}
	return nil
}
