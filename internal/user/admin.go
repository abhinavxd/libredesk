package user

import (
	"database/sql"
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/jakedolan443/fernmail/internal/dbutil"
	"github.com/jakedolan443/fernmail/internal/envelope"
	rmodels "github.com/jakedolan443/fernmail/internal/role/models"
	"github.com/jakedolan443/fernmail/internal/stringutil"
	"github.com/jakedolan443/fernmail/internal/user/models"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/volatiletech/null/v9"
)

// The Users settings screen gives each person exactly one role. Admins see
// every address; everyone else is granted addresses directly or via a team.

// ManagedUser is one row of the Users settings screen.
type ManagedUser struct {
	ID           int            `db:"id" json:"id"`
	FirstName    string         `db:"first_name" json:"first_name"`
	LastName     string         `db:"last_name" json:"last_name"`
	Email        string         `db:"email" json:"email"`
	Enabled      bool           `db:"enabled" json:"enabled"`
	Roles        pq.StringArray `db:"roles" json:"roles"`
	AddressIDs   pq.Int64Array  `db:"address_ids" json:"address_ids"`
	LastActiveAt null.Time      `db:"last_active_at" json:"last_active_at"`
	CreatedAt    time.Time      `db:"created_at" json:"created_at"`
}

// AccessUpdate replaces a user's role, direct address grants and enabled state together.
type AccessUpdate struct {
	Role       string `json:"role"`
	AddressIDs []int  `json:"address_ids"`
	Enabled    bool   `json:"enabled"`
}

const managedUsersQuery = `SELECT u.id, u.first_name, u.last_name, COALESCE(u.email, '') AS email, u.enabled, u.last_active_at, u.created_at,
	COALESCE((SELECT array_agg(r.name ORDER BY r.name) FROM user_roles ur JOIN roles r ON r.id = ur.role_id WHERE ur.user_id = u.id), '{}') AS roles,
	COALESCE((SELECT array_agg(eau.address_id ORDER BY eau.address_id) FROM email_address_users eau WHERE eau.user_id = u.id), '{}') AS address_ids
FROM users u
WHERE u.type = 'agent' AND u.deleted_at IS NULL AND u.email IS DISTINCT FROM 'System'`

// adminLock serialises role, enable and delete changes so two Admins cannot
// each remove the other's access in parallel and leave nobody in charge.
const adminLock = `SELECT pg_advisory_xact_lock(hashtextextended('fernmail:user-admin', 0))`

func adminInputError(msg string) error {
	return envelope.NewError(envelope.InputError, msg, nil)
}

// GetManagedUsers lists every person who can sign in, excluding the built-in System user.
func (u *Manager) GetManagedUsers() ([]ManagedUser, error) {
	users := []ManagedUser{}
	if err := u.db.Select(&users, managedUsersQuery+` ORDER BY lower(u.first_name), lower(u.last_name), u.id`); err != nil {
		u.lo.Error("error fetching managed users", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, u.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return users, nil
}

// GetManagedUser returns one row of the Users settings screen.
func (u *Manager) GetManagedUser(id int) (ManagedUser, error) {
	var user ManagedUser
	err := u.db.Get(&user, managedUsersQuery+` AND u.id = $1`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return user, envelope.NewError(envelope.NotFoundError, u.i18n.Ts("globals.messages.notFound", "name", u.i18n.T("globals.terms.agent")), nil)
	}
	if err != nil {
		u.lo.Error("error fetching managed user", "id", id, "error", err)
		return user, envelope.NewError(envelope.GeneralError, u.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return user, nil
}

// GetRoleNames returns every role, the built-in ones first.
func (u *Manager) GetRoleNames() ([]string, error) {
	names := []string{}
	err := u.db.Select(&names, `SELECT name FROM roles ORDER BY CASE name WHEN $1 THEN 0 WHEN $2 THEN 1 WHEN $3 THEN 2 ELSE 3 END, lower(name)`,
		rmodels.RoleAdmin, rmodels.RoleAgent, rmodels.RoleContributor)
	if err != nil {
		u.lo.Error("error fetching roles", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, u.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return names, nil
}

// CreateManagedUser adds a person with one role and their direct address grants.
func (u *Manager) CreateManagedUser(firstName, lastName, email, role string, addressIDs []int) (ManagedUser, error) {
	firstName, lastName = strings.TrimSpace(firstName), strings.TrimSpace(lastName)
	email = strings.ToLower(strings.TrimSpace(email))
	if firstName == "" || len(firstName) > 140 || len(lastName) > 140 {
		return ManagedUser{}, adminInputError("Enter a first name of up to 140 characters.")
	}
	if !stringutil.ValidEmail(email) || strings.EqualFold(email, models.SystemUserEmail) {
		return ManagedUser{}, envelope.NewError(envelope.InputError, u.i18n.T("validation.invalidEmail"), nil)
	}
	password, err := u.generatePassword()
	if err != nil {
		return ManagedUser{}, envelope.NewError(envelope.GeneralError, u.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	tx, err := u.db.Beginx()
	if err != nil {
		return ManagedUser{}, envelope.NewError(envelope.GeneralError, u.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	defer tx.Rollback()
	if err := validateManagedAccess(tx, role, addressIDs); err != nil {
		return ManagedUser{}, u.adminWriteError(err)
	}
	var id int
	if err := tx.Stmtx(u.q.InsertAgent).QueryRow(email, firstName, lastName, password, null.String{}, pq.Array([]string{role})).Scan(&id); err != nil {
		if dbutil.IsUniqueViolationError(err) {
			return ManagedUser{}, envelope.NewError(envelope.InputError, u.i18n.T("user.sameEmailAlreadyExists"), nil)
		}
		u.lo.Error("error creating managed user", "error", err)
		return ManagedUser{}, envelope.NewError(envelope.GeneralError, u.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	if err := replaceUserAddresses(tx, id, addressIDs); err != nil {
		u.lo.Error("error granting addresses to new user", "error", err)
		return ManagedUser{}, envelope.NewError(envelope.GeneralError, u.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	if err := tx.Commit(); err != nil {
		return ManagedUser{}, envelope.NewError(envelope.GeneralError, u.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return u.GetManagedUser(id)
}

// UpdateAccess replaces one user's role, direct address grants and enabled
// state. actorID is the Admin making the change: nobody can change their own
// role or disable themselves, and the last enabled Admin cannot be removed.
func (u *Manager) UpdateAccess(actorID, id int, in AccessUpdate) error {
	tx, err := u.db.Beginx()
	if err != nil {
		return envelope.NewError(envelope.GeneralError, u.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	defer tx.Rollback()
	if _, err := tx.Exec(adminLock); err != nil {
		return envelope.NewError(envelope.GeneralError, u.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	target, err := u.lockManagedTarget(tx, id)
	if err != nil {
		return err
	}
	if err := validateManagedAccess(tx, in.Role, in.AddressIDs); err != nil {
		return u.adminWriteError(err)
	}
	roleChanges := len(target.Roles) != 1 || target.Roles[0] != in.Role
	if id == actorID && (!slices.Contains(target.Roles, in.Role) || !in.Enabled) {
		return adminInputError("You can't change your own role or disable your own account.")
	}
	losesAdmin := target.Enabled && target.isAdmin() && (in.Role != rmodels.RoleAdmin || !in.Enabled)
	if losesAdmin {
		if err := u.requireAnotherAdmin(tx, id); err != nil {
			return err
		}
	}
	if roleChanges {
		if _, err := tx.Exec(`DELETE FROM user_roles WHERE user_id = $1`, id); err != nil {
			return u.adminWriteError(err)
		}
		if _, err := tx.Exec(`INSERT INTO user_roles (user_id, role_id) SELECT $1, id FROM roles WHERE name = $2`, id, in.Role); err != nil {
			return u.adminWriteError(err)
		}
	}
	if _, err := tx.Exec(`UPDATE users SET enabled = $2, updated_at = NOW() WHERE id = $1`, id, in.Enabled); err != nil {
		return u.adminWriteError(err)
	}
	if err := replaceUserAddresses(tx, id, in.AddressIDs); err != nil {
		return u.adminWriteError(err)
	}
	if err := tx.Commit(); err != nil {
		return u.adminWriteError(err)
	}
	u.InvalidateAgentCache(id)
	return nil
}

// CheckAccessChange applies UpdateAccess's safeguards to the legacy agent
// API, which can set several roles at once.
func (u *Manager) CheckAccessChange(actorID, id int, roles []string, enabled bool) error {
	tx, err := u.db.Beginx()
	if err != nil {
		return envelope.NewError(envelope.GeneralError, u.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	defer tx.Rollback()
	if _, err := tx.Exec(adminLock); err != nil {
		return envelope.NewError(envelope.GeneralError, u.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	target, err := u.lockManagedTarget(tx, id)
	if err != nil {
		return err
	}
	sameRoles := len(roles) == len(target.Roles)
	for _, role := range roles {
		sameRoles = sameRoles && slices.Contains(target.Roles, role)
	}
	if id == actorID && (!sameRoles || !enabled) {
		return adminInputError("You can't change your own role or disable your own account.")
	}
	if target.Enabled && target.isAdmin() && (!slices.Contains(roles, rmodels.RoleAdmin) || !enabled) {
		return u.requireAnotherAdmin(tx, id)
	}
	return nil
}

// DeleteManagedUser soft-deletes a user under the same safeguards as UpdateAccess.
func (u *Manager) DeleteManagedUser(actorID, id int) error {
	if id == actorID {
		return envelope.NewError(envelope.InputError, u.i18n.T("user.userCannotDeleteSelf"), nil)
	}
	tx, err := u.db.Beginx()
	if err != nil {
		return envelope.NewError(envelope.GeneralError, u.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	defer tx.Rollback()
	if _, err := tx.Exec(adminLock); err != nil {
		return envelope.NewError(envelope.GeneralError, u.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	target, err := u.lockManagedTarget(tx, id)
	if err != nil {
		return err
	}
	if target.Enabled && target.isAdmin() {
		if err := u.requireAnotherAdmin(tx, id); err != nil {
			return err
		}
	}
	var deleted int
	if err := tx.Stmtx(u.q.SoftDeleteAgent).Get(&deleted, id); err != nil {
		return u.adminWriteError(err)
	}
	if _, err := tx.Exec(`DELETE FROM email_address_users WHERE user_id = $1`, id); err != nil {
		return u.adminWriteError(err)
	}
	if err := tx.Commit(); err != nil {
		return u.adminWriteError(err)
	}
	u.InvalidateAgentCache(id)
	return nil
}

type managedTarget struct {
	Email   null.String    `db:"email"`
	Enabled bool           `db:"enabled"`
	Roles   pq.StringArray `db:"roles"`
}

func (t managedTarget) isAdmin() bool {
	return slices.Contains(t.Roles, rmodels.RoleAdmin)
}

func (u *Manager) lockManagedTarget(tx *sqlx.Tx, id int) (managedTarget, error) {
	var target managedTarget
	err := tx.Get(&target, `SELECT email, enabled FROM users WHERE id = $1 AND type = 'agent' AND deleted_at IS NULL FOR UPDATE`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return target, envelope.NewError(envelope.NotFoundError, u.i18n.Ts("globals.messages.notFound", "name", u.i18n.T("globals.terms.agent")), nil)
	}
	if err != nil {
		return target, u.adminWriteError(err)
	}
	if target.Email.String == models.SystemUserEmail {
		return target, adminInputError("The built-in System user can't be changed here.")
	}
	if err := tx.Select(&target.Roles, `SELECT r.name FROM user_roles ur JOIN roles r ON r.id = ur.role_id WHERE ur.user_id = $1 ORDER BY r.name`, id); err != nil {
		return target, u.adminWriteError(err)
	}
	return target, nil
}

// requireAnotherAdmin keeps at least one enabled person (not the System user) in the Admin role.
func (u *Manager) requireAnotherAdmin(tx *sqlx.Tx, excludingID int) error {
	var others int
	if err := tx.Get(&others, `SELECT count(DISTINCT u.id) FROM users u
		JOIN user_roles ur ON ur.user_id = u.id JOIN roles r ON r.id = ur.role_id
		WHERE r.name = $2 AND u.id <> $1 AND u.type = 'agent' AND u.enabled AND u.deleted_at IS NULL
		AND u.email IS DISTINCT FROM $3`, excludingID, rmodels.RoleAdmin, models.SystemUserEmail); err != nil {
		return u.adminWriteError(err)
	}
	if others == 0 {
		return adminInputError("At least one person must remain an enabled Admin. Make someone else an Admin first.")
	}
	return nil
}

func validateManagedAccess(tx *sqlx.Tx, role string, addressIDs []int) error {
	var roleExists bool
	if err := tx.Get(&roleExists, `SELECT EXISTS(SELECT 1 FROM roles WHERE name = $1)`, role); err != nil {
		return err
	}
	if !roleExists {
		return adminInputError("Choose a valid role.")
	}
	ids := uniqueIDs(addressIDs)
	if len(ids) != len(addressIDs) {
		return adminInputError("Address list contains an invalid or repeated address.")
	}
	if len(ids) == 0 {
		return nil
	}
	var found int
	if err := tx.Get(&found, `SELECT count(*) FROM email_addresses WHERE id = ANY($1::int[])`, pq.Array(ids)); err != nil {
		return err
	}
	if found != len(ids) {
		return adminInputError("One of the selected addresses no longer exists. Reload and try again.")
	}
	return nil
}

func replaceUserAddresses(tx *sqlx.Tx, userID int, addressIDs []int) error {
	if addressIDs == nil {
		// A nil array would be SQL NULL, and NOT (x = ANY(NULL)) deletes nothing.
		addressIDs = []int{}
	}
	if _, err := tx.Exec(`DELETE FROM email_address_users WHERE user_id = $1::bigint AND NOT (address_id = ANY($2::int[]))`, userID, pq.Array(addressIDs)); err != nil {
		return err
	}
	if len(addressIDs) == 0 {
		return nil
	}
	_, err := tx.Exec(`INSERT INTO email_address_users (address_id, user_id) SELECT unnest($2::int[]), $1::bigint ON CONFLICT DO NOTHING`, userID, pq.Array(addressIDs))
	return err
}

func uniqueIDs(ids []int) []int {
	seen := make(map[int]bool, len(ids))
	out := make([]int, 0, len(ids))
	for _, id := range ids {
		if id <= 0 || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	return out
}

func (u *Manager) adminWriteError(err error) error {
	var envErr envelope.Error
	if errors.As(err, &envErr) {
		return err
	}
	u.lo.Error("error updating user access", "error", err)
	return envelope.NewError(envelope.GeneralError, u.i18n.T("globals.messages.somethingWentWrong"), nil)
}
