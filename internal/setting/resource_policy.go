package setting

import (
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/abhinavxd/libredesk/internal/resourcepolicy"
	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/types"
)

func (m *Manager) GetResourcePolicy() (resourcepolicy.Config, error) {
	return m.GetResourcePolicyTx(nil)
}

func (m *Manager) GetResourcePolicyTx(tx *sqlx.Tx) (resourcepolicy.Config, error) {
	query := m.q.Get
	if tx != nil {
		query = tx.Stmtx(query)
		defer query.Close()
	}
	var raw types.JSONText
	if err := query.Get(&raw, "security.resource_policy"); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return resourcepolicy.Default(), nil
		}
		return resourcepolicy.Blocked(), err
	}
	cfg := resourcepolicy.Config{MaxCacheBytes: resourcepolicy.DefaultMaxCacheBytes}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return resourcepolicy.Blocked(), err
	}
	return resourcepolicy.Normalize(cfg)
}

func (m *Manager) SetResourcePolicy(cfg resourcepolicy.Config) error {
	cfg, err := resourcepolicy.Normalize(cfg)
	if err != nil {
		return err
	}
	raw, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	_, err = m.q.SetResourcePolicy.Exec(raw)
	return err
}
