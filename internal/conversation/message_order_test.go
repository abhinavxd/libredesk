package conversation

import (
	"testing"

	"github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

func TestGetOutgoingPendingMessagesSerializesConversations(t *testing.T) {
	db := testutil.NewDB(t, "conversation")
	var q queries
	if err := dbutil.ScanSQLFile("queries.sql", &q, db, efs); err != nil {
		t.Fatalf("preparing conversation queries: %v", err)
	}

	conversationA, conversationB, senderID := seedOutgoingOrderTest(t, db)
	for _, row := range []struct {
		kind, status string
		private      bool
	}{
		{"incoming", "pending", false},
		{"activity", "pending", false},
		{"outgoing", "sent", false},
		{"outgoing", "failed", false},
		{"outgoing", "pending", true},
	} {
		if _, err := db.Exec(`INSERT INTO conversation_messages (conversation_id, sender_id, sender_type, type, status, private)
			VALUES ($1, $2, 'agent', $3, $4, $5)`, conversationA, senderID, row.kind, row.status, row.private); err != nil {
			t.Fatal(err)
		}
	}
	firstA := insertPendingMessage(t, db, conversationA, senderID, "a1")
	secondA := insertPendingMessage(t, db, conversationA, senderID, "a2")
	firstB := insertPendingMessage(t, db, conversationB, senderID, "b1")
	insertPendingMessage(t, db, conversationB, senderID, "b2")
	if _, err := db.Exec(`UPDATE conversation_messages SET created_at = NOW() + INTERVAL '1 day' WHERE id = $1`, firstA); err != nil {
		t.Fatal(err)
	}

	assertPendingMessageIDs(t, q.GetOutgoingPendingMessages, nil, []int{firstA, firstB})
	assertPendingMessageIDs(t, q.GetOutgoingPendingMessages, []int{conversationA}, []int{firstB})
	assertPendingMessageIDs(t, q.GetOutgoingPendingMessages, []int{conversationA, conversationA, 0, -1}, []int{firstB})
	assertPendingMessageIDs(t, q.GetOutgoingPendingMessages, []int{0, -1}, []int{firstA, firstB})
	assertPendingMessageIDs(t, q.GetOutgoingPendingMessages, []int{conversationA, conversationB}, nil /** want **/)

	if _, err := db.Exec(`UPDATE conversation_messages SET status = 'sent' WHERE id = $1`, firstA); err != nil {
		t.Fatalf("marking first message sent: %v", err)
	}
	assertPendingMessageIDs(t, q.GetOutgoingPendingMessages, nil, []int{secondA, firstB})
	if _, err := db.Exec(`UPDATE conversation_messages SET status = 'sent' WHERE type = 'outgoing' AND private = false`); err != nil {
		t.Fatal(err)
	}
	assertPendingMessageIDs(t, q.GetOutgoingPendingMessages, nil /** processing **/, nil /** want **/)
}

func seedOutgoingOrderTest(t *testing.T, db *sqlx.DB) (int, int, int) {
	t.Helper()
	var contactID, senderID, inboxID, statusID int
	if err := db.QueryRow(`INSERT INTO users (type, email, first_name) VALUES ('contact', 'outgoing-order-contact@example.test', 'Contact') RETURNING id`).Scan(&contactID); err != nil {
		t.Fatalf("inserting contact: %v", err)
	}
	if err := db.QueryRow(`INSERT INTO users (type, email, first_name) VALUES ('agent', 'outgoing-order-agent@example.test', 'Agent') RETURNING id`).Scan(&senderID); err != nil {
		t.Fatalf("inserting sender: %v", err)
	}
	if err := db.QueryRow(`INSERT INTO inboxes (channel, name) VALUES ('whatsapp', 'outgoing-order') RETURNING id`).Scan(&inboxID); err != nil {
		t.Fatalf("inserting inbox: %v", err)
	}
	if err := db.QueryRow(`SELECT id FROM conversation_statuses WHERE category = 'open' LIMIT 1`).Scan(&statusID); err != nil {
		t.Fatalf("selecting status: %v", err)
	}

	insertConversation := func(subject string) int {
		t.Helper()
		var id int
		if err := db.QueryRow(`INSERT INTO conversations (contact_id, inbox_id, status_id, subject) VALUES ($1, $2, $3, $4) RETURNING id`, contactID, inboxID, statusID, subject).Scan(&id); err != nil {
			t.Fatalf("inserting conversation: %v", err)
		}
		return id
	}
	return insertConversation("a"), insertConversation("b"), senderID
}

func insertPendingMessage(t *testing.T, db *sqlx.DB, conversationID, senderID int, content string) int {
	t.Helper()
	var id int
	if err := db.QueryRow(`
		INSERT INTO conversation_messages ("type", status, conversation_id, content, text_content, content_type, sender_id, sender_type, private, meta)
		VALUES ('outgoing', 'pending', $1, $2, $2, 'text', $3, 'agent', false, '{}')
		RETURNING id`, conversationID, content, senderID).Scan(&id); err != nil {
		t.Fatalf("inserting message: %v", err)
	}
	return id
}

func assertPendingMessageIDs(t *testing.T, stmt *sqlx.Stmt, processing, want []int) {
	t.Helper()
	if processing == nil {
		processing = []int{}
	}
	var messages []models.Message
	if err := stmt.Select(&messages, pq.Array(processing)); err != nil {
		t.Fatalf("selecting pending messages: %v", err)
	}
	if len(messages) != len(want) {
		t.Fatalf("got %d messages, want %d", len(messages), len(want))
	}
	for i, message := range messages {
		if message.ID != want[i] {
			t.Fatalf("message %d has id %d, want %d", i, message.ID, want[i])
		}
	}
}
