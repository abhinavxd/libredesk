// Package team provides team lookup and membership updates for mailbox users.
package team

import (
	"database/sql"
	"embed"
	"errors"

	"github.com/jakedolan443/fernmail/internal/dbutil"
	"github.com/jakedolan443/fernmail/internal/envelope"
	"github.com/jakedolan443/fernmail/internal/team/models"
	"github.com/jmoiron/sqlx"
	"github.com/knadh/go-i18n"
	"github.com/lib/pq"
	"github.com/zerodha/logf"
)

var (
	//go:embed queries.sql
	efs embed.FS
)

// Manager handles team-related operations.
type Manager struct {
	lo   *logf.Logger
	i18n *i18n.I18n
	q    queries
}

// Opts contains options for initializing the Manager.
type Opts struct {
	DB   *sqlx.DB
	Lo   *logf.Logger
	I18n *i18n.I18n
}

// queries contains prepared SQL queries.
type queries struct {
	GetTeams        *sqlx.Stmt `query:"get-teams"`
	UpsertUserTeams *sqlx.Stmt `query:"upsert-user-teams"`
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

// GetAll retrieves all teams.
func (u *Manager) GetAll() ([]models.Team, error) {
	var teams = make([]models.Team, 0)
	if err := u.q.GetTeams.Select(&teams); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return teams, nil
		}
		u.lo.Error("error fetching teams", "error", err)
		return teams, envelope.NewError(envelope.GeneralError, u.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return teams, nil
}

// UpsertUserTeams updates/inserts exists user teams
func (u *Manager) UpsertUserTeams(id int, teamNames []string) error {
	if _, err := u.q.UpsertUserTeams.Exec(id, pq.Array(teamNames)); err != nil {
		u.lo.Error("error updating user teams", "error", err)
		return envelope.NewError(envelope.GeneralError, u.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return nil
}
