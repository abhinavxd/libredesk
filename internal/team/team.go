// Package team handles the management of teams and their members.
package team

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"

	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/team/models"
	umodels "github.com/abhinavxd/libredesk/internal/user/models"
	"github.com/jmoiron/sqlx"
	"github.com/knadh/go-i18n"
	"github.com/lib/pq"
	"github.com/volatiletech/null/v9"
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
	GetTeams          *sqlx.Stmt `query:"get-teams"`
	GetUserTeams      *sqlx.Stmt `query:"get-user-teams"`
	GetTeamsCompact   *sqlx.Stmt `query:"get-teams-compact"`
	GetTeam           *sqlx.Stmt `query:"get-team"`
	InsertTeam        *sqlx.Stmt `query:"insert-team"`
	UpdateTeam        *sqlx.Stmt `query:"update-team"`
	DeleteTeam        *sqlx.Stmt `query:"delete-team"`
	GetTeamMembers    *sqlx.Stmt `query:"get-team-members"`
	UpsertUserTeams   *sqlx.Stmt `query:"upsert-user-teams"`
	UserBelongsToTeam *sqlx.Stmt `query:"user-belongs-to-team"`
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
		return teams, envelope.NewError(envelope.GeneralError, u.i18n.Ts("globals.messages.errorFetching", "name", "{globals.entities.team}"), nil)
	}
	return teams, nil
}

// GetAllCompact retrieves all teams with limited fields.
func (u *Manager) GetAllCompact() ([]models.Team, error) {
	var teams = make([]models.Team, 0)
	if err := u.q.GetTeamsCompact.Select(&teams); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return teams, nil
		}
		u.lo.Error("error fetching teams", "error", err)
		return teams, envelope.NewError(envelope.GeneralError, u.i18n.Ts("globals.messages.errorFetching", "name", "{globals.entities.team}"), nil)
	}
	return teams, nil
}

// Get retrieves a team by ID.
func (u *Manager) Get(id int) (models.Team, error) {
	var team models.Team
	if err := u.q.GetTeam.Get(&team, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			u.lo.Error("team not found", "id", id, "error", err)
			return team, envelope.NewError(envelope.InputError, u.i18n.Ts("globals.messages.notFound", "name", "{globals.entities.team}"), nil)
		}
		u.lo.Error("error fetching team", "id", id, "error", err)
		return team, envelope.NewError(envelope.GeneralError, u.i18n.Ts("globals.messages.errorFetching", "name", "{globals.entities.team}"), nil)
	}
	return team, nil
}

// Create creates a new team.
func (u *Manager) Create(name, timezone, conversationAssignmentType string, businessHrsID, slaPolicyID null.Int, emoji string, maxAutoAssignedConversations int) error {
	if _, err := u.q.InsertTeam.Exec(name, timezone, conversationAssignmentType, businessHrsID, slaPolicyID, emoji, maxAutoAssignedConversations); err != nil {
		if dbutil.IsUniqueViolationError(err) {
			return envelope.NewError(envelope.GeneralError, u.i18n.Ts("globals.messages.errorAlreadyExists", "name", "{globals.entities.team}"), nil)
		}
		u.lo.Error("error inserting team", "error", err)
		return envelope.NewError(envelope.GeneralError, u.i18n.Ts("globals.messages.errorCreating", "name", "{globals.entities.team}"), nil)
	}
	return nil
}

// Update updates an existing team.
func (u *Manager) Update(id int, name, timezone, conversationAssignmentType string, businessHrsID, slaPolicyID null.Int, emoji string, maxAutoAssignedConversations int) error {
	if _, err := u.q.UpdateTeam.Exec(id, name, timezone, conversationAssignmentType, businessHrsID, slaPolicyID, emoji, maxAutoAssignedConversations); err != nil {
		u.lo.Error("error updating team", "error", err)
		return envelope.NewError(envelope.GeneralError, u.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.entities.team}"), nil)
	}
	return nil
}

// Delete deletes a team by ID also deletes all the team members and unassigns all the conversations belonging to the team.
func (u *Manager) Delete(id int) error {
	if _, err := u.q.DeleteTeam.Exec(id); err != nil {
		u.lo.Error("error deleting team", "error", err)
		return envelope.NewError(envelope.GeneralError, u.i18n.Ts("globals.messages.errorDeleting", "name", "{globals.entities.team}"), nil)
	}
	return nil
}

// GetUserTeams retrieves teams of a user by user ID.
func (u *Manager) GetUserTeams(userID int) ([]models.Team, error) {
	var teams = make([]models.Team, 0)
	if err := u.q.GetUserTeams.Select(&teams, userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return teams, nil
		}
		u.lo.Error("error fetching teams", "user_id", userID, "error", err)
		return teams, envelope.NewError(envelope.GeneralError, u.i18n.Ts("globals.messages.errorFetching", "name", "{globals.entities.team}"), nil)
	}
	return teams, nil
}

// UpsertUserTeams updates/inserts exists user teams
func (u *Manager) UpsertUserTeams(id int, teamNames []string) error {
	if _, err := u.q.UpsertUserTeams.Exec(id, pq.Array(teamNames)); err != nil {
		u.lo.Error("error updating user teams", "error", err)
		return envelope.NewError(envelope.GeneralError, u.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.entities.team}"), nil)
	}
	return nil
}

// UserBelongsToTeam returns true if the user belongs to the team.
func (u *Manager) UserBelongsToTeam(teamID, userID int) (bool, error) {
	var exists bool
	if err := u.q.UserBelongsToTeam.Get(&exists, teamID, userID); err != nil {
		u.lo.Error("error fetching team members", "team_id", teamID, "user_id", userID, "error", err)
		return false, envelope.NewError(envelope.GeneralError, u.i18n.Ts("globals.messages.errorFetching", "name", "{globals.entities.team}"), nil)
	}
	return exists, nil
}

// GetMembers retrieves members of a team.
func (u *Manager) GetMembers(id int) ([]umodels.User, error) {
	var users []umodels.User
	if err := u.q.GetTeamMembers.Select(&users, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return users, nil
		}
		u.lo.Error("error fetching team members", "team_id", id, "error", err)
		return users, fmt.Errorf("fetching team members: %w", err)
	}
	return users, nil
}
