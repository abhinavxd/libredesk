package conversation

import (
	"context"
	"strings"
	"testing"

	authzModels "github.com/jakedolan443/fernmail/internal/authz/models"
	"github.com/jakedolan443/fernmail/internal/conversation/models"
	"github.com/jakedolan443/fernmail/internal/testutil"
	"github.com/jmoiron/sqlx/types"
)

func TestUnreadAddressCountsOnlyIncludeAccessibleAddresses(t *testing.T) {
	db := testutil.NewDB(t, "unread_address_counts")
	db.MustExec(`
		INSERT INTO users(type, email, first_name) VALUES
			('agent', 'reader@example.test', 'Reader'),
			('contact', 'sender@example.test', 'Sender');
		INSERT INTO inboxes(name, channel) VALUES ('Director transport', 'email'), ('Support transport', 'email');
	`)
	var reader, sender, directorInbox, supportInbox, directorAddress, supportAddress int
	for _, row := range []struct {
		dest *int
		q    string
	}{
		{&reader, `SELECT id FROM users WHERE email='reader@example.test'`},
		{&sender, `SELECT id FROM users WHERE email='sender@example.test'`},
		{&directorInbox, `SELECT id FROM inboxes WHERE name='Director transport'`},
		{&supportInbox, `SELECT id FROM inboxes WHERE name='Support transport'`},
	} {
		if err := db.Get(row.dest, row.q); err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Get(&directorAddress, `INSERT INTO email_addresses(inbox_id,address,kind) VALUES($1,'director@example.test','mailbox') RETURNING id`, directorInbox); err != nil {
		t.Fatal(err)
	}
	if err := db.Get(&supportAddress, `INSERT INTO email_addresses(inbox_id,address,kind) VALUES($1,'support@example.test','mailbox') RETURNING id`, supportInbox); err != nil {
		t.Fatal(err)
	}
	db.MustExec(`INSERT INTO email_address_users(address_id,user_id) VALUES($1,$2)`, directorAddress, reader)

	var directorConversation, supportConversation int
	if err := db.Get(&directorConversation, `INSERT INTO conversations(contact_id,inbox_id,address_id,status_id)
		VALUES($1,$2,$3,(SELECT id FROM conversation_statuses WHERE name='Open')) RETURNING id`, sender, directorInbox, directorAddress); err != nil {
		t.Fatal(err)
	}
	if err := db.Get(&supportConversation, `INSERT INTO conversations(contact_id,inbox_id,address_id,status_id)
		VALUES($1,$2,$3,(SELECT id FROM conversation_statuses WHERE name='Open')) RETURNING id`, sender, supportInbox, supportAddress); err != nil {
		t.Fatal(err)
	}
	db.MustExec(`
		INSERT INTO conversation_messages(conversation_id,sender_id,sender_type,type,status,text_content,meta) VALUES
			($1,$2,'contact','incoming','received','first director message','{}'),
			($1,$2,'contact','incoming','received','second director message','{}'),
			($1,$2,'contact','incoming','received','third director message','{}'),
			($1,$2,'contact','incoming','received','continuity', '{"continuity_email":true}'),
			($3,$2,'contact','incoming','received','hidden support message','{}');
	`, directorConversation, sender, supportConversation)

	m := &Manager{db: db}
	permissions := []string{authzModels.PermConversationsRead, authzModels.PermConversationsReadAll}
	counts, err := m.GetSidebarCounts(reader, permissions, nil)
	if err != nil {
		t.Fatal(err)
	}
	if counts.Unread != 3 || counts.Addresses[directorAddress] != 3 || len(counts.Addresses) != 1 {
		t.Fatalf("counts = %#v, want only three unread director messages", counts)
	}

	// Per-agent reads affect attention badges without touching other addresses.
	db.MustExec(`INSERT INTO conversation_last_seen(user_id,conversation_id,last_seen_at) VALUES($1,$2,NOW()+interval '1 second')`, reader, directorConversation)
	counts, err = m.GetSidebarCounts(reader, permissions, nil)
	if err != nil {
		t.Fatal(err)
	}
	if counts.Unread != 0 || len(counts.Addresses) != 0 {
		t.Fatalf("counts after read = %#v, want no unread messages", counts)
	}
}

// stubSettingsStore satisfies the settingsStore dependency for query-building
// and unrelated conversation tests.
type stubSettingsStore struct{}

func (stubSettingsStore) GetAppRootURL() (string, error) { return "", nil }

func (stubSettingsStore) GetByPrefix(prefix string) (types.JSONText, error) {
	return types.JSONText(`{}`), nil
}

func (stubSettingsStore) Get(key string) (types.JSONText, error) {
	return types.JSONText(`"Etc/UTC"`), nil
}

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
	if len(lists) != 3 || !containsAll(lists,
		models.UnassignedConversations,
		models.AssignedConversations,
		models.TeamAllConversations,
	) {
		t.Fatalf("restricted lists = %v", lists)
	}
}

func containsAll(values []string, want ...string) bool {
	for _, item := range want {
		if !strings.Contains(","+strings.Join(values, ",")+",", ","+item+",") {
			return false
		}
	}
	return true
}

func TestListConditionsAlwaysRequireAddressAccess(t *testing.T) {
	args := []any{7}
	conditions, err := appendListTypeConditions([]string{models.AllConversations}, 7, 7, nil, &args)
	if err != nil {
		t.Fatal(err)
	}
	if len(conditions) != 1 || !strings.Contains(conditions[0], "can_access_email_address") {
		t.Fatalf("all list conditions must enforce address access: %v", conditions)
	}

	args = []any{7}
	conditions, err = appendListTypeConditions([]string{models.TeamAllConversations}, 7, 7, nil, &args)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(conditions[0], "IN (NULL)") || !strings.Contains(conditions[0], "can_access_email_address") {
		t.Fatalf("empty team scope must be valid and address-gated: %s", conditions[0])
	}
}

func TestGetUnreadAddressCountsRejectsEmptyListScope(t *testing.T) {
	m := &Manager{}
	if _, err := m.getUnreadAddressCounts(context.Background(), 1, nil, nil); err == nil {
		t.Fatal("expected an empty list scope to be rejected")
	}
}
