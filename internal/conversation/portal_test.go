package conversation

import (
	"testing"
	"time"

	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/zerodha/logf"
)

func newPortalTestManager(t *testing.T) *Manager {
	t.Helper()
	db := testutil.NewDB(t, "portal")
	var q queries
	if err := dbutil.ScanSQLFile("queries.sql", &q, db, efs); err != nil {
		t.Fatalf("preparing queries: %v", err)
	}
	lo := logf.New(logf.Opts{})
	return &Manager{q: q, lo: &lo, i18n: testutil.NewI18n(t), db: db}
}

func TestGetContactPortalConversations(t *testing.T) {
	m := newPortalTestManager(t)

	var inboxID, contactID, otherContactID int
	m.db.QueryRow(`INSERT INTO inboxes (name, channel) VALUES ('Support', 'email') RETURNING id`).Scan(&inboxID)
	m.db.QueryRow(`INSERT INTO users (type, first_name, email) VALUES ('contact', 'A', 'a@example.com') RETURNING id`).Scan(&contactID)
	m.db.QueryRow(`INSERT INTO users (type, first_name, email) VALUES ('contact', 'B', 'b@example.com') RETURNING id`).Scan(&otherContactID)

	// Default seeded statuses: Open (open), Snoozed (waiting), Resolved (resolved), Closed (resolved).
	statusID := func(name string) int {
		var id int
		m.db.QueryRow(`SELECT id FROM conversation_statuses WHERE name = $1`, name).Scan(&id)
		return id
	}

	now := time.Now()
	insert := func(contact, status int, subject string, lastMessageAt time.Time, lastSender string) {
		var sender any
		if lastSender != "" {
			sender = lastSender
		}
		if _, err := m.db.Exec(`
			INSERT INTO conversations (contact_id, inbox_id, status_id, subject, last_message, last_message_at, last_message_sender)
			VALUES ($1, $2, $3, $4, 'hi', $5, $6)`,
			contact, inboxID, status, subject, lastMessageAt, sender); err != nil {
			t.Fatalf("inserting conversation: %v", err)
		}
	}

	insert(contactID, statusID("Open"), "open ticket", now.Add(-1*time.Hour), "contact")
	insert(contactID, statusID("Snoozed"), "snoozed ticket", now.Add(-2*time.Hour), "agent")
	insert(contactID, statusID("Resolved"), "resolved ticket", now.Add(-3*time.Hour), "agent")
	insert(contactID, statusID("Closed"), "closed ticket", now.Add(-4*time.Hour), "agent")
	insert(otherContactID, statusID("Open"), "other contact ticket", now, "contact")

	// All conversations of the contact, newest activity first, never another contact's.
	all, total, err := m.GetContactPortalConversations(contactID, "", 1, 10)
	if err != nil {
		t.Fatalf("fetching all: %v", err)
	}
	if total != 4 || len(all) != 4 {
		t.Fatalf("want 4 conversations, got total=%d len=%d", total, len(all))
	}
	if all[0].Subject.String != "open ticket" || all[3].Subject.String != "closed ticket" {
		t.Errorf("wrong order: first=%q last=%q", all[0].Subject.String, all[3].Subject.String)
	}
	for _, c := range all {
		if c.Subject.String == "other contact ticket" {
			t.Fatal("leaked another contact's conversation")
		}
	}

	// Open filter excludes both resolved-category statuses.
	open, total, err := m.GetContactPortalConversations(contactID, "open", 1, 10)
	if err != nil {
		t.Fatalf("fetching open: %v", err)
	}
	if total != 2 || len(open) != 2 {
		t.Fatalf("want 2 open conversations, got total=%d len=%d", total, len(open))
	}

	// Resolved filter covers Resolved and Closed.
	resolved, total, err := m.GetContactPortalConversations(contactID, "resolved", 1, 10)
	if err != nil {
		t.Fatalf("fetching resolved: %v", err)
	}
	if total != 2 || len(resolved) != 2 {
		t.Fatalf("want 2 resolved conversations, got total=%d len=%d", total, len(resolved))
	}
	for _, c := range resolved {
		if c.StatusCategory != "resolved" {
			t.Errorf("resolved filter returned category %q", c.StatusCategory)
		}
	}

	// Pagination: page size 3 splits 4 rows into 3 + 1, total stays 4.
	page1, total, err := m.GetContactPortalConversations(contactID, "", 1, 3)
	if err != nil {
		t.Fatalf("fetching page 1: %v", err)
	}
	if total != 4 || len(page1) != 3 {
		t.Fatalf("page 1: want total=4 len=3, got total=%d len=%d", total, len(page1))
	}
	page2, total, err := m.GetContactPortalConversations(contactID, "", 2, 3)
	if err != nil {
		t.Fatalf("fetching page 2: %v", err)
	}
	if total != 4 || len(page2) != 1 {
		t.Fatalf("page 2: want total=4 len=1, got total=%d len=%d", total, len(page2))
	}

	// A contact with no conversations gets an empty page, not an error.
	none, total, err := m.GetContactPortalConversations(999999, "", 1, 10)
	if err != nil {
		t.Fatalf("fetching for unknown contact: %v", err)
	}
	if total != 0 || len(none) != 0 {
		t.Fatalf("want empty result, got total=%d len=%d", total, len(none))
	}
}
