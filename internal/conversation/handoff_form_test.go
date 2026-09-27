package conversation

import (
	"testing"

	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/zerodha/logf"
)

func TestClearHandoffFormPending(t *testing.T) {
	db := testutil.NewDB(t, "handoff_form_pending")
	var contactID, inboxID, conversationID int
	if err := db.Get(&contactID, `INSERT INTO users (type, email, first_name, last_name) VALUES ('contact', 'customer@example.com', 'Customer', '') RETURNING id`); err != nil {
		t.Fatal(err)
	}
	if err := db.Get(&inboxID, `INSERT INTO inboxes (name, channel) VALUES ('Chat', 'livechat') RETURNING id`); err != nil {
		t.Fatal(err)
	}
	if err := db.Get(&conversationID, `INSERT INTO conversations (contact_id, inbox_id, status_id) VALUES ($1, $2, (SELECT id FROM conversation_statuses LIMIT 1)) RETURNING id`, contactID, inboxID); err != nil {
		t.Fatal(err)
	}
	insertMessage := func(meta string) string {
		var uuid string
		if err := db.Get(&uuid, `INSERT INTO conversation_messages (type, status, conversation_id, sender_id, sender_type, content, meta) VALUES ('outgoing', 'sent', $1, $2, 'agent', 'Please fill the form', $3) RETURNING uuid`, conversationID, contactID, meta); err != nil {
			t.Fatal(err)
		}
		return uuid
	}

	lo := logf.New(logf.Opts{})
	m := &Manager{lo: &lo}
	if err := dbutil.ScanSQLFile("queries.sql", &m.q, db, efs); err != nil {
		t.Fatal(err)
	}

	pending := insertMessage(`{"handoff_form_pending": true, "ai_assistant_id": 7}`)
	cleared, err := m.ClearHandoffFormPending(pending)
	if err != nil || !cleared {
		t.Fatalf("first clear = %v, %v; want true, nil", cleared, err)
	}
	var flag, assistantID string
	if err := db.QueryRow(`SELECT meta->>'handoff_form_pending', meta->>'ai_assistant_id' FROM conversation_messages WHERE uuid = $1`, pending).Scan(&flag, &assistantID); err != nil {
		t.Fatal(err)
	}
	if flag != "false" || assistantID != "7" {
		t.Fatalf("meta after clear = pending %q, assistant %q; want false, 7", flag, assistantID)
	}

	if cleared, err := m.ClearHandoffFormPending(pending); err != nil || cleared {
		t.Fatalf("second clear = %v, %v; want false, nil", cleared, err)
	}
	for _, meta := range []string{`{}`, `{"handoff_form_pending": false}`} {
		if cleared, err := m.ClearHandoffFormPending(insertMessage(meta)); err != nil || cleared {
			t.Fatalf("clear with meta %s = %v, %v; want false, nil", meta, cleared, err)
		}
	}
}
