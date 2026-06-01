package conversation

import (
	"strings"
	"testing"

	cmodels "github.com/abhinavxd/libredesk/internal/conversation/models"
)

const conversationsListTestQuery = "SELECT conversations.id FROM conversations WHERE 1=1 %s"

func TestMakeConversationsListQueryAddsUnreadOnlyFilter(t *testing.T) {
	query, args, err := (&Manager{}).makeConversationsListQuery(
		42,
		42,
		nil,
		[]string{cmodels.AssignedConversations},
		conversationsListTestQuery,
		"",
		"",
		true,
		1,
		30,
		"[]",
	)
	if err != nil {
		t.Fatalf("make query: %v", err)
	}

	for _, want := range []string{
		"conversations.assigned_user_id = $3",
		"EXISTS (",
		"FROM conversation_messages cm",
		"FROM conversation_last_seen cls",
		"cls.user_id = $1",
		"continuity_email",
	} {
		if !strings.Contains(query, want) {
			t.Fatalf("query does not contain %q:\n%s", want, query)
		}
	}

	if len(args) != 5 {
		t.Fatalf("args length = %d, want 5: %#v", len(args), args)
	}
	if args[0] != 42 || args[1] != false || args[2] != 42 || args[3] != 30 || args[4] != 0 {
		t.Fatalf("unexpected args: %#v", args)
	}
}

func TestMakeConversationsListQueryOmitsUnreadOnlyFilter(t *testing.T) {
	query, _, err := (&Manager{}).makeConversationsListQuery(
		42,
		42,
		nil,
		[]string{cmodels.AssignedConversations},
		conversationsListTestQuery,
		"",
		"",
		false,
		1,
		30,
		"[]",
	)
	if err != nil {
		t.Fatalf("make query: %v", err)
	}

	if strings.Contains(query, "conversation_last_seen cls") {
		t.Fatalf("query contains unread filter when unreadOnly is false:\n%s", query)
	}
}
