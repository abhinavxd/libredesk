// Package activity manages activity logs for all users.
package activitylog

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"strings"

	"github.com/abhinavxd/libredesk/internal/activity_log/models"
	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/abhinavxd/libredesk/internal/envelope"
	umodels "github.com/abhinavxd/libredesk/internal/user/models"
	"github.com/jmoiron/sqlx"
	"github.com/leonelquinteros/gotext"
	"github.com/zerodha/logf"
)

var (
	//go:embed queries.sql
	efs embed.FS
)

type Manager struct {
	q    queries
	lo   *logf.Logger
	i18n *gotext.Locale
	db   *sqlx.DB
}

// Opts contains options for initializing the Manager.
type Opts struct {
	DB   *sqlx.DB
	Lo   *logf.Logger
	I18n *gotext.Locale
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
func (m *Manager) GetAll(order, orderBy, filtersJSON string, page, pageSize int, location string) ([]models.ActivityLog, error) {
	query, qArgs, err := m.makeQuery(page, pageSize, order, orderBy, filtersJSON, location)
	if err != nil {
		m.lo.Error("error creating activity log list query", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.Get("Something went wrong"), nil)
	}

	// Start a read-only txn.
	tx, err := m.db.BeginTxx(context.Background(), &sql.TxOptions{
		ReadOnly: true,
	})
	if err != nil {
		m.lo.Error("error starting read-only transaction", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.Get("Something went wrong"), nil)
	}
	defer tx.Rollback()

	// Execute query
	var activityLogs = make([]models.ActivityLog, 0)
	if err := tx.Select(&activityLogs, query, qArgs...); err != nil {
		m.lo.Error("error fetching activity logs", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.Get("Something went wrong"), nil)
	}
	return activityLogs, nil
}

// Login records a login event for the given user.
func (al *Manager) Login(userID int, email, ip string) error {
	description := al.i18n.Get("%[1]s (%[2]s) logged in", email, fmt.Sprintf("#%d", userID))
	return al.create(
		models.AgentLogin,
		description,
		userID,
		umodels.UserModel,
		userID,
		ip,
	)
}

// Logout records a logout event for the given user.
func (al *Manager) Logout(userID int, email, ip string) error {
	description := al.i18n.Get("%[1]s (%[2]s) logged out", email, fmt.Sprintf("#%d", userID))
	return al.create(
		models.AgentLogout,
		description,
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
		description = al.i18n.Get("%[1]s (%[2]s) changed %[3]s (%[4]s) status to away",
			actorEmail,
			fmt.Sprintf("#%d", actorID),
			targetEmail,
			fmt.Sprintf("#%d", targetID))
	} else {
		description = al.i18n.Get("%[1]s (%[2]s) is away", actorEmail, fmt.Sprintf("#%d", actorID))
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
		description = al.i18n.Get("%[1]s (%[2]s) changed %[3]s (%[4]s) status to away and reassigning",
			actorEmail,
			fmt.Sprintf("#%d", actorID),
			targetEmail,
			fmt.Sprintf("#%d", targetID))
	} else {
		description = al.i18n.Get("%[1]s (%[2]s) is away and reassigning", actorEmail, fmt.Sprintf("#%d", actorID))
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
		description = al.i18n.Get("%[1]s (%[2]s) changed %[3]s (%[4]s) status to online",
			actorEmail,
			fmt.Sprintf("#%d", actorID),
			targetEmail,
			fmt.Sprintf("#%d", targetID))
	} else {
		description = al.i18n.Get("%[1]s (%[2]s) is online", actorEmail, fmt.Sprintf("#%d", actorID))
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
			return err
		}
	case umodels.AwayAndReassigning:
		if err := al.AwayReassigned(actorID, actorEmail, ip, targetID, targetEmail); err != nil {
			return err
		}
	}
	return nil
}

// PasswordSet records a password set event.
func (al *Manager) PasswordSet(actorID int, actorEmail, ip string, targetID int, targetEmail string) error {
	description := al.i18n.Get("%[1]s (%[2]s) set password for %[3]s (%[4]s)",
		actorEmail,
		fmt.Sprintf("#%d", actorID),
		targetEmail,
		fmt.Sprintf("#%d", targetID))
	return al.create(
		models.AgentPasswordSet,
		description,
		actorID,
		umodels.UserModel,
		targetID,
		ip,
	)
}

// RolePermissionsChanged records a role permissions change event.
func (al *Manager) RolePermissionsChanged(actorID int, actorEmail, ip string, roleID int, roleName string, added, removed []string) error {
	var description string
	if len(removed) > 0 && len(added) > 0 {
		description = al.i18n.Get("%[1]s (%[2]s) removed permission(s) %[3]s and added permission(s) %[4]s to role %[5]s (%[6]s)",
			actorEmail,
			fmt.Sprintf("#%d", actorID),
			strings.Join(removed, ", "),
			strings.Join(added, ", "),
			roleName,
			fmt.Sprintf("#%d", roleID))
	} else if len(removed) > 0 {
		description = al.i18n.Get("%[1]s (%[2]s) removed permission(s) %[3]s from role %[4]s (%[5]s)",
			actorEmail,
			fmt.Sprintf("#%d", actorID),
			strings.Join(removed, ", "),
			roleName,
			fmt.Sprintf("#%d", roleID))
	} else if len(added) > 0 {
		description = al.i18n.Get("%[1]s (%[2]s) added permission(s) %[3]s to role %[4]s (%[5]s)",
			actorEmail,
			fmt.Sprintf("#%d", actorID),
			strings.Join(added, ", "),
			roleName,
			fmt.Sprintf("#%d", roleID))
	} else {
		return nil // No changes
	}
	return al.create(
		models.AgentRolePermissionsChanged,
		description,
		actorID,
		"role",
		roleID,
		ip,
	)
}

// ContactDeleted records permanent deletion of a contact and their data.
func (al *Manager) ContactDeleted(actorID int, actorEmail, ip string, contactID int, contactEmail string) error {
	description := al.i18n.Get("%[1]s (%[2]s) deleted contact %[3]s (%[4]s)",
		actorEmail,
		fmt.Sprintf("#%d", actorID),
		contactEmail,
		fmt.Sprintf("#%d", contactID))
	return al.create(
		models.ContactDeleted,
		description,
		actorID,
		umodels.UserModel,
		contactID,
		ip,
	)
}

// ContactDataExported records an export of a contact's stored data.
func (al *Manager) ContactDataExported(actorID int, actorEmail, ip string, contactID int, contactEmail string) error {
	description := al.i18n.Get("%[1]s (%[2]s) exported data of contact %[3]s (%[4]s)",
		actorEmail,
		fmt.Sprintf("#%d", actorID),
		contactEmail,
		fmt.Sprintf("#%d", contactID))
	return al.create(
		models.ContactDataExported,
		description,
		actorID,
		umodels.UserModel,
		contactID,
		ip,
	)
}

// create creates a new activity log in DB.
func (m *Manager) create(activityType, activityDescription string, actorID int, targetModelType string, targetModelID int, ip string) error {
	if _, err := m.q.InsertActivity.Exec(activityType, activityDescription, actorID, targetModelType, targetModelID, ip); err != nil {
		m.lo.Error("error inserting activity log", "actor_id", actorID, "target_model", targetModelType, "target_id", targetModelID, "activity_type", activityType, "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.Get("Something went wrong"), nil)
	}
	return nil
}

// makeQuery constructs the SQL query for fetching activity logs with filters and pagination.
func (m *Manager) makeQuery(page, pageSize int, order, orderBy, filtersJSON, location string) (string, []any, error) {
	var (
		baseQuery = m.q.GetAllActivities
		qArgs     []any
	)
	return dbutil.BuildPaginatedQuery(baseQuery, qArgs, dbutil.PaginationOptions{
		Order:    order,
		OrderBy:  orderBy,
		Page:     page,
		PageSize: pageSize,
		Location: location,
	}, filtersJSON, dbutil.AllowedFields{
		"activity_logs": {"activity_type", "actor_id", "ip", "created_at"},
	}, nil)
}
