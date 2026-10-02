package migrations

import (
	"github.com/jakedolan443/fernmail/internal/testutil"
	"testing"
)

func TestRemoveNotificationsPreservesAccountMailAndMailbox(t *testing.T) {
	db := testutil.NewDB(t, "remove_notifications")
	db.MustExec(`
 CREATE TYPE user_notification_type AS ENUM ('mention');
 CREATE TYPE notification_channel AS ENUM ('email');
 CREATE TABLE user_notifications(id int);
 CREATE TABLE user_notification_preferences(id int);
 CREATE TABLE notification_email_queue(id int);
 CREATE TABLE notification_push_subscriptions(id int);
 DELETE FROM settings WHERE key = 'account_email.host';
 INSERT INTO settings(key,value) VALUES ('notification.email.host','"smtp.example.test"'), ('notification.push.vapid_private_key','"old-key"');
 INSERT INTO inboxes(name,channel) VALUES ('Keep mailbox','email');
 `)
	for range 2 {
		if err := V3_1_0(db, nil, nil); err != nil {
			t.Fatal(err)
		}
	}
	var host string
	if err := db.Get(&host, "SELECT value #>> '{}' FROM settings WHERE key='account_email.host'"); err != nil || host != "smtp.example.test" {
		t.Fatalf("account SMTP not preserved: %q %v", host, err)
	}
	var count int
	for _, query := range []string{
		"SELECT count(*) FROM settings WHERE key LIKE 'notification.%'",
		"SELECT count(*) FROM pg_tables WHERE schemaname=current_schema() AND tablename IN ('user_notifications','user_notification_preferences','notification_email_queue','notification_push_subscriptions')",
		"SELECT count(*) FROM pg_type WHERE typname IN ('notification_channel','user_notification_type')",
	} {
		if err := db.Get(&count, query); err != nil || count != 0 {
			t.Fatalf("retired data remains: %s: %d %v", query, count, err)
		}
	}
	if err := db.Get(&count, "SELECT count(*) FROM inboxes WHERE name='Keep mailbox'"); err != nil || count != 1 {
		t.Fatalf("mailbox lost: %d %v", count, err)
	}
}
