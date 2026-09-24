package migrations

import (
	"slices"
	"testing"

	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

func TestV2_9_0TemplateComponentsMigration(t *testing.T) {
	db := testutil.NewDB(t, "migration_v2_9_0_templates")
	query := `SELECT data_type || ':' || is_nullable || ':' || COALESCE(column_default, '') FROM information_schema.columns WHERE table_name = 'whatsapp_templates' AND column_name = 'component_types'`
	var expected string
	if err := db.Get(&expected, query); err != nil {
		t.Fatal(err)
	}
	db.MustExec(`ALTER TABLE whatsapp_templates DROP COLUMN component_types`)
	for range 2 {
		if err := V2_9_0(db, nil, nil); err != nil {
			t.Fatal(err)
		}
	}
	var actual string
	if err := db.Get(&actual, query); err != nil {
		t.Fatal(err)
	}
	if actual != expected {
		t.Fatalf("migration column = %q, schema column = %q", actual, expected)
	}
}

func TestV2_9_0PrivateNotePermissionMigration(t *testing.T) {
	db := testutil.NewDB(t, "migration_v2_9_0")

	roles := []struct {
		name        string
		permissions pq.StringArray
		wantPrivate bool
	}{
		{"With old permission", pq.StringArray{"conversations:read", "messages:write"}, true},
		{"Without old permission", pq.StringArray{"conversations:read", "messages:read"}, false},
		{"Already migrated", pq.StringArray{"messages:write", "messages:write_private"}, true},
	}
	for _, role := range roles {
		if _, err := db.Exec(`INSERT INTO roles (name, description, permissions) VALUES ($1, '', $2)`, role.name, role.permissions); err != nil {
			t.Fatalf("inserting role %q: %v", role.name, err)
		}
	}

	// Running the migration twice verifies that it does not append duplicates.
	for range 2 {
		if err := V2_9_0(db, nil, nil); err != nil {
			t.Fatalf("running migration: %v", err)
		}
	}

	for _, role := range roles {
		var got pq.StringArray
		if err := db.Get(&got, `SELECT permissions FROM roles WHERE name = $1`, role.name); err != nil {
			t.Fatalf("reading role %q: %v", role.name, err)
		}
		count := 0
		for _, permission := range got {
			if permission == "messages:write_private" {
				count++
			}
		}
		if slices.Contains(got, "messages:write_private") != role.wantPrivate {
			t.Errorf("role %q permissions = %v, want private permission = %v", role.name, got, role.wantPrivate)
		}
		if count > 1 {
			t.Errorf("role %q has duplicate private permissions: %v", role.name, got)
		}
	}
}

func TestV2_9_0NotificationMigration(t *testing.T) {
	db := testutil.NewDB(t, "migration_v2_9_0_notifications")
	db.MustExec(`
		DROP TABLE notification_email_queue;
		DROP TABLE notification_push_subscriptions;
		DROP TABLE user_notification_preferences;
		DROP TYPE notification_channel;
		DELETE FROM settings WHERE "key" IN (
			'notification.push.vapid_public_key',
			'notification.push.vapid_private_key'
		);
		DELETE FROM templates WHERE "name" IN (
			'New reply from contact',
			'New reply on participating conversation',
			'Conversation reopened'
		);
	`)

	for range 2 {
		if err := V2_9_0(db, nil, nil); err != nil {
			t.Fatalf("running migration: %v", err)
		}
	}

	for _, table := range []string{
		"user_notification_preferences",
		"notification_push_subscriptions",
		"notification_email_queue",
	} {
		var exists bool
		if err := db.Get(&exists, `SELECT to_regclass($1) IS NOT NULL`, table); err != nil {
			t.Fatalf("checking table %q: %v", table, err)
		}
		if !exists {
			t.Errorf("table %q was not created", table)
		}
	}

	var pushChannel bool
	if err := db.Get(&pushChannel, `
		SELECT EXISTS (
			SELECT 1 FROM pg_enum e
			JOIN pg_type t ON t.oid = e.enumtypid
			WHERE t.typname = 'notification_channel' AND e.enumlabel = 'push'
		)
	`); err != nil {
		t.Fatalf("checking push channel: %v", err)
	}
	if !pushChannel {
		t.Error("push notification channel was not created")
	}

	var settingCount, templateCount int
	if err := db.Get(&settingCount, `SELECT COUNT(*) FROM settings WHERE "key" LIKE 'notification.push.vapid_%'`); err != nil {
		t.Fatalf("counting VAPID settings: %v", err)
	}
	if settingCount != 2 {
		t.Errorf("VAPID setting count = %d, want 2", settingCount)
	}
	if err := db.Get(&templateCount, `
		SELECT COUNT(*) FROM templates WHERE "name" IN (
			'New reply from contact',
			'New reply on participating conversation',
			'Conversation reopened'
		)
	`); err != nil {
		t.Fatalf("counting notification templates: %v", err)
	}
	if templateCount != 3 {
		t.Errorf("notification template count = %d, want 3", templateCount)
	}
}

func TestV2_9_0PreservesExistingQueuedEmails(t *testing.T) {
	db := testutil.NewDB(t, "migration_v2_9_0_queue")
	columnQuery := `SELECT data_type || ':' || is_nullable || ':' || COALESCE(column_default, '') FROM information_schema.columns WHERE table_name = 'notification_email_queue' AND column_name = $1`
	var expectedAttemptsColumn string
	if err := db.Get(&expectedAttemptsColumn, columnQuery, "attempts"); err != nil {
		t.Fatal(err)
	}
	db.MustExec(`ALTER TABLE notification_email_queue DROP COLUMN attempts`)
	db.MustExec(`INSERT INTO users (type, email, first_name, last_name) VALUES ('agent', 'queued@example.com', 'Agent', '')`)
	db.MustExec(`INSERT INTO notification_email_queue (user_id, notification_type, recipient_email, subject, content, send_at) VALUES ((SELECT id FROM users LIMIT 1), 'new_reply', 'queued@example.com', 'Reply', 'Pending reply', now())`)
	for range 2 {
		if err := V2_9_0(db, nil, nil); err != nil {
			t.Fatal(err)
		}
	}
	var actualAttemptsColumn string
	if err := db.Get(&actualAttemptsColumn, columnQuery, "attempts"); err != nil {
		t.Fatal(err)
	}
	if actualAttemptsColumn != expectedAttemptsColumn {
		t.Fatalf("attempts column = %q, schema column = %q", actualAttemptsColumn, expectedAttemptsColumn)
	}
	var preserved bool
	if err := db.Get(&preserved, `SELECT content = 'Pending reply' AND attempts = 0 FROM notification_email_queue`); err != nil {
		t.Fatal(err)
	}
	if !preserved {
		t.Fatal("existing queued email changed")
	}
}

func TestWidgetCampaignMigration(t *testing.T) {
	db := testutil.NewDB(t, "widget_migration")
	db.MustExec(`DROP TABLE widget_campaign_deliveries`)
	for range 2 {
		if err := V2_9_0(db, nil, nil); err != nil {
			t.Fatal(err)
		}
	}
	var count int
	if err := db.Get(&count, `SELECT COUNT(*) FROM pg_indexes WHERE tablename = 'widget_campaign_deliveries'`); err != nil {
		t.Fatal(err)
	}
	if count != 4 {
		t.Fatalf("expected primary key and three indexes, got %d", count)
	}
}

func TestHelpArticleTranslationGroupMigration(t *testing.T) {
	db := testutil.NewDB(t, "article_translation_group_migration")
	db.MustExec(`
		INSERT INTO help_centers (name, slug, allowed_locales) VALUES ('Docs', 'docs', '["en", "fr"]');
		INSERT INTO article_collections (help_center_id, slug, locale, name) VALUES
			(1, 'general', 'en', 'General'),
			(1, 'general', 'fr', 'Général');
		INSERT INTO help_articles (collection_id, slug, locale, title) VALUES
			(1, 'billing', 'en', 'Billing'),
			(2, 'billing', 'fr', 'Facturation');
		ALTER TABLE help_articles DROP COLUMN translation_group_id;
	`)

	for range 2 {
		if err := V2_9_0(db, nil, nil); err != nil {
			t.Fatal(err)
		}
	}

	var groups int
	if err := db.Get(&groups, `SELECT COUNT(DISTINCT translation_group_id) FROM help_articles`); err != nil {
		t.Fatal(err)
	}
	if groups != 2 {
		t.Fatalf("expected every existing article in its own group, got %d groups", groups)
	}
}

func TestBackfillLastResolvedAtIsBatchedAndResumable(t *testing.T) {
	db := testutil.NewDB(t, "migrations")
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

	again, err := backfillLastResolvedAt(db, 2)
	if err != nil {
		t.Fatalf("second backfill: %v", err)
	}
	if again != 0 {
		t.Fatalf("expected the repeat run to touch nothing, it ran %d batches", again)
	}
}

func TestCreateIndexConcurrentlyReplacesAnInvalidIndex(t *testing.T) {
	db := testutil.NewDB(t, "migrations")
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

	for i := range n {
		if _, err := db.Exec(`INSERT INTO conversations (contact_id, inbox_id, status_id, resolved_at, last_resolved_at)
			VALUES ($1, $2, $3, NOW() - ($4 || ' hours')::interval, NULL)`, contactID, inboxID, statusID, i+1); err != nil {
			t.Fatalf("seeding a conversation: %v", err)
		}
	}
}

func indexIsValid(t *testing.T, db *sqlx.DB, name string) bool {
	t.Helper()
	var valid bool
	if err := db.Get(&valid, `SELECT indisvalid FROM pg_index WHERE indexrelid = $1::regclass`, name); err != nil {
		t.Fatalf("reading index validity: %v", err)
	}
	return valid
}
