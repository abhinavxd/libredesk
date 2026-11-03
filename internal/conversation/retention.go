package conversation

import (
	"context"
	"errors"
	"time"
)

// retentionBatchSize bounds one delete so a large backlog doesn't hold locks for long.
const retentionBatchSize = 500

// RunRetention deletes resolved conversations older than the retention period once an hour.
// retentionDays is read on every run, so a changed setting applies without a restart; 0 keeps everything.
func (c *Manager) RunRetention(ctx context.Context, retentionDays func() int) {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			days := retentionDays()
			if days <= 0 {
				continue
			}
			deleted, err := c.DeleteExpiredConversations(ctx, days)
			if err != nil && !errors.Is(err, context.Canceled) {
				c.lo.Error("error deleting expired conversations", "error", err)
			}
			if deleted > 0 {
				c.lo.Info("deleted expired conversations", "count", deleted, "retention_days", days)
			}
		}
	}
}

// DeleteExpiredConversations deletes resolved conversations that have had no message and no status change
// for more than days days, in batches, and returns how many went.
func (c *Manager) DeleteExpiredConversations(ctx context.Context, days int) (int, error) {
	total := 0
	for {
		res, err := c.q.DeleteExpiredConversations.ExecContext(ctx, days, retentionBatchSize)
		if err != nil {
			return total, err
		}
		n, err := res.RowsAffected()
		if err != nil {
			return total, err
		}
		total += int(n)
		if n < retentionBatchSize {
			return total, nil
		}
	}
}
