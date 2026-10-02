package conversation

import (
	"context"
	"slices"
	"time"

	"github.com/jakedolan443/fernmail/internal/envelope"
)

// MarkAddressRead marks the current snapshot of an address read for one agent.
// Use the same scope as lists/counts, across all pages and conversation statuses.
// Lock conversations in the same order as message insertion before taking the
// read snapshot. A message transaction cannot commit late with a timestamp
// below the saved watermark and silently disappear from unread counts.
func (m *Manager) MarkAddressRead(ctx context.Context, userID, addressID int, permissions []string, teamIDs []int) (time.Time, error) {
	lists := ListsForUserPermissions(permissions)
	denied := func() error {
		return envelope.NewError(envelope.PermissionError, m.i18n.T("status.deniedPermission"), nil)
	}
	if userID < 1 || addressID < 1 || !slices.Contains(permissions, "conversations:read") || len(lists) == 0 {
		return time.Time{}, denied()
	}
	args := []any{userID, addressID}
	conditions, err := appendListTypeConditions(lists, userID, userID, teamIDs, &args)
	if err != nil {
		return time.Time{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, sidebarCountsQueryTimeout)
	defer cancel()
	tx, err := m.db.BeginTxx(ctx, nil)
	if err != nil {
		return time.Time{}, err
	}
	defer tx.Rollback()
	// This must be a separate statement: READ COMMITTED takes a new snapshot
	// after any message writer holding the conversation lock has committed.
	_, err = tx.ExecContext(ctx, `SELECT conversations.id FROM conversations
		JOIN inboxes ON inboxes.id=conversations.inbox_id AND inboxes.channel='email'
		WHERE conversations.address_id=$2 AND can_access_email_address($2,$1)
		`+listTypeWhereClause(conditions)+`
		ORDER BY conversations.id FOR NO KEY UPDATE OF conversations`, args...)
	if err != nil {
		return time.Time{}, err
	}
	query := `WITH snapshot AS MATERIALIZED (
		SELECT statement_timestamp() AS marked_at, can_access_email_address($2, $1) AS allowed
	), marked AS (
		INSERT INTO conversation_last_seen(user_id, conversation_id, last_seen_at)
		SELECT $1, conversations.id, MAX(messages.created_at)
		FROM conversations
		JOIN inboxes ON inboxes.id = conversations.inbox_id AND inboxes.channel = 'email'
		JOIN conversation_messages messages ON messages.conversation_id = conversations.id
		CROSS JOIN snapshot
		WHERE conversations.address_id = $2 AND snapshot.allowed
		  AND messages.created_at <= snapshot.marked_at
		  ` + listTypeWhereClause(conditions) + `
		GROUP BY conversations.id
		ON CONFLICT (conversation_id, user_id) DO UPDATE
		SET last_seen_at = GREATEST(conversation_last_seen.last_seen_at, EXCLUDED.last_seen_at), updated_at = NOW()
		RETURNING conversation_id
	)
	SELECT marked_at, allowed FROM snapshot`
	var markedAt time.Time
	var allowed bool
	if err := tx.QueryRowxContext(ctx, query, args...).Scan(&markedAt, &allowed); err != nil {
		return time.Time{}, err
	}
	if !allowed {
		return time.Time{}, denied()
	}
	if err := tx.Commit(); err != nil {
		return time.Time{}, err
	}
	return markedAt, nil
}
