package conversation

import (
	"testing"

	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/zerodha/logf"
)

func TestPortalConversationListScopesToContact(t *testing.T) {
	db := testutil.NewDB(t, "portal_contact_scope")
	var contacts [2]int
	for i, email := range []string{"portal-one@example.com", "portal-two@example.com"} {
		if err := db.Get(&contacts[i], `INSERT INTO users (type, email, first_name, last_name) VALUES ('contact', $1, 'Portal', '') RETURNING id`, email); err != nil {
			t.Fatal(err)
		}
	}
	var inboxID int
	if err := db.Get(&inboxID, `INSERT INTO inboxes (name, channel) VALUES ('Portal test', 'email') RETURNING id`); err != nil {
		t.Fatal(err)
	}
	want := make([]string, 0, 2)
	for _, contactID := range contacts {
		var uuid string
		if err := db.Get(&uuid, `INSERT INTO conversations (contact_id, inbox_id, status_id, subject) VALUES ($1, $2, (SELECT id FROM conversation_statuses LIMIT 1), 'Private ticket') RETURNING uuid`, contactID, inboxID); err != nil {
			t.Fatal(err)
		}
		want = append(want, uuid)
	}
	lo := logf.New(logf.Opts{})
	m := &Manager{db: db, lo: &lo, i18n: testutil.NewI18n(t)}
	for index, contactID := range contacts {
		got, total, err := m.GetPortalConversations(contactID, 1, 10, "", "created_at", "asc")
		if err != nil {
			t.Fatalf("listing contact %d: %v", contactID, err)
		}
		if total != 1 || len(got) != 1 || got[0].UUID != want[index] {
			t.Fatalf("contact %d got conversations %v, total %d; want only %s", contactID, got, total, want[index])
		}
	}
}
