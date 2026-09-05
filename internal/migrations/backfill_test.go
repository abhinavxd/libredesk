package migrations

import (
	"testing"

	"github.com/abhinavxd/libredesk/internal/testdb"
	"github.com/jmoiron/sqlx"
)

func TestBackfillLastResolvedAtIsBatchedAndResumable(t *testing.T) {
	db := testdb.New(t, "migrations")
	seedResolvedConversations(t, db, 5)

	batches, err := backfillLastResolvedAt(db, 2)
	if err != nil {
		t.Fatalf("backfill: %v", err)
	}
	if batches < 3 {
		t.Fatalf("5 rows at a batch size of 2 must take at least 3 batches, took %d", batches)
	}

	var pending int
	if err := db.Get(&pending, `SELECT count(*) FROM conversations WHERE resolved_at IS NOT NULL AND last_resolved_at IS NULL`); err != nil {
		t.Fatalf("counting: %v", err)
	}
	if pending != 0 {
		t.Fatalf("%d rows left unbackfilled", pending)
	}
	var mismatched int
	if err := db.Get(&mismatched, `SELECT count(*) FROM conversations WHERE resolved_at IS NOT NULL AND last_resolved_at IS DISTINCT FROM resolved_at`); err != nil {
		t.Fatalf("counting: %v", err)
	}
	if mismatched != 0 {
		t.Fatalf("%d rows carry the wrong last_resolved_at", mismatched)
	}

	// Re-running after an interrupted upgrade must be a no-op rather than a second full rewrite.
	again, err := backfillLastResolvedAt(db, 2)
	if err != nil {
		t.Fatalf("second backfill: %v", err)
	}
	if again != 0 {
		t.Fatalf("expected the repeat run to touch nothing, it ran %d batches", again)
	}
}

func seedResolvedConversations(t *testing.T, db *sqlx.DB, n int) {
	t.Helper()
	if _, err := db.Exec(`DELETE FROM conversations`); err != nil {
		t.Fatalf("clearing conversations: %v", err)
	}
	if _, err := db.Exec(`DELETE FROM users WHERE email = 'backfill@example.com'`); err != nil {
		t.Fatalf("clearing the contact: %v", err)
	}

	var contactID int
	if err := db.Get(&contactID, `INSERT INTO users (type, email, first_name) VALUES ('contact', 'backfill@example.com', 'Back') RETURNING id`); err != nil {
		t.Fatalf("seeding a contact: %v", err)
	}
	var inboxID int
	if err := db.Get(&inboxID, `INSERT INTO inboxes (channel, config, "name", enabled, "from") VALUES ('email', '{}'::jsonb, 'backfill-inbox', true, '') RETURNING id`); err != nil {
		t.Fatalf("seeding an inbox: %v", err)
	}
	var statusID int
	if err := db.Get(&statusID, `SELECT id FROM conversation_statuses ORDER BY id LIMIT 1`); err != nil {
		t.Fatalf("reading a status: %v", err)
	}

	for i := 0; i < n; i++ {
		if _, err := db.Exec(`INSERT INTO conversations (contact_id, inbox_id, status_id, resolved_at, last_resolved_at)
			VALUES ($1, $2, $3, NOW() - ($4 || ' hours')::interval, NULL)`, contactID, inboxID, statusID, i+1); err != nil {
			t.Fatalf("seeding a conversation: %v", err)
		}
	}
}

func TestCreateIndexConcurrentlyReplacesAnInvalidIndex(t *testing.T) {
	db := testdb.New(t, "migrations")
	const name = "index_test_concurrent_build"
	ddl := `CREATE INDEX CONCURRENTLY IF NOT EXISTS ` + name + ` ON conversations (last_inbound_at)`

	if _, err := db.Exec(`DROP INDEX IF EXISTS ` + name); err != nil {
		t.Fatalf("clearing: %v", err)
	}
	if err := createIndexConcurrently(db, name, ddl); err != nil {
		t.Fatalf("first build: %v", err)
	}
	if !indexIsValid(t, db, name) {
		t.Fatal("expected a valid index after the first build")
	}

	if err := createIndexConcurrently(db, name, ddl); err != nil {
		t.Fatalf("repeat build: %v", err)
	}

	// Leftover of an interrupted concurrent build.
	if _, err := db.Exec(`UPDATE pg_index SET indisvalid = false WHERE indexrelid = $1::regclass`, name); err != nil {
		t.Skipf("cannot mark an index invalid on this server: %v", err)
	}
	if indexIsValid(t, db, name) {
		t.Fatal("expected the index to be marked invalid for the test")
	}
	if err := createIndexConcurrently(db, name, ddl); err != nil {
		t.Fatalf("rebuild after an interrupted build: %v", err)
	}
	if !indexIsValid(t, db, name) {
		t.Fatal("an invalid index survived the rebuild, so the index silently never works")
	}
	db.Exec(`DROP INDEX IF EXISTS ` + name)
}

func indexIsValid(t *testing.T, db *sqlx.DB, name string) bool {
	t.Helper()
	var valid bool
	if err := db.Get(&valid, `SELECT indisvalid FROM pg_index WHERE indexrelid = $1::regclass`, name); err != nil {
		t.Fatalf("reading index validity: %v", err)
	}
	return valid
}
