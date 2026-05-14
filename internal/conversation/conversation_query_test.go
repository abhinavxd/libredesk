package conversation

import (
	"strings"
	"testing"

	cmodels "github.com/abhinavxd/libredesk/internal/conversation/models"
)

const conversationsListTestQuery = "SELECT conversations.id FROM conversations WHERE 1=1 %s"

func TestMakeConversationsListQueryForTeamAll(t *testing.T) {
	query, args, err := (&Manager{}).makeConversationsListQuery(
		42,
		0,
		[]int{7},
		[]string{cmodels.TeamAllConversations},
		conversationsListTestQuery,
		"",
		"",
		1,
		30,
		"[]",
	)
	if err != nil {
		t.Fatalf("make query: %v", err)
	}

	if !strings.Contains(query, "conversations.assigned_team_id IN ($3)") {
		t.Fatalf("query does not filter team conversations:\n%s", query)
	}
	if strings.Contains(query, "conversations.assigned_user_id IS NULL") {
		t.Fatalf("team all query should include assigned conversations:\n%s", query)
	}

	if len(args) != 5 {
		t.Fatalf("args length = %d, want 5: %#v", len(args), args)
	}
	if args[0] != 42 || args[1] != false || args[2] != 7 || args[3] != 30 || args[4] != 0 {
		t.Fatalf("unexpected args: %#v", args)
	}
}

func TestMakeConversationsListQueryForTeamUnassigned(t *testing.T) {
	query, _, err := (&Manager{}).makeConversationsListQuery(
		42,
		0,
		[]int{7},
		[]string{cmodels.TeamUnassignedConversations},
		conversationsListTestQuery,
		"",
		"",
		1,
		30,
		"[]",
	)
	if err != nil {
		t.Fatalf("make query: %v", err)
	}

	if !strings.Contains(query, "conversations.assigned_team_id IN ($3)") {
		t.Fatalf("query does not filter team conversations:\n%s", query)
	}
	if !strings.Contains(query, "conversations.assigned_user_id IS NULL") {
		t.Fatalf("team unassigned query should exclude assigned conversations:\n%s", query)
	}
}
