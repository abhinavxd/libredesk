// Package address owns Fernmail's user-facing email endpoints. Transport
// inboxes remain an implementation detail; every conversation belongs to one
// address so routing, permissions, unread counts, and reply identity agree.
package address

import (
	"database/sql"
	"errors"
	"fmt"
	"net/mail"
	"slices"
	"strings"
	"time"

	"github.com/jakedolan443/fernmail/internal/stringutil"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

const (
	KindMailbox = "mailbox"
	KindAlias   = "alias"
)

var ErrNotFound = errors.New("address not found")

// Address is a deliverable email endpoint. Mailbox entries own a transport
// inbox; aliases share that inbox's IMAP/SMTP connection.
type Address struct {
	ID          int       `db:"id" json:"id"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time `db:"updated_at" json:"updated_at"`
	InboxID     int       `db:"inbox_id" json:"inbox_id"`
	Address     string    `db:"address" json:"address"`
	DisplayName string    `db:"display_name" json:"display_name"`
	Kind        string    `db:"kind" json:"kind"`
	Enabled     bool      `db:"enabled" json:"enabled"`
	UserIDs     []int     `json:"user_ids,omitempty"`
	TeamIDs     []int     `json:"team_ids,omitempty"`
}

type Principal struct {
	ID   int    `db:"id" json:"id"`
	Name string `db:"name" json:"name"`
}

// Access contains the available people and teams as well as the selected
// principals, so the settings page can edit one address atomically. There is
// no open-to-all address: Admins see everything, everyone else needs a grant.
type Access struct {
	UserIDs []int       `json:"user_ids"`
	TeamIDs []int       `json:"team_ids"`
	Users   []Principal `json:"users,omitempty"`
	Teams   []Principal `json:"teams,omitempty"`
}

type Manager struct {
	db *sqlx.DB
}

func New(db *sqlx.DB) (*Manager, error) {
	if db == nil {
		return nil, errors.New("address database is required")
	}
	return &Manager{db: db}, nil
}

func normalize(raw string) (string, error) {
	raw = strings.TrimSpace(strings.ToLower(raw))
	if parsed, err := mail.ParseAddress(raw); err == nil {
		raw = strings.ToLower(strings.TrimSpace(parsed.Address))
	}
	if !stringutil.ValidEmail(raw) {
		return "", fmt.Errorf("invalid email address")
	}
	return raw, nil
}

func validateInput(in Address) (Address, error) {
	address, err := normalize(in.Address)
	if err != nil {
		return Address{}, err
	}
	in.Address = address
	in.DisplayName = strings.TrimSpace(in.DisplayName)
	if len(in.DisplayName) > 140 {
		return Address{}, fmt.Errorf("display name is too long")
	}
	if in.Kind != KindMailbox && in.Kind != KindAlias {
		return Address{}, fmt.Errorf("invalid address kind")
	}
	if in.InboxID < 1 {
		return Address{}, fmt.Errorf("transport inbox is required")
	}
	// A nil user list leaves per-user grants untouched: the Users settings
	// screen owns them, the address form only edits team grants.
	if in.UserIDs != nil {
		in.UserIDs = uniquePositive(in.UserIDs)
	}
	in.TeamIDs = uniquePositive(in.TeamIDs)
	return in, nil
}

func uniquePositive(ids []int) []int {
	result := make([]int, 0, len(ids))
	for _, id := range ids {
		if id > 0 && !slices.Contains(result, id) {
			result = append(result, id)
		}
	}
	return result
}

func (m *Manager) CanAccess(userID, addressID int) (bool, error) {
	var allowed bool
	err := m.db.Get(&allowed, `SELECT can_access_email_address($1, $2)`, addressID, userID)
	return allowed, err
}

func (m *Manager) Get(id int) (Address, error) {
	var out Address
	err := m.db.Get(&out, `SELECT id, created_at, updated_at, inbox_id, address, display_name, kind, enabled
		FROM email_addresses WHERE id=$1`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return out, ErrNotFound
	}
	return out, err
}

func (m *Manager) GetForInbox(inboxID int) ([]Address, error) {
	items := []Address{}
	err := m.db.Select(&items, `SELECT id, created_at, updated_at, inbox_id, address, display_name, kind, enabled
		FROM email_addresses WHERE inbox_id=$1 ORDER BY CASE WHEN kind='mailbox' THEN 0 ELSE 1 END, lower(address), id`, inboxID)
	return items, err
}

func (m *Manager) GetAccessible(userID int) ([]Address, error) {
	items := []Address{}
	err := m.db.Select(&items, `SELECT id, created_at, updated_at, inbox_id, address, display_name, kind, enabled
		FROM email_addresses WHERE can_access_email_address(id, $1)
		ORDER BY CASE WHEN kind='mailbox' THEN 0 ELSE 1 END, lower(address), id`, userID)
	return items, err
}

func (m *Manager) GetAll() ([]Address, error) {
	items := []Address{}
	if err := m.db.Select(&items, `SELECT id, created_at, updated_at, inbox_id, address, display_name, kind, enabled
		FROM email_addresses ORDER BY CASE WHEN kind='mailbox' THEN 0 ELSE 1 END, lower(address), id`); err != nil {
		return nil, err
	}
	for i := range items {
		access, err := m.GetAccess(items[i].ID)
		if err != nil {
			return nil, err
		}
		items[i].UserIDs = access.UserIDs
		items[i].TeamIDs = access.TeamIDs
	}
	return items, nil
}

// GetAllCompact lists every address without its access grants.
func (m *Manager) GetAllCompact() ([]Address, error) {
	items := []Address{}
	err := m.db.Select(&items, `SELECT id, created_at, updated_at, inbox_id, address, display_name, kind, enabled
		FROM email_addresses ORDER BY CASE WHEN kind='mailbox' THEN 0 ELSE 1 END, lower(address), id`)
	return items, err
}

// GetPrincipals returns the selectable access principals for a new address.
// Existing addresses use GetAccess so the response includes their selections.
func (m *Manager) GetPrincipals() (Access, error) {
	access := Access{Users: []Principal{}, Teams: []Principal{}, UserIDs: []int{}, TeamIDs: []int{}}
	if err := m.db.Select(&access.Users, `SELECT id, concat_ws(' ', first_name, NULLIF(last_name, '')) || COALESCE(' (' || email || ')', '') AS name
		FROM users WHERE type='agent' AND deleted_at IS NULL AND email IS DISTINCT FROM 'System'
		ORDER BY lower(first_name), id`); err != nil {
		return access, err
	}
	if err := m.db.Select(&access.Teams, `SELECT id, name FROM teams ORDER BY lower(name), id`); err != nil {
		return access, err
	}
	return access, nil
}

func (m *Manager) GetAccess(addressID int) (Access, error) {
	access := Access{UserIDs: []int{}, TeamIDs: []int{}, Users: []Principal{}, Teams: []Principal{}}
	var exists bool
	if err := m.db.Get(&exists, `SELECT EXISTS(SELECT 1 FROM email_addresses WHERE id=$1)`, addressID); err != nil {
		return access, err
	} else if !exists {
		return access, ErrNotFound
	}
	if err := m.db.Select(&access.UserIDs, `SELECT eau.user_id FROM email_address_users eau
		JOIN users u ON u.id=eau.user_id
		WHERE eau.address_id=$1 AND u.type='agent' AND u.deleted_at IS NULL ORDER BY eau.user_id`, addressID); err != nil {
		return access, err
	}
	if err := m.db.Select(&access.TeamIDs, `SELECT team_id FROM email_address_teams WHERE address_id=$1 ORDER BY team_id`, addressID); err != nil {
		return access, err
	}
	principals, err := m.GetPrincipals()
	if err != nil {
		return access, err
	}
	access.Users, access.Teams = principals.Users, principals.Teams
	return access, nil
}

func (m *Manager) Create(in Address) (Address, error) {
	in, err := validateInput(in)
	if err != nil {
		return Address{}, err
	}
	if in.Kind != KindAlias {
		return Address{}, fmt.Errorf("mailbox addresses are created with their transport inbox")
	}
	tx, err := m.db.Beginx()
	if err != nil {
		return Address{}, err
	}
	defer tx.Rollback()
	if err := validateTransport(tx, in.InboxID); err != nil {
		return Address{}, err
	}
	var out Address
	err = tx.Get(&out, `INSERT INTO email_addresses(inbox_id, address, display_name, kind, enabled)
		VALUES($1,$2,$3,$4,$5)
		RETURNING id, created_at, updated_at, inbox_id, address, display_name, kind, enabled`,
		in.InboxID, in.Address, in.DisplayName, in.Kind, in.Enabled)
	if err != nil {
		return Address{}, err
	}
	if err := replaceAccess(tx, out.ID, in.UserIDs, in.TeamIDs); err != nil {
		return Address{}, err
	}
	if err := tx.Commit(); err != nil {
		return Address{}, err
	}
	out.UserIDs, out.TeamIDs = in.UserIDs, in.TeamIDs
	return out, nil
}

func (m *Manager) Update(id int, in Address) (Address, error) {
	in, err := validateInput(in)
	if err != nil {
		return Address{}, err
	}
	tx, err := m.db.Beginx()
	if err != nil {
		return Address{}, err
	}
	defer tx.Rollback()
	if err := validateTransport(tx, in.InboxID); err != nil {
		return Address{}, err
	}
	var existing Address
	if err := tx.Get(&existing, `SELECT id, created_at, updated_at, inbox_id, address, display_name, kind, enabled
		FROM email_addresses WHERE id=$1 FOR UPDATE`, id); errors.Is(err, sql.ErrNoRows) {
		return Address{}, ErrNotFound
	} else if err != nil {
		return Address{}, err
	}
	if existing.Kind == KindMailbox && (in.Kind != KindMailbox || in.InboxID != existing.InboxID || in.Address != existing.Address) {
		return Address{}, fmt.Errorf("the primary mailbox address is owned by its transport")
	}
	if existing.Kind == KindAlias && in.Kind != KindAlias {
		return Address{}, fmt.Errorf("an alias cannot be converted into a mailbox address")
	}
	var hasHistory bool
	if err := tx.Get(&hasHistory, `SELECT EXISTS(SELECT 1 FROM conversations WHERE address_id=$1)`, id); err != nil {
		return Address{}, err
	}
	if hasHistory && (in.Address != existing.Address || in.InboxID != existing.InboxID) {
		return Address{}, fmt.Errorf("an address with existing conversations cannot be moved or renamed")
	}
	var out Address
	err = tx.Get(&out, `UPDATE email_addresses
		SET inbox_id=$2, address=$3, display_name=$4, kind=$5, enabled=$6, updated_at=NOW()
		WHERE id=$1
		RETURNING id, created_at, updated_at, inbox_id, address, display_name, kind, enabled`,
		id, in.InboxID, in.Address, in.DisplayName, in.Kind, in.Enabled)
	if errors.Is(err, sql.ErrNoRows) {
		return Address{}, ErrNotFound
	}
	if err != nil {
		return Address{}, err
	}
	if err := replaceAccess(tx, id, in.UserIDs, in.TeamIDs); err != nil {
		return Address{}, err
	}
	if err := tx.Commit(); err != nil {
		return Address{}, err
	}
	out.UserIDs, out.TeamIDs = in.UserIDs, in.TeamIDs
	return out, nil
}

func (m *Manager) Delete(id int) (Address, error) {
	tx, err := m.db.Beginx()
	if err != nil {
		return Address{}, err
	}
	defer tx.Rollback()
	var out Address
	if err := tx.Get(&out, `SELECT id, created_at, updated_at, inbox_id, address, display_name, kind, enabled
		FROM email_addresses WHERE id=$1 FOR UPDATE`, id); errors.Is(err, sql.ErrNoRows) {
		return Address{}, ErrNotFound
	} else if err != nil {
		return Address{}, err
	}
	if out.Kind == KindMailbox {
		return Address{}, fmt.Errorf("a mailbox address is removed with its transport inbox")
	}
	var hasHistory bool
	if err := tx.Get(&hasHistory, `SELECT EXISTS(SELECT 1 FROM conversations WHERE address_id=$1)`, id); err != nil {
		return Address{}, err
	}
	if hasHistory {
		return Address{}, fmt.Errorf("an address with existing conversations must be disabled instead of deleted")
	}
	if _, err := tx.Exec(`DELETE FROM email_addresses WHERE id=$1`, id); err != nil {
		return Address{}, err
	}
	return out, tx.Commit()
}

// EnsureMailboxAddress creates the primary address for a newly created email
// transport. It deliberately does not overwrite an existing policy.
func (m *Manager) EnsureMailboxAddress(inboxID int, email string) (Address, error) {
	email, err := normalize(email)
	if err != nil {
		return Address{}, err
	}
	tx, err := m.db.Beginx()
	if err != nil {
		return Address{}, err
	}
	defer tx.Rollback()
	if err := validateTransport(tx, inboxID); err != nil {
		return Address{}, err
	}
	var out Address
	err = tx.Get(&out, `INSERT INTO email_addresses(inbox_id, address, kind, enabled)
		VALUES($1,$2,'mailbox',TRUE)
		ON CONFLICT (lower(address)) DO NOTHING
		RETURNING id, created_at, updated_at, inbox_id, address, display_name, kind, enabled`, inboxID, email)
	if errors.Is(err, sql.ErrNoRows) {
		err = tx.Get(&out, `SELECT id, created_at, updated_at, inbox_id, address, display_name, kind, enabled
			FROM email_addresses WHERE lower(address)=lower($1) FOR UPDATE`, email)
	}
	if err != nil {
		return Address{}, err
	}
	if out.InboxID != inboxID || out.Kind != KindMailbox {
		return Address{}, fmt.Errorf("email address is already assigned to another endpoint")
	}
	return out, tx.Commit()
}

func validateTransport(tx *sqlx.Tx, inboxID int) error {
	var exists bool
	if err := tx.Get(&exists, `SELECT EXISTS(SELECT 1 FROM inboxes WHERE id=$1 AND channel='email' AND deleted_at IS NULL)`, inboxID); err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("email transport inbox not found")
	}
	return nil
}

// replaceAccess replaces the address's team grants, and its user grants only
// when userIDs is non-nil.
func replaceAccess(tx *sqlx.Tx, addressID int, userIDs, teamIDs []int) error {
	if userIDs != nil {
		if err := validateIDs(tx, userIDs, `SELECT count(DISTINCT id) FROM users WHERE id=ANY($1) AND type='agent' AND deleted_at IS NULL`); err != nil {
			return err
		}
		if _, err := tx.Exec(`DELETE FROM email_address_users WHERE address_id=$1`, addressID); err != nil {
			return err
		}
		if len(userIDs) > 0 {
			if _, err := tx.Exec(`INSERT INTO email_address_users(address_id,user_id) SELECT $1, unnest($2::bigint[]) ON CONFLICT DO NOTHING`, addressID, pq.Array(userIDs)); err != nil {
				return err
			}
		}
	}
	if err := validateIDs(tx, teamIDs, `SELECT count(DISTINCT id) FROM teams WHERE id=ANY($1)`); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM email_address_teams WHERE address_id=$1`, addressID); err != nil {
		return err
	}
	if len(teamIDs) > 0 {
		if _, err := tx.Exec(`INSERT INTO email_address_teams(address_id,team_id) SELECT $1, unnest($2::int[]) ON CONFLICT DO NOTHING`, addressID, pq.Array(teamIDs)); err != nil {
			return err
		}
	}
	return nil
}

func validateIDs(tx *sqlx.Tx, ids []int, query string) error {
	if len(ids) == 0 {
		return nil
	}
	var count int
	if err := tx.Get(&count, query, pq.Array(ids)); err != nil {
		return err
	}
	if count != len(ids) {
		return fmt.Errorf("invalid address access principal")
	}
	return nil
}
