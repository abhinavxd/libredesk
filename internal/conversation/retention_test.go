package conversation

import (
	"context"
	"testing"

	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/zerodha/logf"
)

func TestDeleteExpiredConversations(t *testing.T) {
	db := testutil.NewDB(t, "retention")
	var contactID, inboxID int
	if err := db.Get(&contactID, `INSERT INTO users (type, email, first_name, last_name) VALUES ('contact', 'customer@example.com', 'Customer', '') RETURNING id`); err != nil {
		t.Fatal(err)
	}
	if err := db.Get(&inboxID, `INSERT INTO inboxes (name, channel) VALUES ('Mail', 'email') RETURNING id`); err != nil {
		t.Fatal(err)
	}
	// insert creates a conversation with one message, in the given status, with each timestamp an interval back from now.
	insert := func(status, createdAgo, lastMessageAgo, lastResolvedAgo string) int {
		var id int
		if err := db.Get(&id, `INSERT INTO conversations (contact_id, inbox_id, status_id, created_at, last_message_at, last_resolved_at)
			VALUES ($1, $2, (SELECT id FROM conversation_statuses WHERE name = $3), NOW() - $4::interval, NOW() - $5::interval, NOW() - $6::interval) RETURNING id`,
			contactID, inboxID, status, createdAgo, lastMessageAgo, lastResolvedAgo); err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(`INSERT INTO conversation_messages (type, status, conversation_id, sender_id, sender_type, content) VALUES ('incoming', 'received', $1, $2, 'contact', 'Hello')`, id, contactID); err != nil {
			t.Fatal(err)
		}
		return id
	}
	expiredClosed := insert("Closed", "90 days", "60 days", "45 days")
	expiredResolved := insert("Resolved", "90 days", "60 days", "45 days")
	recentlyResolved := insert("Resolved", "90 days", "60 days", "10 days")
	newMessageSince := insert("Closed", "90 days", "5 days", "45 days")
	openOld := insert("Open", "90 days", "60 days", "60 days")

	lo := logf.New(logf.Opts{})
	m := &Manager{lo: &lo}
	if err := dbutil.ScanSQLFile("queries.sql", &m.q, db, efs); err != nil {
		t.Fatal(err)
	}

	deleted, err := m.DeleteExpiredConversations(context.Background(), 30)
	if err != nil {
		t.Fatal(err)
	}
	if deleted != 2 {
		t.Fatalf("deleted = %d, want 2", deleted)
	}
	for id, want := range map[int]bool{expiredClosed: false, expiredResolved: false, recentlyResolved: true, newMessageSince: true, openOld: true} {
		var exists bool
		if err := db.Get(&exists, `SELECT EXISTS (SELECT 1 FROM conversations WHERE id = $1)`, id); err != nil {
			t.Fatal(err)
		}
		if exists != want {
			t.Errorf("conversation %d exists = %v, want %v", id, exists, want)
		}
	}
	var messages int
	if err := db.Get(&messages, `SELECT COUNT(*) FROM conversation_messages WHERE conversation_id IN ($1, $2)`, expiredClosed, expiredResolved); err != nil {
		t.Fatal(err)
	}
	if messages != 0 {
		t.Fatalf("messages of deleted conversations = %d, want 0", messages)
	}

	// A second run finds nothing left.
	if deleted, err := m.DeleteExpiredConversations(context.Background(), 30); err != nil || deleted != 0 {
		t.Fatalf("second run = %d, %v; want 0, nil", deleted, err)
	}
}
