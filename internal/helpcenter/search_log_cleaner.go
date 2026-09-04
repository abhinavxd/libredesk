package helpcenter

import (
	"context"
	"time"
)

// RunSearchLogCleaner deletes stale search log rows on start and then once a day.
func (m *Manager) RunSearchLogCleaner(ctx context.Context) {
	startDelay := time.NewTimer(60 * time.Second)
	defer startDelay.Stop()
	select {
	case <-ctx.Done():
		return
	case <-startDelay.C:
	}
	if err := m.DeleteStaleSearchQueries(); err != nil {
		m.lo.Error("error cleaning stale search queries", "error", err)
	}

	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := m.DeleteStaleSearchQueries(); err != nil {
				m.lo.Error("error cleaning stale search queries", "error", err)
			}
		}
	}
}

// DeleteStaleSearchQueries removes search log rows older than the retention window.
func (m *Manager) DeleteStaleSearchQueries() error {
	res, err := m.q.DeleteStaleSearchQueries.Exec(searchLogRetentionDays)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n > 0 {
		m.lo.Info("cleaned stale help center search queries", "count", n, "retention_days", searchLogRetentionDays)
	}
	return nil
}
