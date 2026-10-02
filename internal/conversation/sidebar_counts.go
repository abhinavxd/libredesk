package conversation

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	authzModels "github.com/jakedolan443/fernmail/internal/authz/models"
	"github.com/jakedolan443/fernmail/internal/conversation/models"
	"github.com/jakedolan443/fernmail/internal/envelope"
)

const sidebarCountsQueryTimeout = 10 * time.Second

// GetSidebarCounts returns unread-message counts for the addresses visible to
// this agent. Counts deliberately represent attention, not open conversation
// totals: opening or resolving a thread must not make an unread badge lie.
func (c *Manager) GetSidebarCounts(viewingUserID int, permissions []string, teamIDs []int) (models.SidebarCounts, error) {
	out := models.SidebarCounts{Addresses: map[int]int{}}
	lists := ListsForUserPermissions(permissions)
	if len(lists) == 0 {
		return out, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), sidebarCountsQueryTimeout)
	defer cancel()

	counts, err := c.getUnreadAddressCounts(ctx, viewingUserID, teamIDs, lists)
	if err != nil {
		return out, err
	}
	for addressID, count := range counts {
		out.Addresses[addressID] = count
		out.Unread += count
	}
	return out, nil
}

// getUnreadAddressCounts uses the same assignment and access predicates as the
// lists themselves. A message belongs to exactly one address, even when its
// source transport mailbox receives mail for several aliases.
func (c *Manager) getUnreadAddressCounts(ctx context.Context, userID int, teamIDs []int, lists []string) (map[int]int, error) {
	counts := map[int]int{}
	args := []any{userID}
	conditions, err := appendListTypeConditions(lists, userID, userID, teamIDs, &args)
	if err != nil {
		return counts, err
	}
	query := `SELECT conversations.address_id, COUNT(*) AS count
		FROM conversations
		JOIN inboxes ON inboxes.id = conversations.inbox_id
		JOIN conversation_messages AS messages ON messages.conversation_id = conversations.id
		LEFT JOIN conversation_last_seen AS seen ON seen.conversation_id = conversations.id AND seen.user_id = $1
		WHERE inboxes.channel = 'email'
		  AND conversations.address_id IS NOT NULL
		  AND messages.created_at > COALESCE(seen.last_seen_at, '1970-01-01'::TIMESTAMPTZ)
		  AND (messages.meta IS NULL OR NOT COALESCE((messages.meta->>'continuity_email')::boolean, false))
		  ` + listTypeWhereClause(conditions) + `
		GROUP BY conversations.address_id`
	var rows []struct {
		AddressID int `db:"address_id"`
		Count     int `db:"count"`
	}
	if err := c.db.SelectContext(ctx, &rows, query, args...); err != nil {
		c.lo.Error("error fetching unread address counts", "error", err)
		return counts, envelope.NewError(envelope.GeneralError, c.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	for _, row := range rows {
		counts[row.AddressID] = row.Count
	}
	return counts, nil
}

// ListsForUserPermissions returns conversation list types the user may access.
func ListsForUserPermissions(permissions []string) []string {
	lists := []string{}
	hasTeamAll := slices.Contains(permissions, authzModels.PermConversationsReadTeamAll)

	for _, perm := range permissions {
		if perm == authzModels.PermConversationsReadAll {
			return []string{models.AllConversations}
		}
		if perm == authzModels.PermConversationsReadUnassigned {
			lists = append(lists, models.UnassignedConversations)
		}
		if perm == authzModels.PermConversationsReadAssigned {
			lists = append(lists, models.AssignedConversations)
		}
		if perm == authzModels.PermConversationsReadTeamInbox && !hasTeamAll {
			lists = append(lists, models.TeamUnassignedConversations)
		}
		if perm == authzModels.PermConversationsReadTeamAll {
			lists = append(lists, models.TeamAllConversations)
		}
	}
	return lists
}

// appendListTypeConditions returns assignment conditions and always adds the
// address-level access check. This is shared by address lists, unread counts,
// search-like list operations, and notifications.
func appendListTypeConditions(listTypes []string, viewingUserID, userID int, teamIDs []int, args *[]any) ([]string, error) {
	if len(listTypes) == 0 {
		return nil, fmt.Errorf("no conversation list types specified")
	}
	if slices.Contains(listTypes, models.AllConversations) {
		*args = append(*args, viewingUserID)
		return []string{fmt.Sprintf("can_access_email_address(conversations.address_id, $%d)", len(*args))}, nil
	}
	conditions := make([]string, 0, len(listTypes))
	for _, lt := range listTypes {
		switch lt {
		case models.AssignedConversations:
			*args = append(*args, userID)
			conditions = append(conditions, fmt.Sprintf("conversations.assigned_user_id = $%d", len(*args)))
		case models.UnassignedConversations:
			conditions = append(conditions, "conversations.assigned_user_id IS NULL AND conversations.assigned_team_id IS NULL")
		case models.TeamUnassignedConversations:
			conditions = append(conditions, fmt.Sprintf("(conversations.assigned_team_id IN (%s) AND conversations.assigned_user_id IS NULL)", appendTeamIDArgs(teamIDs, args)))
		case models.TeamAllConversations:
			conditions = append(conditions, fmt.Sprintf("(conversations.assigned_team_id IN (%s))", appendTeamIDArgs(teamIDs, args)))
		case models.MentionedConversations:
			*args = append(*args, viewingUserID)
			conditions = append(conditions, fmt.Sprintf(`conversations.id IN (
				SELECT cm.conversation_id
				FROM conversation_mentions cm
				WHERE cm.mentioned_user_id = $%d
				   OR EXISTS(
					   SELECT 1 FROM team_members tm
					   WHERE tm.team_id = cm.mentioned_team_id AND tm.user_id = $%d
				   )
			)`, len(*args), len(*args)))
		default:
			return nil, fmt.Errorf("unknown conversation type: %s", lt)
		}
	}

	scope := "TRUE"
	if len(conditions) > 0 {
		scope = "(" + strings.Join(conditions, " OR ") + ")"
	}
	*args = append(*args, viewingUserID)
	return []string{fmt.Sprintf("%s AND can_access_email_address(conversations.address_id, $%d)", scope, len(*args))}, nil
}

func appendTeamIDArgs(teamIDs []int, args *[]any) string {
	if len(teamIDs) == 0 {
		return "NULL"
	}
	placeholders := make([]string, len(teamIDs))
	for i, id := range teamIDs {
		*args = append(*args, id)
		placeholders[i] = fmt.Sprintf("$%d", len(*args))
	}
	return strings.Join(placeholders, ",")
}

func listTypeWhereClause(conditions []string) string {
	if len(conditions) == 0 {
		return ""
	}
	return "AND (" + strings.Join(conditions, " OR ") + ")"
}
