package user

import (
	"encoding/json"
	"fmt"
	"net/mail"
	"slices"

	"github.com/jmoiron/sqlx"
)

func (m *Manager) GetImageSenders(userID int) ([]string, error) {
	return m.GetImageSendersTx(nil, userID)
}

func (m *Manager) GetImageSendersTx(tx *sqlx.Tx, userID int) ([]string, error) {
	query := m.q.GetImageSenders
	if tx != nil {
		query = tx.Stmtx(query)
		defer query.Close()
	}
	var raw []byte
	if err := query.Get(&raw, userID); err != nil {
		return nil, err
	}
	var senders []string
	err := json.Unmarshal(raw, &senders)
	return senders, err
}

func (m *Manager) SetImageSender(userID int, sender string, trust bool) error {
	address, err := mail.ParseAddress(sender)
	if err != nil || address.Address != sender || len(sender) > 254 {
		return fmt.Errorf("invalid sender address")
	}
	result, err := m.q.SetImageSender.Exec(userID, sender, trust)
	if err != nil {
		return err
	}
	if n, err := result.RowsAffected(); err != nil || n == 0 {
		senders, err := m.GetImageSenders(userID)
		if err != nil || slices.Contains(senders, sender) != trust {
			return fmt.Errorf("unable to save sender permission (maximum 100 senders)")
		}
	}
	return nil
}
