package conversation

import (
	"slices"
	"strings"
	"testing"

	authzModels "github.com/abhinavxd/libredesk/internal/authz/models"
	"github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/dbutil"
	vmodels "github.com/abhinavxd/libredesk/internal/view/models"
	"github.com/jmoiron/sqlx/types"
)

// stubSettingsStore satisfies the settingsStore dependency for query building tests.
type stubSettingsStore struct{}

func (stubSettingsStore) GetAppRootURL() (string, error) { return "", nil }

func (stubSettingsStore) GetByPrefix(prefix string) (types.JSONText, error) {
	return types.JSONText(`{}`), nil
}

func (stubSettingsStore) Get(key string) (types.JSONText, error) {
	return types.JSONText(`"Etc/UTC"`), nil
}

// newTestManager returns a Manager with only the dependencies query building needs.
func newTestManager() *Manager {
	return &Manager{settingsStore: stubSettingsStore{}}
}

func TestListsForUserPermissions(t *testing.T) {
	agentPerms := []string{
		authzModels.PermConversationsReadAll,
		authzModels.PermConversationsReadUnassigned,
		authzModels.PermConversationsReadAssigned,
		authzModels.PermConversationsReadTeamInbox,
		authzModels.PermConversationsReadTeamAll,
		authzModels.PermConversationsRead,
	}

	lists := ListsForUserPermissions(agentPerms)
	if len(lists) != 1 || lists[0] != models.AllConversations {
		t.Fatalf("read_all should short-circuit to all only, got %v", lists)
	}

	restricted := []string{
		authzModels.PermConversationsReadAssigned,
		authzModels.PermConversationsReadUnassigned,
		authzModels.PermConversationsReadTeamInbox,
		authzModels.PermConversationsReadTeamAll,
	}
	lists = ListsForUserPermissions(restricted)
	want := []string{
		models.UnassignedConversations,
		models.AssignedConversations,
		models.TeamAllConversations,
	}
	if len(lists) != len(want) {
		t.Fatalf("got %d list types, want %d: %v", len(lists), len(want), lists)
	}
	for _, w := range want {
		if !slices.Contains(lists, w) {
			t.Fatalf("missing list type %q in %v", w, lists)
		}
	}
	if slices.Contains(lists, models.TeamUnassignedConversations) {
		t.Fatalf("team unassigned should be omitted when team all is present: %v", lists)
	}
}

func TestUserCanAccessView(t *testing.T) {
	userID := 5
	teamIDs := []int{2, 3}

	personalOther := vmodels.View{Visibility: vmodels.VisibilityUser, UserID: intPtr(99)}
	if UserCanAccessView(personalOther, userID, teamIDs) {
		t.Fatal("expected no access to another user's personal view")
	}

	personalOwn := vmodels.View{Visibility: vmodels.VisibilityUser, UserID: intPtr(userID)}
	if !UserCanAccessView(personalOwn, userID, teamIDs) {
		t.Fatal("expected access to own personal view")
	}

	sharedAll := vmodels.View{Visibility: vmodels.VisibilityAll}
	if !UserCanAccessView(sharedAll, userID, teamIDs) {
		t.Fatal("expected access to shared-all view")
	}

	teamView := vmodels.View{Visibility: vmodels.VisibilityTeam, TeamID: intPtr(2)}
	if !UserCanAccessView(teamView, userID, teamIDs) {
		t.Fatal("expected access to team view for member")
	}

	otherTeam := vmodels.View{Visibility: vmodels.VisibilityTeam, TeamID: intPtr(9)}
	if UserCanAccessView(otherTeam, userID, teamIDs) {
		t.Fatal("expected no access to other team view")
	}
}

func TestMakeConversationsCountQueryOpenAndInboxFilter(t *testing.T) {
	m := newTestManager()
	filters := `[{"model":"conversations","field":"inbox_id","operator":"equals","value":"4"}]`
	query, args, err := m.makeConversationsCountQuery(nil, 1, []int{}, []string{models.AllConversations}, filters, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(query, "category = 'open'") {
		t.Fatalf("view counts must not force open category: %s", query)
	}
	if !strings.Contains(query, "inbox_id") {
		t.Fatalf("expected inbox_id filter in query: %s", query)
	}
	if len(args) < 1 {
		t.Fatalf("expected at least 1 arg, got %d: %v", len(args), args)
	}
}

func TestMakeConversationsCountQueryAssignedList(t *testing.T) {
	m := newTestManager()
	query, args, err := m.makeConversationsCountQuery(nil, 7, []int{}, []string{models.AssignedConversations}, "[]", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(query, "assigned_user_id = $") {
		t.Fatalf("expected assigned_user_id condition: %s", query)
	}
	if len(args) != 1 || args[0] != 7 {
		t.Fatalf("expected single assignee arg, got %v", args)
	}
}

func TestMakeConversationsCountQueryEmptyListTypes(t *testing.T) {
	m := newTestManager()
	_, _, err := m.makeConversationsCountQuery(nil, 1, []int{}, []string{}, "[]", "")
	if err == nil {
		t.Fatal("expected error for empty list types")
	}
}

func TestMakeViewCountsQuerySingleStatement(t *testing.T) {
	m := newTestManager()
	views := []vmodels.View{
		{ID: 3, Filters: []byte(`[{"model":"conversations","field":"inbox_id","operator":"equals","value":"1"}]`)},
		{ID: 9, Filters: []byte(`[{"model":"conversations","field":"inbox_id","operator":"equals","value":"2"}]`)},
	}

	query, args, err := m.makeViewCountsQuery(4, []int{}, []string{models.AllConversations}, views)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Count(query, "UNION ALL") != 1 {
		t.Fatalf("expected the two views to be unioned into one statement: %s", query)
	}
	if !strings.Contains(query, "SELECT 3 AS view_id") || !strings.Contains(query, "SELECT 9 AS view_id") {
		t.Fatalf("expected both view ids to be selected: %s", query)
	}
	// Each view contributes one inbox_id filter argument, numbered continuously.
	if len(args) != 2 {
		t.Fatalf("expected 2 args across both views, got %d: %v", len(args), args)
	}
	if !strings.Contains(query, "$1") || !strings.Contains(query, "$2") {
		t.Fatalf("expected continuous placeholder numbering: %s", query)
	}
	if strings.Contains(query, "$3") {
		t.Fatalf("placeholder numbering ran past the bound args: %s", query)
	}
}

func TestMakeConversationsCountQueryEmptyTeamIDs(t *testing.T) {
	m := newTestManager()
	// A user with the team permission but no teams must still produce valid SQL.
	query, _, err := m.makeConversationsCountQuery(nil, 1, []int{}, []string{models.TeamAllConversations}, "[]", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(query, "IN ()") {
		t.Fatalf("empty team list produced invalid SQL: %s", query)
	}
	if !strings.Contains(query, "IN (NULL)") {
		t.Fatalf("expected empty team list to match nothing: %s", query)
	}
}

func TestViewFiltersValidateAgainstListFields(t *testing.T) {
	filters := `[{"model":"conversations","field":"inbox_id","operator":"equals","value":"1"}]`
	if err := dbutil.ValidateFilters(filters, conversationListAllowedFields, conversationFilterRenderers); err != nil {
		t.Fatalf("expected valid filters, got %v", err)
	}
}

func intPtr(v int) *int { return &v }
