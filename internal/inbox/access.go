package inbox

import (
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// Mailbox is safe to expose to readers; it contains no connection settings or secrets.
type Mailbox struct {
	ID      int    `db:"id" json:"id"`
	Name    string `db:"name" json:"name"`
	From    string `db:"from" json:"from"`
	Channel string `db:"channel" json:"channel"`
	Enabled bool   `db:"enabled" json:"enabled"`
}

type AccessChoice struct {
	ID   int    `db:"id" json:"id"`
	Name string `db:"name" json:"name"`
}

type InboxAccess struct {
	Restricted bool           `json:"restricted"`
	UserIDs    []int          `json:"user_ids"`
	RoleIDs    []int          `json:"role_ids"`
	Users      []AccessChoice `json:"users,omitempty"`
	Roles      []AccessChoice `json:"roles,omitempty"`
}

func (m *Manager) CanAccess(userID, inboxID int) (bool, error) {
	var allowed bool
	err := m.db.Get(&allowed, `SELECT can_access_inbox($1, $2)`, inboxID, userID)
	return allowed, err
}

func (m *Manager) GetMailboxes(userID int) ([]Mailbox, error) {
	result := []Mailbox{}
	err := m.db.Select(&result, `SELECT id, name, COALESCE("from", '') AS "from", channel, enabled
        FROM inboxes WHERE can_access_inbox(id, $1) ORDER BY lower(name), id`, userID)
	return result, err
}

func (m *Manager) GetAccess(inboxID int) (InboxAccess, error) {
	access := InboxAccess{UserIDs: []int{}, RoleIDs: []int{}, Users: []AccessChoice{}, Roles: []AccessChoice{}}
	if _, err := m.GetDBRecord(inboxID); err != nil {
		return access, err
	}
	if err := m.db.Get(&access.Restricted, `SELECT COALESCE((SELECT restricted FROM inbox_access WHERE inbox_id=$1), false)`, inboxID); err != nil {
		return access, err
	}
	if err := m.db.Select(&access.UserIDs, `SELECT iu.user_id FROM inbox_users iu JOIN users u ON u.id=iu.user_id WHERE iu.inbox_id=$1 AND u.deleted_at IS NULL AND u.type='agent' ORDER BY iu.user_id`, inboxID); err != nil {
		return access, err
	}
	if err := m.db.Select(&access.RoleIDs, `SELECT role_id FROM inbox_roles WHERE inbox_id=$1 ORDER BY role_id`, inboxID); err != nil {
		return access, err
	}
	if err := m.db.Select(&access.Users, `SELECT id, concat_ws(' ', first_name, NULLIF(last_name, '')) || COALESCE(' (' || email || ')', '') AS name
        FROM users WHERE type='agent' AND deleted_at IS NULL AND email IS DISTINCT FROM 'System' ORDER BY lower(first_name), id`); err != nil {
		return access, err
	}
	err := m.db.Select(&access.Roles, `SELECT id, name FROM roles WHERE name != 'Admin' ORDER BY lower(name), id`)
	return access, err
}

// UpdateAccess validates every principal and replaces the policy atomically.
func (m *Manager) UpdateAccess(inboxID int, access InboxAccess) error {
	tx, err := m.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var id int
	if err := tx.Get(&id, `SELECT id FROM inboxes WHERE id=$1 AND deleted_at IS NULL AND channel='email' FOR UPDATE`, inboxID); err != nil {
		return err
	}
	if err := validatePrincipals(tx, access.UserIDs, `SELECT count(DISTINCT id) FROM users WHERE id=ANY($1) AND type='agent' AND deleted_at IS NULL`); err != nil {
		return err
	}
	if err := validatePrincipals(tx, access.RoleIDs, `SELECT count(DISTINCT id) FROM roles WHERE id=ANY($1)`); err != nil {
		return err
	}
	if _, err := tx.Exec(`INSERT INTO inbox_access(inbox_id, restricted) VALUES ($1,$2)
        ON CONFLICT(inbox_id) DO UPDATE SET restricted=EXCLUDED.restricted`, inboxID, access.Restricted); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM inbox_users WHERE inbox_id=$1`, inboxID); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM inbox_roles WHERE inbox_id=$1`, inboxID); err != nil {
		return err
	}
	if _, err := tx.Exec(`INSERT INTO inbox_users SELECT $1, unnest($2::bigint[]) ON CONFLICT DO NOTHING`, inboxID, pq.Array(access.UserIDs)); err != nil {
		return err
	}
	if _, err := tx.Exec(`INSERT INTO inbox_roles SELECT $1, unnest($2::int[]) ON CONFLICT DO NOTHING`, inboxID, pq.Array(access.RoleIDs)); err != nil {
		return err
	}
	return tx.Commit()
}

func validatePrincipals(tx *sqlx.Tx, ids []int, query string) error {
	unique := map[int]bool{}
	for _, id := range ids {
		unique[id] = true
	}
	var count int
	if err := tx.Get(&count, query, pq.Array(ids)); err != nil {
		return err
	}
	if count != len(unique) {
		return fmt.Errorf("invalid inbox access user or role")
	}
	return nil
}
