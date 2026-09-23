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
	query := NormalizeQuery(models.Query{Cursor: "cursor", PageSize: 500})

	if query.PageSize != maxPageSize {
		t.Fatalf("page size = %d, want %d", query.PageSize, maxPageSize)
	}
	if query.Cursor != "cursor" {
		t.Fatalf("cursor = %q, want cursor", query.Cursor)
	}
	if query.Filters != "[]" {
		t.Fatalf("filters = %q, want []", query.Filters)
	}
	if query.Sort != models.SortNewest {
		t.Fatalf("sort = %q, want %q", query.Sort, models.SortNewest)
	}
}

func TestBuildConversationQueryPrioritizesExactReference(t *testing.T) {
	manager := &Manager{filterLocation: func() string { return "UTC" }}
	orderBy, err := conversationResultOrder(models.SortNewest)
	if err != nil {
		t.Fatalf("building result order: %v", err)
	}
	query, args, err := manager.buildQuery(
		"SELECT 1 FROM conversations WHERE $3",
		normalizeQuery(models.Query{Term: "108", PageSize: 500}, maxPageSize),
		models.ReadScope{},
		orderBy,
		nil,
		conversationCursorCondition,
	)
	if err != nil {
		t.Fatalf("building query: %v", err)
	}
	if !strings.Contains(query, "ORDER BY (conversations.reference_number = $1) DESC, conversations.last_message_at DESC NULLS LAST") {
		t.Fatalf("query does not prioritize exact references: %s", query)
	}
	if strings.Contains(query, "COUNT(*) OVER()") || strings.Contains(query, " OFFSET ") {
		t.Fatalf("query performs unbounded pagination work: %s", query)
	}
	if got := args[len(args)-1]; got != maxPageSize+1 {
		t.Fatalf("limit = %v, want %d", got, maxPageSize+1)
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
	if _, err := db.Exec(`UPDATE conversations SET created_at = last_message_at WHERE reference_number = '108' OR reference_number BETWEEN '200' AND '209'`); err != nil {
		t.Fatalf("setting conversation creation times: %v", err)
	}
	if _, err := db.Exec(`UPDATE conversations SET last_message_at = NULL WHERE reference_number IN ('208', '209')`); err != nil {
		t.Fatalf("clearing conversation timestamps: %v", err)
	}
	conversationID, senderID := insertSearchConversation(t, db, "999", "other@example.com", "Needle", "Name", "Needle subject", oldest)
	for i := range 11 {
		if _, err := db.Exec(`
			INSERT INTO conversation_messages (type, status, conversation_id, text_content, sender_id, sender_type, created_at)
			VALUES ('incoming', 'received', $1, $2, $3, 'contact', $4)
		`, conversationID, fmt.Sprintf("ordinary message %d", i), senderID, oldest.Add(time.Duration(i)*time.Minute)); err != nil {
			t.Fatalf("inserting message: %v", err)
		}
	}

	scope := models.ReadScope{Read: true, ReadAll: true}
	results, hasMore, cursor, err := manager.Conversations(models.Query{Term: "108", PageSize: 10}, scope)
	if err != nil {
		t.Fatalf("searching conversations: %v", err)
	}
	if !hasMore || cursor == "" {
		t.Fatalf("first page has_more = %t, cursor = %q", hasMore, cursor)
	}
	if len(results) != 10 {
		t.Fatalf("result count = %d, want 10", len(results))
	}
	if results[0].ReferenceNumber != "108" {
		t.Fatalf("first reference = %q, want 108", results[0].ReferenceNumber)
	}
	newestCursor := cursor

	results, hasMore, nextCursor, err := manager.Conversations(models.Query{Term: "108", Cursor: cursor, PageSize: 10}, scope)
	if err != nil {
		t.Fatalf("searching next conversation page: %v", err)
	}
	if len(results) != 1 || hasMore || nextCursor != "" {
		t.Fatalf("next page returned %d results, has_more %t, cursor %q", len(results), hasMore, nextCursor)
	}

	results, hasMore, cursor, err = manager.Conversations(models.Query{Term: "108", Sort: models.SortOldest, PageSize: 4}, scope)
	if err != nil {
		t.Fatalf("searching oldest conversations: %v", err)
	}
	if len(results) != 4 || !hasMore || cursor == "" {
		t.Fatalf("oldest conversation search returned %d results, has_more %t, cursor %q", len(results), hasMore, cursor)
	}
	if results[0].ReferenceNumber != "108" || results[1].ReferenceNumber != "200" {
		t.Fatalf("oldest conversation references = %q, %q, want 108, 200", results[0].ReferenceNumber, results[1].ReferenceNumber)
	}
	results, hasMore, nextCursor, err = manager.Conversations(models.Query{Term: "108", Sort: models.SortOldest, Cursor: cursor, PageSize: 4}, scope)
	if err != nil {
		t.Fatalf("searching next oldest conversation page: %v", err)
	}
	if len(results) != 4 || !hasMore || nextCursor == "" || results[0].ReferenceNumber != "203" {
		t.Fatalf("next oldest conversation page returned results %+v, has_more %t, cursor %q", results, hasMore, nextCursor)
	}
	results, hasMore, cursor, err = manager.Conversations(models.Query{Term: "108", Sort: models.SortOldest, Cursor: nextCursor, PageSize: 4}, scope)
	if err != nil {
		t.Fatalf("searching final oldest conversation page: %v", err)
	}
	if len(results) != 3 || hasMore || cursor != "" || results[1].ReferenceNumber != "208" || results[2].ReferenceNumber != "209" {
		t.Fatalf("final oldest conversation page returned results %+v, has_more %t, cursor %q", results, hasMore, cursor)
	}
	if _, _, _, err := manager.Conversations(models.Query{Term: "108", Sort: models.SortOldest, Cursor: newestCursor, PageSize: 4}, scope); err == nil {
		t.Fatal("conversation cursor was accepted with a different sort")
	}

	results, hasMore, cursor, err = manager.Conversations(models.Query{Term: "108", Sort: models.SortStartedFirst, PageSize: 4}, scope)
	if err != nil {
		t.Fatalf("searching conversations started first: %v", err)
	}
	if len(results) != 4 || !hasMore || cursor == "" || results[0].ReferenceNumber != "108" || results[1].ReferenceNumber != "200" {
		t.Fatalf("started-first conversation search returned results %+v, has_more %t, cursor %q", results, hasMore, cursor)
	}
	results, _, _, err = manager.Conversations(models.Query{Term: "108", Sort: models.SortStartedFirst, Cursor: cursor, PageSize: 4}, scope)
	if err != nil {
		t.Fatalf("searching next started-first conversation page: %v", err)
	}
	if len(results) != 4 || results[0].ReferenceNumber != "203" {
		t.Fatalf("next started-first conversation page returned results %+v", results)
	}

	results, hasMore, cursor, err = manager.Conversations(models.Query{Term: "108", Sort: models.SortStartedLast, PageSize: 4}, scope)
	if err != nil {
		t.Fatalf("searching conversations started last: %v", err)
	}
	if len(results) != 4 || !hasMore || cursor == "" || results[0].ReferenceNumber != "108" || results[1].ReferenceNumber != "209" {
		t.Fatalf("started-last conversation search returned results %+v, has_more %t, cursor %q", results, hasMore, cursor)
	}
	results, _, _, err = manager.Conversations(models.Query{Term: "108", Sort: models.SortStartedLast, Cursor: cursor, PageSize: 4}, scope)
	if err != nil {
		t.Fatalf("searching next started-last conversation page: %v", err)
	}
	if len(results) != 4 || results[0].ReferenceNumber != "206" {
		t.Fatalf("next started-last conversation page returned results %+v", results)
	}

	results, hasMore, cursor, err = manager.Conversations(models.Query{Term: "Needle", PageSize: 10}, scope)
	if err != nil {
		t.Fatalf("searching unsupported fields: %v", err)
	}
	if len(results) != 0 || hasMore || cursor != "" {
		t.Fatalf("subject or name matched: results %d, has_more %t, cursor %q", len(results), hasMore, cursor)
	}

	messages, hasMore, cursor, err := manager.Messages(models.Query{Term: "ordinary", PageSize: 10}, scope)
	if err != nil {
		t.Fatalf("searching messages: %v", err)
	}
	if len(messages) != 10 || !hasMore || cursor == "" {
		t.Fatalf("message search returned results %+v, has_more %t, cursor %q", messages, hasMore, cursor)
	}
	messages, hasMore, nextCursor, err = manager.Messages(models.Query{Term: "ordinary", Cursor: cursor, PageSize: 10}, scope)
	if err != nil {
		t.Fatalf("searching next message page: %v", err)
	}
	if len(messages) != 1 || hasMore || nextCursor != "" {
		t.Fatalf("next message page returned %d results, has_more %t, cursor %q", len(messages), hasMore, nextCursor)
	}
	messages, hasMore, cursor, err = manager.Messages(models.Query{Term: "ordinary", Sort: models.SortOldest, PageSize: 10}, scope)
	if err != nil {
		t.Fatalf("searching oldest messages: %v", err)
	}
	if len(messages) != 10 || !hasMore || cursor == "" || messages[0].TextContent != "ordinary message 0" {
		t.Fatalf("oldest message search returned results %+v, has_more %t, cursor %q", messages, hasMore, cursor)
	}
	messages, hasMore, nextCursor, err = manager.Messages(models.Query{Term: "ordinary", Sort: models.SortOldest, Cursor: cursor, PageSize: 10}, scope)
	if err != nil {
		t.Fatalf("searching next oldest message page: %v", err)
	}
	if len(messages) != 1 || hasMore || nextCursor != "" || messages[0].TextContent != "ordinary message 10" {
		t.Fatalf("next oldest message page returned results %+v, has_more %t, cursor %q", messages, hasMore, nextCursor)
	}
	if _, _, _, err := manager.Messages(models.Query{Term: "ordinary", Cursor: "invalid", PageSize: 10}, scope); err == nil {
		t.Fatal("invalid message cursor was accepted")
	}
	if _, _, _, err := manager.Messages(models.Query{Term: "ordinary", Sort: models.SortStartedFirst, PageSize: 10}, scope); err == nil {
		t.Fatal("conversation-only sort was accepted for messages")
	}

	for _, term := range []string{"%%%", "___", "%_%"} {
		results, hasMore, cursor, err = manager.Conversations(models.Query{Term: term, PageSize: 10}, scope)
		if err != nil {
			t.Fatalf("searching conversations for %q: %v", term, err)
		}
		if len(results) != 0 || hasMore || cursor != "" {
			t.Fatalf("conversation search for %q returned %d results, has_more %t, cursor %q", term, len(results), hasMore, cursor)
		}

		messages, hasMore, cursor, err = manager.Messages(models.Query{Term: term, PageSize: 10}, scope)
		if err != nil {
			t.Fatalf("searching messages for %q: %v", term, err)
		}
		if len(messages) != 0 || hasMore || cursor != "" {
			t.Fatalf("message search for %q returned %d results, has_more %t, cursor %q", term, len(messages), hasMore, cursor)
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
