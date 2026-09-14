package migrations

import (
	"slices"
	"testing"

	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/lib/pq"
)

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
	var expectedMessageTimeColumn, expectedAttemptsColumn string
	if err := db.Get(&expectedMessageTimeColumn, columnQuery, "message_created_at"); err != nil {
		t.Fatal(err)
	}
	if err := db.Get(&expectedAttemptsColumn, columnQuery, "attempts"); err != nil {
		t.Fatal(err)
	}
	db.MustExec(`ALTER TABLE notification_email_queue DROP COLUMN message_created_at, DROP COLUMN attempts`)
	db.MustExec(`INSERT INTO users (type, email, first_name, last_name) VALUES ('agent', 'queued@example.com', 'Agent', '')`)
	db.MustExec(`INSERT INTO notification_email_queue (user_id, notification_type, recipient_email, subject, content, send_at) VALUES ((SELECT id FROM users LIMIT 1), 'new_reply', 'queued@example.com', 'Reply', 'Pending reply', now())`)
	for range 2 {
		if err := V2_9_0(db, nil, nil); err != nil {
			t.Fatal(err)
		}
	}
	var actualMessageTimeColumn, actualAttemptsColumn string
	if err := db.Get(&actualMessageTimeColumn, columnQuery, "message_created_at"); err != nil {
		t.Fatal(err)
	}
	if err := db.Get(&actualAttemptsColumn, columnQuery, "attempts"); err != nil {
		t.Fatal(err)
	}
	if actualMessageTimeColumn != expectedMessageTimeColumn {
		t.Fatalf("message time column = %q, schema column = %q", actualMessageTimeColumn, expectedMessageTimeColumn)
	}
	if actualAttemptsColumn != expectedAttemptsColumn {
		t.Fatalf("attempts column = %q, schema column = %q", actualAttemptsColumn, expectedAttemptsColumn)
	}
	var preserved bool
	if err := db.Get(&preserved, `SELECT content = 'Pending reply' AND message_created_at IS NULL AND attempts = 0 FROM notification_email_queue`); err != nil {
		t.Fatal(err)
	}
	if !preserved {
		t.Fatal("existing queued email changed")
	}
}
