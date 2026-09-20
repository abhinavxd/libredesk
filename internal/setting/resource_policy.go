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

// UpdateResourcePolicy merges the supplied fields under PostgreSQL's row lock.
// Concurrent editors of privacy and cache size cannot overwrite one another.
func (m *Manager) UpdateResourcePolicy(update resourcepolicy.Update) (resourcepolicy.Config, error) {
	update, err := resourcepolicy.NormalizeUpdate(update)
	if err != nil {
		return resourcepolicy.Blocked(), err
	}
	patch, err := json.Marshal(update)
	if err != nil {
		return resourcepolicy.Blocked(), err
	}
	defaults, err := json.Marshal(resourcepolicy.Default())
	if err != nil {
		return resourcepolicy.Blocked(), err
	}
	var raw types.JSONText
	if err := m.q.UpdateResourcePolicy.Get(&raw, defaults, patch); err != nil {
		return resourcepolicy.Blocked(), err
	}
	cfg := resourcepolicy.Default()
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return resourcepolicy.Blocked(), err
	}
	return resourcepolicy.Normalize(cfg)
}
