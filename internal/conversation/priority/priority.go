// Package priority handles the management of conversation priorities.
package priority

import (
	"embed"

	"github.com/abhinavxd/libredesk/internal/conversation/priority/models"
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

// Manager handles changes to priorities.
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
	GetAll *sqlx.Stmt `query:"get-all"`
	Get    *sqlx.Stmt `query:"get"`
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

// GetAll retrieves all priorities.
func (m *Manager) GetAll() ([]models.Priority, error) {
	var priorities = make([]models.Priority, 0)
	if err := m.q.GetAll.Select(&priorities); err != nil {
		m.lo.Error("error fetching priorities", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorFetching", "name", m.i18n.P("globals.entities.priority")), nil)
	}
	return priorities, nil
}

// Get retrieves a priority by ID.
func (m *Manager) Get(id int) (models.Priority, error) {
	var priority models.Priority
	if err := m.q.Get.Get(&priority, id); err != nil {
		m.lo.Error("error fetching priority", "error", err)
		return priority, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorFetching", "name", m.i18n.S("globals.entities.priority")), nil)
	}
	return priority, nil
}
