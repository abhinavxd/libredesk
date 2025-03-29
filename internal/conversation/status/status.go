// Package status handles the management of conversation statuses.
package status

import (
	"embed"
	"slices"

	"github.com/abhinavxd/libredesk/internal/conversation/status/models"
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

// Manager handles changes to statuses.
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
	GetStatus      *sqlx.Stmt `query:"get-status"`
	GetAllStatuses *sqlx.Stmt `query:"get-all-statuses"`
	InsertStatus   *sqlx.Stmt `query:"insert-status"`
	DeleteStatus   *sqlx.Stmt `query:"delete-status"`
	UpdateStatus   *sqlx.Stmt `query:"update-status"`
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

// GetAll retrieves all statuses.
func (m *Manager) GetAll() ([]models.Status, error) {
	var statuses = make([]models.Status, 0)
	if err := m.q.GetAllStatuses.Select(&statuses); err != nil {
		m.lo.Error("error fetching statuses", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorFetching", "name", m.i18n.P("globals.entities.status")), nil)
	}
	return statuses, nil
}

// Create creates a new status.
func (m *Manager) Create(name string) error {
	if _, err := m.q.InsertStatus.Exec(name); err != nil {
		m.lo.Error("error inserting status", "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorCreating", "name", m.i18n.T("globals.entities.status")), nil)
	}
	return nil
}

// Delete deletes a status by ID.
func (m *Manager) Delete(id int) error {
	// Disallow deletion of default statuses.
	status, err := m.Get(id)
	if err != nil {
		return envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorFetching", "name", m.i18n.T("globals.entities.status")), nil)
	}

	if slices.Contains(models.DefaultStatuses, status.Name) {
		return envelope.NewError(envelope.InputError, m.i18n.T("conversationStatus.cannotUpdateDefault"), nil)
	}

	if _, err := m.q.DeleteStatus.Exec(id); err != nil {
		if dbutil.IsForeignKeyError(err) {
			return envelope.NewError(envelope.InputError, m.i18n.T("converstionStatus.cannotBeDeleted"), nil)
		}
		m.lo.Error("error deleting status", "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorDeleting", "name", m.i18n.T("globals.entities.status")), nil)
	}
	return nil
}

// Update updates a status by id.
func (m *Manager) Update(id int, name string) error {
	// Disallow updating of default statuses.
	status, err := m.Get(id)
	if err != nil {
		return envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorFetching", "name", m.i18n.Ts("globals.entities.status")), nil)
	}

	if slices.Contains(models.DefaultStatuses, status.Name) {
		return envelope.NewError(envelope.InputError, m.i18n.T("conversationStatus.cannotUpdateDefault"), nil)
	}

	if _, err := m.q.UpdateStatus.Exec(id, name); err != nil {
		m.lo.Error("error updating status", "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorUpdating", "name", m.i18n.Ts("globals.entities.status")), nil)
	}
	return nil
}

// Get retrieves a status by ID.
func (m *Manager) Get(id int) (models.Status, error) {
	var status models.Status
	if err := m.q.GetStatus.Get(&status, id); err != nil {
		m.lo.Error("error fetching status", "error", err)
		return status, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorFetching", "name", m.i18n.Ts("globals.entities.status")), nil)
	}
	return status, nil
}
