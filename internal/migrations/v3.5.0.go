package migrations

import (
	_ "embed"
	"encoding/json"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

//go:embed address_first.sql
var addressFirstSQL string

type legacyAddressView struct {
	Visibility string          `db:"visibility"`
	UserID     *int            `db:"user_id"`
	TeamID     *int            `db:"team_id"`
	Filters    json.RawMessage `db:"filters"`
}

type legacyAddressPolicy struct {
	all     bool
	userIDs map[int]struct{}
	teamIDs map[int]struct{}
}

// V3_5_0 promotes email endpoints into first-class addresses and retires the
// generic Views system. Existing email-alias views become address grants before
// the old table is deliberately destroyed.
func V3_5_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	tx, err := db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(addressFirstSQL); err != nil {
		return err
	}

	var hasViews bool
	if err := tx.Get(&hasViews, `SELECT to_regclass('public.views') IS NOT NULL`); err != nil {
		return err
	}
	if hasViews {
		var views []legacyAddressView
		if err := tx.Select(&views, `SELECT visibility, user_id, team_id, filters FROM views`); err != nil {
			return err
		}
		policies := map[string]*legacyAddressPolicy{}
		for _, view := range views {
			for _, alias := range findAliasFilters(view.Filters) {
				policy := policies[alias]
				if policy == nil {
					policy = &legacyAddressPolicy{userIDs: map[int]struct{}{}, teamIDs: map[int]struct{}{}}
					policies[alias] = policy
				}
				if view.Visibility == "all" {
					policy.all = true
				}
				if view.UserID != nil {
					policy.userIDs[*view.UserID] = struct{}{}
				}
				if view.TeamID != nil {
					policy.teamIDs[*view.TeamID] = struct{}{}
				}
			}
		}
		for alias, policy := range policies {
			var addressID int
			if err := tx.Get(&addressID, `SELECT id FROM email_addresses WHERE lower(address) = lower($1)`, alias); err != nil {
				continue
			}
			// A legacy all-agents view is broader than any overlapping team/user
			// view. Preserve that rather than accidentally turning it into an
			// admin-only address after the legacy table is removed.
			if _, err := tx.Exec(`UPDATE email_addresses SET restricted = $2, updated_at = NOW() WHERE id = $1`, addressID, !policy.all); err != nil {
				return err
			}
			// View visibility never bypassed the old transport ACL. Restricted
			// transports are migrated exclusively from that ACL below, including
			// future team membership (a team View must not become a new bypass).
			var transportRestricted bool
			if err := tx.Get(&transportRestricted, `SELECT EXISTS (
				SELECT 1 FROM email_addresses a JOIN inbox_access ia ON ia.inbox_id=a.inbox_id
				WHERE a.id=$1 AND ia.restricted)`, addressID); err != nil {
				return err
			}
			if transportRestricted {
				continue
			}
			for userID := range policy.userIDs {
				if _, err := tx.Exec(`INSERT INTO email_address_users(address_id, user_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, addressID, userID); err != nil {
					return err
				}
			}
			for teamID := range policy.teamIDs {
				if _, err := tx.Exec(`INSERT INTO email_address_teams(address_id, team_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, addressID, teamID); err != nil {
					return err
				}
			}
		}
	}
	if err := migrateLegacyInboxAccess(tx); err != nil {
		return err
	}

	if _, err := tx.Exec(`
		UPDATE roles
		SET permissions = array_remove(
			array_remove(
				array_remove(permissions, 'view:manage'),
				'shared_views:manage'
			),
			'conversations:write'
		);
		DROP TABLE IF EXISTS views CASCADE;
		DROP TYPE IF EXISTS view_visibility;
	`); err != nil {
		return err
	}
	return tx.Commit()
}

// migrateLegacyInboxAccess prevents an upgrade from widening access when the
// previous release had restricted a physical mailbox. Address permissions are
// now canonical, so direct members and members of legacy roles are copied to
// every endpoint on that transport before the hidden transport gate stops
// participating in address authorization.
func migrateLegacyInboxAccess(tx *sqlx.Tx) error {
	var exists bool
	if err := tx.Get(&exists, `SELECT to_regclass('public.inbox_access') IS NOT NULL`); err != nil {
		return err
	}
	if !exists {
		return nil
	}
	if _, err := tx.Exec(`
		UPDATE email_addresses AS address
		SET restricted = TRUE, updated_at = NOW()
		FROM inbox_access AS access
		WHERE access.inbox_id = address.inbox_id AND access.restricted;

		INSERT INTO email_address_users(address_id, user_id)
		SELECT DISTINCT address.id, access_user.user_id
		FROM email_addresses AS address
		JOIN inbox_access AS access ON access.inbox_id = address.inbox_id AND access.restricted
		JOIN inbox_users AS access_user ON access_user.inbox_id = access.inbox_id
		JOIN users AS agent ON agent.id = access_user.user_id
		WHERE agent.type = 'agent' AND agent.deleted_at IS NULL
		ON CONFLICT DO NOTHING;

		INSERT INTO email_address_users(address_id, user_id)
		SELECT DISTINCT address.id, role_member.user_id
		FROM email_addresses AS address
		JOIN inbox_access AS access ON access.inbox_id = address.inbox_id AND access.restricted
		JOIN inbox_roles AS access_role ON access_role.inbox_id = access.inbox_id
		JOIN user_roles AS role_member ON role_member.role_id = access_role.role_id
		JOIN users AS agent ON agent.id = role_member.user_id
		WHERE agent.type = 'agent' AND agent.deleted_at IS NULL
		ON CONFLICT DO NOTHING;
	`); err != nil {
		return err
	}
	return nil
}

func findAliasFilters(raw json.RawMessage) []string {
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil
	}
	seen := map[string]struct{}{}
	var walk func(any)
	walk = func(node any) {
		switch n := node.(type) {
		case []any:
			for _, item := range n {
				walk(item)
			}
		case map[string]any:
			if field, _ := n["field"].(string); field == "email_alias" {
				if operator, _ := n["operator"].(string); operator == "equals" {
					if alias, _ := n["value"].(string); alias != "" {
						seen[strings.ToLower(strings.TrimSpace(alias))] = struct{}{}
					}
				}
			}
			for _, item := range n {
				walk(item)
			}
		}
	}
	walk(value)
	aliases := make([]string, 0, len(seen))
	for alias := range seen {
		aliases = append(aliases, alias)
	}
	return aliases
}
