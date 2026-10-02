package conversation

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/jakedolan443/fernmail/internal/conversation/models"
	"github.com/jakedolan443/fernmail/internal/dbutil"
	"github.com/jakedolan443/fernmail/internal/testutil"
	"github.com/jmoiron/sqlx"
)

func TestMailQueriesExcludeHistoricalChat(t *testing.T) {
	db := testutil.NewDB(t, "mail_queries")
	db.MustExec(`
 INSERT INTO users(type, email, first_name) VALUES ('contact', 'sender@example.test', 'Sender');
 INSERT INTO inboxes(name, channel) VALUES ('Mail', 'email'), ('Old chat', 'livechat');
 INSERT INTO conversations(contact_id, inbox_id, status_id)
 SELECT (SELECT id FROM users WHERE email = 'sender@example.test'), id, (SELECT id FROM conversation_statuses WHERE name='Open') FROM inboxes;
 `)
	var q struct {
		Get      *sqlx.Stmt `query:"get-conversation"`
		ListItem *sqlx.Stmt `query:"get-conversation-list-item"`
	}
	if err := dbutil.ScanSQLFile("queries.sql", &q, db, efs); err != nil {
		t.Fatal(err)
	}
	for _, channel := range []string{"email", "livechat"} {
		var uuid string
		if err := db.Get(&uuid, `SELECT c.uuid FROM conversations c JOIN inboxes i ON i.id=c.inbox_id WHERE i.channel=$1`, channel); err != nil {
			t.Fatal(err)
		}
		var conv models.Conversation
		err := q.Get.Get(&conv, 0, uuid, "")
		var item models.ConversationListItem
		itemErr := q.ListItem.Get(&item, uuid)
		if channel == "email" {
			if err != nil || itemErr != nil {
				t.Fatalf("email inaccessible: %v, %v", err, itemErr)
			}
		} else if !errors.Is(err, sql.ErrNoRows) || !errors.Is(itemErr, sql.ErrNoRows) {
			t.Fatalf("chat accessible: %v, %v", err, itemErr)
		}
	}
}
