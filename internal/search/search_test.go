package search

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/abhinavxd/libredesk/internal/search/models"
	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/jmoiron/sqlx"
	"github.com/zerodha/logf"
)

func TestNormalizeQuery(t *testing.T) {
	query := NormalizeQuery(models.Query{Page: 0, PageSize: 500})

	if query.Page != 1 {
		t.Fatalf("page = %d, want 1", query.Page)
	}
	if query.PageSize != maxPageSize {
		t.Fatalf("page size = %d, want %d", query.PageSize, maxPageSize)
	}
	if query.Filters != "[]" {
		t.Fatalf("filters = %q, want []", query.Filters)
	}
}

func TestBuildConversationQueryPrioritizesExactReference(t *testing.T) {
	manager := &Manager{filterLocation: func() string { return "UTC" }}
	query, args, err := manager.buildQuery(
		"SELECT 1 FROM conversations WHERE $3",
		models.Query{Term: "108", Page: 2, PageSize: 500},
		models.ReadScope{},
		conversationResultOrder,
		nil,
		maxPageSize,
	)
	if err != nil {
		t.Fatalf("building query: %v", err)
	}
	if !strings.Contains(query, "ORDER BY (conversations.reference_number = $1) DESC, conversations.last_message_at DESC NULLS LAST") {
		t.Fatalf("query does not prioritize exact references: %s", query)
	}
	if got := args[len(args)-2]; got != maxPageSize {
		t.Fatalf("limit = %v, want %d", got, maxPageSize)
	}
	if got := args[len(args)-1]; got != maxPageSize {
		t.Fatalf("offset = %v, want %d", got, maxPageSize)
	}
}

func TestFirstPageSearchKeepsLegacyLimits(t *testing.T) {
	conversationQuery := normalizeQuery(models.Query{PageSize: 500}, maxConversationFirstPageSize)
	if conversationQuery.PageSize != 500 {
		t.Fatalf("conversation page size = %d, want 500", conversationQuery.PageSize)
	}

	messageQuery := normalizeQuery(models.Query{PageSize: 100}, maxMessageFirstPageSize)
	if messageQuery.PageSize != maxMessageFirstPageSize {
		t.Fatalf("message page size = %d, want %d", messageQuery.PageSize, maxMessageFirstPageSize)
	}
}

func TestConversationSearchFieldsAndRanking(t *testing.T) {
	db := testutil.NewDB(t, "search_conversations")
	lo := logf.New(logf.Opts{})
	manager, err := New(Opts{
		DB:             db,
		Lo:             &lo,
		I18n:           testutil.NewI18n(t),
		FilterLocation: func() string { return "UTC" },
	})
	if err != nil {
		t.Fatalf("creating search manager: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO inboxes (name, channel) VALUES ('Search test', 'email')`); err != nil {
		t.Fatalf("inserting inbox: %v", err)
	}

	oldest := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)
	insertSearchConversation(t, db, "108", "exact-108@example.com", "Exact", "Contact", "Old subject", oldest)
	for i := range 10 {
		insertSearchConversation(
			t,
			db,
			fmt.Sprintf("2%02d", i),
			fmt.Sprintf("new-108-%02d@example.com", i),
			"Recent",
			"Contact",
			"Recent subject",
			oldest.Add(time.Duration(i+1)*time.Hour),
		)
	}
	conversationID, senderID := insertSearchConversation(t, db, "999", "other@example.com", "Needle", "Name", "Needle subject", oldest)
	if _, err := db.Exec(`
		INSERT INTO conversation_messages (type, status, conversation_id, text_content, sender_id, sender_type)
		VALUES ('incoming', 'received', $1, 'ordinary message', $2, 'contact')
	`, conversationID, senderID); err != nil {
		t.Fatalf("inserting message: %v", err)
	}

	scope := models.ReadScope{Read: true, ReadAll: true}
	results, total, err := manager.Conversations(models.Query{Term: "108", Page: 1, PageSize: 10}, scope)
	if err != nil {
		t.Fatalf("searching conversations: %v", err)
	}
	if total != 11 {
		t.Fatalf("total = %d, want 11", total)
	}
	if len(results) != 10 {
		t.Fatalf("result count = %d, want 10", len(results))
	}
	if results[0].ReferenceNumber != "108" {
		t.Fatalf("first reference = %q, want 108", results[0].ReferenceNumber)
	}

	results, total, err = manager.Conversations(models.Query{Term: "Needle", Page: 1, PageSize: 10}, scope)
	if err != nil {
		t.Fatalf("searching unsupported fields: %v", err)
	}
	if total != 0 || len(results) != 0 {
		t.Fatalf("subject or name matched: total %d, results %d", total, len(results))
	}

	messages, messageTotal, err := manager.Messages(models.Query{Term: "ordinary", Page: 1, PageSize: 10}, scope)
	if err != nil {
		t.Fatalf("searching messages: %v", err)
	}
	if messageTotal != 1 || len(messages) != 1 || messages[0].TextContent != "ordinary message" {
		t.Fatalf("message search returned total %d, results %+v", messageTotal, messages)
	}

	for _, term := range []string{"%%%", "___", "%_%"} {
		results, total, err = manager.Conversations(models.Query{Term: term, Page: 1, PageSize: 10}, scope)
		if err != nil {
			t.Fatalf("searching conversations for %q: %v", term, err)
		}
		if total != 0 || len(results) != 0 {
			t.Fatalf("conversation search for %q returned total %d, results %d", term, total, len(results))
		}

		messages, messageTotal, err = manager.Messages(models.Query{Term: term, Page: 1, PageSize: 10}, scope)
		if err != nil {
			t.Fatalf("searching messages for %q: %v", term, err)
		}
		if messageTotal != 0 || len(messages) != 0 {
			t.Fatalf("message search for %q returned total %d, results %d", term, messageTotal, len(messages))
		}

		contacts, err := manager.Contacts(term, 10)
		if err != nil {
			t.Fatalf("searching contacts for %q: %v", term, err)
		}
		if len(contacts) != 0 {
			t.Fatalf("contact search for %q returned %d results", term, len(contacts))
		}
	}
}

func insertSearchConversation(t *testing.T, db *sqlx.DB, reference, email, firstName, lastName, subject string, lastMessageAt time.Time) (int, int) {
	t.Helper()

	var contactID int
	if err := db.Get(&contactID, `
		INSERT INTO users (type, email, first_name, last_name)
		VALUES ('contact', $1, $2, $3)
		RETURNING id
	`, email, firstName, lastName); err != nil {
		t.Fatalf("inserting contact: %v", err)
	}

	var conversationID int
	if err := db.Get(&conversationID, `
		INSERT INTO conversations (contact_id, inbox_id, status_id, reference_number, subject, last_message_at)
		VALUES (
			$1,
			(SELECT id FROM inboxes LIMIT 1),
			(SELECT id FROM conversation_statuses WHERE name = 'Open'),
			$2,
			$3,
			$4
		)
		RETURNING id
	`, contactID, reference, subject, lastMessageAt); err != nil {
		t.Fatalf("inserting conversation: %v", err)
	}
	return conversationID, contactID
}
