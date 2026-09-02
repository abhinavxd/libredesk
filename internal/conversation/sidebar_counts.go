package conversation

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"

	authzModels "github.com/abhinavxd/libredesk/internal/authz/models"
	"github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/abhinavxd/libredesk/internal/envelope"
	vmodels "github.com/abhinavxd/libredesk/internal/view/models"
)

const sidebarCountsViewBatchSize = 50
const sidebarCountsQueryTimeout = 10 * time.Second

// sidebarCountsScanCap bounds each view count's scan; the UI shows anything above 99 as "99+".
const sidebarCountsScanCap = 100

// GetSidebarCounts returns open counts for the standard inboxes and a count per accessible view.
func (c *Manager) GetSidebarCounts(viewingUserID int, permissions []string, teamIDs []int, views []vmodels.View) (models.SidebarCounts, error) {
	out := models.SidebarCounts{Views: map[int]int{}}
	ctx, cancel := context.WithTimeout(context.Background(), sidebarCountsQueryTimeout)
	defer cancel()

	if err := c.fillStandardSidebarCounts(ctx, &out, viewingUserID, permissions); err != nil {
		return out, err
	}

	lists := ListsForUserPermissions(permissions)
	if len(lists) == 0 {
		return out, nil
	}

	accessible := make([]vmodels.View, 0, len(views))
	for _, view := range views {
		if UserCanAccessView(view, viewingUserID, teamIDs) {
			accessible = append(accessible, view)
		}
	}
	for start := 0; start < len(accessible); start += sidebarCountsViewBatchSize {
		batch := accessible[start:min(start+sidebarCountsViewBatchSize, len(accessible))]
		viewCounts, err := c.getViewCounts(ctx, viewingUserID, teamIDs, lists, batch)
		if err != nil {
			return out, err
		}
		maps.Copy(out.Views, viewCounts)
	}
	return out, nil
}

// GetViewCount returns the capped open count for one view.
func (c *Manager) GetViewCount(viewingUserID int, permissions []string, teamIDs []int, view vmodels.View) (int, error) {
	lists := ListsForUserPermissions(permissions)
	if len(lists) == 0 {
		return 0, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), sidebarCountsQueryTimeout)
	defer cancel()

	counts, err := c.getViewCounts(ctx, viewingUserID, teamIDs, lists, []vmodels.View{view})
	if err != nil {
		return 0, err
	}
	return counts[view.ID], nil
}

func (c *Manager) fillStandardSidebarCounts(ctx context.Context, out *models.SidebarCounts, userID int, permissions []string) error {
	var row struct {
		Assigned   int `db:"assigned"`
		Unassigned int `db:"unassigned"`
		Mentioned  int `db:"mentioned"`
		All        int `db:"all"`
	}

	if err := c.q.GetSidebarStandardCounts.GetContext(ctx, &row, userID); err != nil {
		c.lo.Error("error fetching sidebar standard counts", "error", err)
		return envelope.NewError(envelope.GeneralError, c.i18n.T("globals.messages.somethingWentWrong"), nil)
	}

	if slices.Contains(permissions, authzModels.PermConversationsReadAssigned) {
		out.Assigned = row.Assigned
	}
	if slices.Contains(permissions, authzModels.PermConversationsReadUnassigned) {
		out.Unassigned = row.Unassigned
	}
	if slices.Contains(permissions, authzModels.PermConversationsRead) {
		out.Mentioned = row.Mentioned
	}
	if slices.Contains(permissions, authzModels.PermConversationsReadAll) {
		out.All = row.All
	}
	return nil
}

// getViewCounts returns the conversation count per view ID in one round trip.
func (c *Manager) getViewCounts(ctx context.Context, userID int, teamIDs []int, listTypes []string, views []vmodels.View) (map[int]int, error) {
	counts := map[int]int{}
	if len(views) == 0 {
		return counts, nil
	}

	query, qArgs, err := c.makeViewCountsQuery(userID, teamIDs, listTypes, views)
	if err != nil {
		c.lo.Error("error making view counts query", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, c.i18n.T("globals.messages.somethingWentWrong"), nil)
	}

	var rows []struct {
		ViewID int `db:"view_id"`
		Count  int `db:"count"`
	}
	if err := c.db.SelectContext(ctx, &rows, query, qArgs...); err != nil {
		c.lo.Error("error fetching view counts", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, c.i18n.T("globals.messages.somethingWentWrong"), nil)
	}

	for _, row := range rows {
		counts[row.ViewID] = row.Count
	}
	return counts, nil
}

// makeViewCountsQuery unions one counting subquery per view into a single (view_id, count) statement.
func (c *Manager) makeViewCountsQuery(userID int, teamIDs []int, listTypes []string, views []vmodels.View) (string, []any, error) {
	var (
		args  = []any{}
		parts = make([]string, 0, len(views))
		loc   = c.filterLocation()
	)

	for _, view := range views {
		countQuery, nextArgs, err := c.makeConversationsCountQuery(args, userID, teamIDs, listTypes, string(view.Filters), loc)
		if err != nil {
			return "", nil, err
		}
		parts = append(parts, fmt.Sprintf("SELECT %d AS view_id, (SELECT COUNT(*) FROM (%s LIMIT %d) capped) AS count", view.ID, countQuery, sidebarCountsScanCap))
		args = nextArgs
	}

	return strings.Join(parts, " UNION ALL "), args, nil
}

// makeConversationsCountQuery builds a query selecting matching conversations, with placeholders continuing after existingArgs.
func (c *Manager) makeConversationsCountQuery(existingArgs []any, userID int, teamIDs []int, listTypes []string, filtersJSON, loc string) (string, []any, error) {
	if len(listTypes) == 0 {
		return "", nil, fmt.Errorf("no conversation list types specified")
	}

	qArgs := existingArgs
	conditions, err := appendListTypeConditions(listTypes, userID, userID, teamIDs, &qArgs)
	if err != nil {
		return "", nil, err
	}

	baseQuery := fmt.Sprintf(c.q.GetConversationsCountBase, listTypeWhereClause(conditions))

	return dbutil.BuildFilterQuery(baseQuery, qArgs, filtersJSON, conversationListAllowedFields, conversationFilterRenderers, loc)
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

// UserCanAccessView reports whether a user can access a view.
func UserCanAccessView(view vmodels.View, userID int, teamIDs []int) bool {
	switch view.Visibility {
	case vmodels.VisibilityUser:
		return view.UserID != nil && *view.UserID == userID
	case vmodels.VisibilityAll:
		return true
	case vmodels.VisibilityTeam:
		if view.TeamID == nil {
			return false
		}
		return slices.Contains(teamIDs, *view.TeamID)
	default:
		return false
	}
}
