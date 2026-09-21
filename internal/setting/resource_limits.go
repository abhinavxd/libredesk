package setting

import (
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/abhinavxd/libredesk/internal/resourceusage"
	"github.com/jmoiron/sqlx/types"
)

// GetResourceLimits returns the persisted resource budgets, falling back to
// the supplied configuration values for installations that predate this
// setting.
func (m *Manager) GetResourceLimits(defaults resourceusage.Limits) (resourceusage.Limits, error) {
	var raw types.JSONText
	if err := m.q.Get.Get(&raw, "system.resource_limits"); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return defaults, nil
		}
		return resourceusage.Limits{}, err
	}
	var limits resourceusage.Limits
	if err := json.Unmarshal(raw, &limits); err != nil {
		return resourceusage.Limits{}, err
	}
	return resourceusage.Normalize(limits)
}

// SetResourceLimits persists validated resource budgets.
func (m *Manager) SetResourceLimits(limits resourceusage.Limits) error {
	limits, err := resourceusage.Normalize(limits)
	if err != nil {
		return err
	}
	raw, err := json.Marshal(limits)
	if err != nil {
		return err
	}
	_, err = m.q.SetResourceLimits.Exec(raw)
	return err
}
