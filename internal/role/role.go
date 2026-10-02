// Package role provides role lookup for mailbox users.
package role

import (
	"embed"

	"github.com/jakedolan443/fernmail/internal/dbutil"
	"github.com/jakedolan443/fernmail/internal/envelope"
	"github.com/jakedolan443/fernmail/internal/role/models"
	"github.com/jmoiron/sqlx"
	"github.com/knadh/go-i18n"

	"github.com/zerodha/logf"
)

var (
	//go:embed queries.sql
	efs embed.FS
)

// Manager handles role-related operations.
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

// GetAll retrieves all roles.
func (u *Manager) GetAll() ([]models.Role, error) {
	var roles = make([]models.Role, 0)
	if err := u.q.GetAll.Select(&roles); err != nil {
		u.lo.Error("error fetching roles", "error", err)
		return roles, envelope.NewError(envelope.GeneralError, u.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return roles, nil
}
