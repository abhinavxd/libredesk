package conversation

import (
	"context"
	"time"
)

// RunDraftCleaner runs the draft cleanup routine every 2 hours.
func (c *Manager) RunDraftCleaner(ctx context.Context, retentionPeriod time.Duration) {
	if retentionPeriod <= 0 {
		c.lo.Info("draft retention period is non-positive, skipping draft cleaner", "retention_period", retentionPeriod)
		return
	}
	ticker := time.NewTicker(2 * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := c.DeleteStaleDrafts(ctx, retentionPeriod); err != nil {
				c.lo.Error("error cleaning stale drafts", "error", err)
			}
		}
	}
}
