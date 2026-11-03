package conversation

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
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
// conversation are deleted too, with their avatar. Returns how many conversations and contacts went.
func (c *Manager) DeleteExpiredConversations(ctx context.Context, days int, deleteContacts bool) (conversations, contacts int, err error) {
	for {
		var contactIDs []int64
		if err := c.q.DeleteExpiredConversations.SelectContext(ctx, &contactIDs, days, retentionBatchSize); err != nil {
			return conversations, contacts, err
		}
		conversations += len(contactIDs)
		if deleteContacts && len(contactIDs) > 0 {
			n, err := c.deleteContactsWithoutConversations(ctx, contactIDs)
			contacts += n
			if err != nil {
				return conversations, contacts, err
			}
		}
		if len(contactIDs) < retentionBatchSize {
			return conversations, contacts, nil
		}
	}
}

// deleteContactsWithoutConversations deletes those of the given contacts that have no conversation left and
// removes their avatars from the media store. Returns how many contacts went.
func (c *Manager) deleteContactsWithoutConversations(ctx context.Context, contactIDs []int64) (int, error) {
	var avatars []sql.NullString
	if err := c.q.DeleteContactsWithoutConversations.SelectContext(ctx, &avatars, pq.Array(contactIDs)); err != nil {
		return 0, err
	}
	for _, avatar := range avatars {
		if !avatar.Valid || avatar.String == "" || c.mediaStore == nil {
			continue
		}
		if err := c.mediaStore.Delete(filepath.Base(avatar.String)); err != nil {
			c.lo.Error("error deleting avatar of retired contact", "file", filepath.Base(avatar.String), "error", err)
		}
	}
	return len(avatars), nil
}
