// Package role handles role-related operations including creating, updating, fetching, and deleting roles.
package role

import (
	"database/sql"
	"embed"

	amodels "github.com/abhinavxd/libredesk/internal/authz/models"
	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/role/models"
	"github.com/jmoiron/sqlx"
	"github.com/knadh/go-i18n"
	"github.com/lib/pq"
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
	Get    *sqlx.Stmt `query:"get-role"`
	GetAll *sqlx.Stmt `query:"get-all"`
	Delete *sqlx.Stmt `query:"delete-role"`
	Insert *sqlx.Stmt `query:"insert-role"`
	Update *sqlx.Stmt `query:"update-role"`
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
		return roles, envelope.NewError(envelope.GeneralError, u.i18n.Ts("globals.messages.errorFetching", "name", "{globals.entities.role}"), nil)
	}
	return roles, nil
}

// Get retrieves a role by ID.
func (u *Manager) Get(id int) (models.Role, error) {
	var role = models.Role{}
	if err := u.q.Get.Get(&role, id); err != nil {
		if err == sql.ErrNoRows {
			return role, envelope.NewError(envelope.NotFoundError, u.i18n.Ts("globals.messages.notFound", "name", "{globals.entities.role}"), nil)
		}
		u.lo.Error("error fetching role", "error", err)
		return role, envelope.NewError(envelope.GeneralError, u.i18n.Ts("globals.messages.errorFetching", "name", "{globals.entities.role}"), nil)
	}
	return role, nil
}

// Delete deletes a role by ID.
func (u *Manager) Delete(id int) error {
	// Disallow deletion of default roles.
	role, err := u.Get(id)
	if err != nil {
		return err
	}
	for _, r := range models.DefaultRoles {
		if role.Name == r {
			return envelope.NewError(envelope.InputError, u.i18n.Ts("globals.messages.errorDeleting", "name", "default {globals.entities.role}"), nil)
		}
	}
	if _, err := u.q.Delete.Exec(id); err != nil {
		u.lo.Error("error deleting role", "error", err)
		return envelope.NewError(envelope.GeneralError, u.i18n.Ts("globals.messages.errorDeleting", "name", "{globals.entities.role}"), nil)
	}
	return nil
}

// Create creates a new role.
func (u *Manager) Create(r models.Role) error {
	if err := u.validatePermissions(r.Permissions); err != nil {
		return err
	}
	if _, err := u.q.Insert.Exec(r.Name, r.Description, pq.Array(r.Permissions)); err != nil {
		if dbutil.IsUniqueViolationError(err) {
			return envelope.NewError(envelope.InputError, u.i18n.Ts("globals.messages.errorAlreadyExists", "name", "{globals.entities.role}"), nil)
		}
		u.lo.Error("error inserting role", "error", err)
		return envelope.NewError(envelope.GeneralError, u.i18n.Ts("globals.messages.errorCreating", "name", "{globals.entities.role}"), nil)
	}
	return nil
}

// Update updates an existing role.
func (u *Manager) Update(id int, r models.Role) error {
	if err := u.validatePermissions(r.Permissions); err != nil {
		return err
	}

	// Disallow updating `Admin` role, as the main System login requires it.
	role, err := u.Get(id)
	if err != nil {
		return err
	}
	if role.Name == models.RoleAdmin {
		return envelope.NewError(envelope.InputError, u.i18n.Ts("globals.messages.errorUpdating", "name", "Admin {globals.entities.role}"), nil)
	}

	if _, err := u.q.Update.Exec(id, r.Name, r.Description, pq.Array(r.Permissions)); err != nil {
		u.lo.Error("error updating role", "error", err)
		return envelope.NewError(envelope.GeneralError, u.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.entities.role}"), nil)
	}
	return nil
}

// validatePermissions returns true if all given permissions are valid
func (u *Manager) validatePermissions(permissions []string) error {
	if len(permissions) == 0 {
		return envelope.NewError(envelope.InputError, u.i18n.Ts("globals.messages.empty", "name", u.i18n.P("globals.entities.permission")), nil)
	}
	for _, perm := range permissions {
		if !amodels.IsValidPermission(perm) {
			u.lo.Error("error unknown permission", "permission", perm)
			return envelope.NewError(envelope.InputError, u.i18n.Ts("role.invalidPermission", "name", perm), nil)
		}
	}
	return nil
}
