package conversation

import (
	"context"
	"errors"
	"time"

	"github.com/lib/pq"
)

// retentionBatchSize bounds one delete so a large backlog doesn't hold locks for long.
const retentionBatchSize = 500

// RunRetention deletes resolved conversations older than the retention period once an hour.
// The policy is read on every run, so a changed setting applies without a restart; 0 days keeps everything.
// With deleteContacts, a contact goes along with their last conversation.
func (c *Manager) RunRetention(ctx context.Context, policy func() (days int, deleteContacts bool)) {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			days, deleteContacts := policy()
			if days <= 0 {
				continue
			}
			conversations, contacts, err := c.DeleteExpiredConversations(ctx, days, deleteContacts)
			if err != nil && !errors.Is(err, context.Canceled) {
				c.lo.Error("error deleting expired conversations", "error", err)
			}
			if conversations > 0 {
				c.lo.Info("deleted expired conversations", "conversations", conversations, "contacts", contacts, "retention_days", days)
			}
		}
	}
}

// DeleteExpiredConversations deletes resolved conversations that have had no message and no status change
// for more than days days, in batches. With deleteContacts, contacts and visitors left without any
// conversation are deleted too; their avatars go with the unlinked-media sweep. Each batch is one
// transaction, so a failure leaves nothing half done and the next run picks the batch up again.
// Returns how many conversations and contacts went.
func (c *Manager) DeleteExpiredConversations(ctx context.Context, days int, deleteContacts bool) (conversations, contacts int, err error) {
	for {
		candidates, n, m, err := c.deleteExpiredBatch(ctx, days, deleteContacts)
		conversations += n
		contacts += m
		if err != nil || candidates < retentionBatchSize || n == 0 {
			return conversations, contacts, err
		}
	}
}

// deleteExpiredBatch deletes one batch of expired conversations and, if asked, the contacts they leave
// without any conversation. Candidates are locked before the delete, so a message being inserted into one
// of them commits first and the delete sees it; the contacts likewise, so a conversation created for one
// of them meanwhile waits on its foreign key and keeps the contact.
func (c *Manager) deleteExpiredBatch(ctx context.Context, days int, deleteContacts bool) (candidates, conversations, contacts int, err error) {
	tx, err := c.db.BeginTxx(ctx, nil)
	if err != nil {
		return 0, 0, 0, err
	}
	defer tx.Rollback()

	var ids, locked []int64
	if err := tx.Stmtx(c.q.GetExpiredConversations).SelectContext(ctx, &ids, days, retentionBatchSize); err != nil {
		return 0, 0, 0, err
	}
	if len(ids) == 0 {
		return 0, 0, 0, nil
	}
	if err := tx.Stmtx(c.q.LockConversations).SelectContext(ctx, &locked, pq.Array(ids)); err != nil {
		return 0, 0, 0, err
	}
	var contactIDs []int64
	if err := tx.Stmtx(c.q.DeleteExpiredConversations).SelectContext(ctx, &contactIDs, pq.Array(ids), days); err != nil {
		return 0, 0, 0, err
	}
	if deleteContacts && len(contactIDs) > 0 {
		var deleted []int64
		if err := tx.Stmtx(c.q.LockContacts).SelectContext(ctx, &locked, pq.Array(contactIDs)); err != nil {
			return 0, 0, 0, err
		}
		if err := tx.Stmtx(c.q.DeleteContactsWithoutConversations).SelectContext(ctx, &deleted, pq.Array(contactIDs)); err != nil {
			return 0, 0, 0, err
		}
		contacts = len(deleted)
	}
	if err := tx.Commit(); err != nil {
		return 0, 0, 0, err
	}
	return len(ids), len(contactIDs), contacts, nil
}
