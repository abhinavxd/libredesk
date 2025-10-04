// Package activity manages activity logs for all users.
package activitylog

import (
	"context"
	"database/sql"
	"embed"
	"fmt"

	"github.com/abhinavxd/libredesk/internal/activity_log/models"
	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/abhinavxd/libredesk/internal/envelope"
	umodels "github.com/abhinavxd/libredesk/internal/user/models"
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
	db   *sqlx.DB
}

// Opts contains options for initializing the Manager.
type Opts struct {
	DB   *sqlx.DB
	Lo   *logf.Logger
	I18n *i18n.I18n
}

// queries contains prepared SQL queries.
type queries struct {
	GetAllActivities string     `query:"get-all-activities"`
	InsertActivity   *sqlx.Stmt `query:"insert-activity"`
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
		db:   opts.DB,
	}, nil
}

// GetAll retrieves all activity logs.
func (m *Manager) GetAll(order, orderBy, filtersJSON string, page, pageSize int) ([]models.ActivityLog, error) {
	query, qArgs, err := m.makeQuery(page, pageSize, order, orderBy, filtersJSON)
	if err != nil {
		m.lo.Error("error creating activity log list query", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.activityLog}"), nil)
	}

	// Start a read-only txn.
	tx, err := m.db.BeginTxx(context.Background(), &sql.TxOptions{
		ReadOnly: true,
	})
	if err != nil {
		m.lo.Error("error starting read-only transaction", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.activityLog}"), nil)
	}
	defer tx.Rollback()

	// Execute query
	var activityLogs = make([]models.ActivityLog, 0)
	if err := tx.Select(&activityLogs, query, qArgs...); err != nil {
		m.lo.Error("error fetching activity logs", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.activityLog}"), nil)
	}
	return activityLogs, nil
}

// Login records a login event for the given user.
func (al *Manager) Login(userID int, email, ip string) error {
	return al.create(
		models.AgentLogin,
		fmt.Sprintf("%s (#%d) logged in", email, userID),
		userID,
		umodels.UserModel,
		userID,
		ip,
	)
}

// Logout records a logout event for the given user.
func (al *Manager) Logout(userID int, email, ip string) error {
	return al.create(
		models.AgentLogout,
		fmt.Sprintf("%s (#%d) logged out", email, userID),
		userID,
		umodels.UserModel,
		userID,
		ip,
	)
}

// Away records an away event for the given user.
func (al *Manager) Away(actorID int, actorEmail, ip string, targetID int, targetEmail string) error {
	var description string
	if targetID != 0 && targetEmail != "" && (targetID != actorID || targetEmail != actorEmail) {
		description = fmt.Sprintf("%s (#%d) changed %s (#%d) status to away", actorEmail, actorID, targetEmail, targetID)
	} else {
		description = fmt.Sprintf("%s (#%d) is away", actorEmail, actorID)
	}
	return al.create(
		models.AgentAway, /* activity type*/
		description,
		actorID,           /*actor_id*/
		umodels.UserModel, /*target_model_type*/
		actorID,           /*target_model_id*/
		ip,
	)
}

// AwayReassigned records an away and reassigned event for the given user.
func (al *Manager) AwayReassigned(actorID int, actorEmail, ip string, targetID int, targetEmail string) error {
	var description string
	if targetID != 0 && targetEmail != "" && (targetID != actorID || targetEmail != actorEmail) {
		description = fmt.Sprintf("%s (#%d) changed %s (#%d) status to away and reassigning", actorEmail, actorID, targetEmail, targetID)
	} else {
		description = fmt.Sprintf("%s (#%d) is away and reassigning", actorEmail, actorID)
	}
	return al.create(
		models.AgentAwayReassigned, /* activity type*/
		description,
		actorID,           /*actor_id*/
		umodels.UserModel, /*target_model_type*/
		actorID,           /*target_model_id*/
		ip,
	)
}

// Online records an online event for the given user.
func (al *Manager) Online(actorID int, actorEmail, ip string, targetID int, targetEmail string) error {
	var description string
	if targetID != 0 && targetEmail != "" && (targetID != actorID || targetEmail != actorEmail) {
		description = fmt.Sprintf("%s (#%d) changed %s (#%d) status to online", actorEmail, actorID, targetEmail, targetID)
	} else {
		description = fmt.Sprintf("%s (#%d) is online", actorEmail, actorID)
	}
	return al.create(
		models.AgentOnline, /* activity type*/
		description,
		actorID,           /*actor_id*/
		umodels.UserModel, /*target_model_type*/
		actorID,           /*target_model_id*/
		ip,
	)
}

// UserAvailability records a user availability event for the given user.
func (al *Manager) UserAvailability(actorID int, actorEmail, status, ip, targetEmail string, targetID int) error {
	switch status {
	case umodels.Online:
		if err := al.Online(actorID, actorEmail, ip, targetID, targetEmail); err != nil {
			return err
		}
	case umodels.AwayManual:
		if err := al.Away(actorID, actorEmail, ip, targetID, targetEmail); err != nil {
			al.lo.Error("error logging away activity", "error", err)
			return err
		}
	case umodels.AwayAndReassigning:
		if err := al.AwayReassigned(actorID, actorEmail, ip, targetID, targetEmail); err != nil {
			al.lo.Error("error logging away and reassigning activity", "error", err)
			return err
		}
	}
	return nil
}

// create creates a new activity log in DB.
func (m *Manager) create(activityType, activityDescription string, actorID int, targetModelType string, targetModelID int, ip string) error {
	var activityLog models.ActivityLog
	if err := m.q.InsertActivity.Get(&activityLog, activityType, activityDescription, actorID, targetModelType, targetModelID, ip); err != nil {
		m.lo.Error("error inserting activity log", "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorCreating", "name", "{globals.terms.activityLog}"), nil)
	}
	return nil
}

// makeQuery constructs the SQL query for fetching activity logs with filters and pagination.
func (m *Manager) makeQuery(page, pageSize int, order, orderBy, filtersJSON string) (string, []any, error) {
	var (
		baseQuery = m.q.GetAllActivities
		qArgs     []any
	)
	return dbutil.BuildPaginatedQuery(baseQuery, qArgs, dbutil.PaginationOptions{
		Order:    order,
		OrderBy:  orderBy,
		Page:     page,
		PageSize: pageSize,
	}, filtersJSON, dbutil.AllowedFields{
		"activity_logs": {"activity_type", "actor_id", "ip", "created_at"},
	})
}
