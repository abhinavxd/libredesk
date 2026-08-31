package conversation

import (
	"context"
	"fmt"
	"time"
)

// RunUnsnoozer runs the conversation unsnoozer.
func (c *Manager) RunUnsnoozer(ctx context.Context, unsnoozeInterval time.Duration) {
	ticker := time.NewTicker(unsnoozeInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.unsnoozeAll(ctx)
		}
	}
}

// unsnoozeAll unsnoozes all snoozed conversations.
func (c *Manager) unsnoozeAll(ctx context.Context) {
	res, err := c.q.UnsnoozeAll.ExecContext(ctx)
	if err != nil {
		c.lo.Error("error unsnoozing all conversations", "error", err)
		return
	}
	rows, _ := res.RowsAffected()
	if rows > 0 {
		c.lo.Info(fmt.Sprintf("unsnoozed %d conversations", rows))
	}
}
