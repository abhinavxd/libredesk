package user

import (
	"fmt"
	"strings"

	"github.com/jakedolan443/fernmail/internal/user/models"
	"github.com/volatiletech/null/v9"
)

// ResolveEmailSender stores only the address and display name needed by messages.
// Sender identities are not editable address-book or CRM records. The legacy
// users table keeps existing message foreign keys valid across upgrades.
func (u *Manager) ResolveEmailSender(sender *models.User) error {
	address := strings.ToLower(strings.TrimSpace(sender.Email.String))
	if address == "" {
		return fmt.Errorf("email sender requires an address")
	}
	sender.Email = null.StringFrom(address)
	// Serialize per address so concurrent arrivals cannot create duplicate identities.
	tx, err := u.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.Exec(`SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, address); err != nil {
		return err
	}
	err = tx.QueryRow(`WITH existing AS (
  SELECT id FROM users WHERE type = 'contact' AND lower(email) = $1 ORDER BY id LIMIT 1
 ), inserted AS (
  INSERT INTO users (type, email, first_name, last_name, enabled)
  SELECT 'contact', $1, $2, $3, true WHERE NOT EXISTS (SELECT 1 FROM existing)
  RETURNING id
 ) SELECT id FROM existing UNION ALL SELECT id FROM inserted`, address, sender.FirstName, sender.LastName).Scan(&sender.ID)
	if err != nil {
		return fmt.Errorf("resolving email sender: %w", err)
	}
	return tx.Commit()
}
