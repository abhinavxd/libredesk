package conversation

import (
	"testing"

	"github.com/jakedolan443/fernmail/internal/conversation/models"
)

func TestMailboxListReturnsErrorWhenDatabaseUnavailable(t *testing.T) {
	m, db, _, _, _ := mailTestManager(t)
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := m.GetConversations(1, 1, nil, []string{models.AllConversations}, "", "", "[]", 1, 20); err == nil {
		t.Fatal("expected a database error")
	}
}
