package migrations

import (
	"testing"

	"github.com/jakedolan443/fernmail/internal/testutil"
)

func TestReliabilityUpgradePreservesMailAndDoesNotRestoreViews(t *testing.T) {
	db := testutil.NewDB(t, "mail_reliability_upgrade")
	// Recreate the pre-upgrade shape with real pending mail and a legacy draft.
	db.MustExec(`
		DROP TABLE conversation_draft_media;
		ALTER TABLE media DROP COLUMN uploaded_by;
		DROP TRIGGER reserve_received_mail_source ON conversation_messages;
		DROP TABLE received_mail_sources, mail_delivery_attempts, incoming_mail_queue, mail_sync_failures, mail_sync_cursors;
		ALTER TABLE conversation_messages DROP COLUMN reply_to_source_id;
		INSERT INTO users(type,email,first_name) VALUES('agent','agent@example.test','Agent'),('contact','sender@example.test','Sender');
		INSERT INTO inboxes(name,channel) VALUES('Mailbox','email');
		INSERT INTO email_addresses(inbox_id,address,kind) SELECT id,'mail@example.test','mailbox' FROM inboxes;
		INSERT INTO conversations(contact_id,inbox_id,address_id,status_id)
		SELECT u.id,i.id,a.id,s.id FROM users u CROSS JOIN inboxes i CROSS JOIN email_addresses a CROSS JOIN conversation_statuses s
		WHERE u.type='contact' AND s.name='Open';
		INSERT INTO conversation_messages(conversation_id,sender_id,sender_type,type,status,source_id,content)
		SELECT c.id,u.id,'contact','incoming','received','original@example.test','Original incoming mail' FROM conversations c CROSS JOIN users u WHERE u.type='contact';
		INSERT INTO conversation_messages(conversation_id,sender_id,sender_type,type,status,source_id,content)
		SELECT c.id,u.id,'agent','outgoing','pending','reply@example.test','Pending reply' FROM conversations c CROSS JOIN users u WHERE u.type='agent';
		INSERT INTO media(store,filename,content_type,size,model_type,uuid)
		VALUES('fs','draft.txt','text/plain',42,'messages','550e8400-e29b-41d4-a716-446655440000');
		INSERT INTO conversation_drafts(conversation_id,user_id,type,content,meta)
		SELECT c.id,u.id,'reply','Legacy draft',jsonb_build_object('attachments',jsonb_build_array(jsonb_build_object('id',m.id)))
		FROM conversations c CROSS JOIN users u CROSS JOIN media m WHERE u.type='agent';
	`)
	for range 2 {
		if err := V3_6_0(db, nil, nil); err != nil {
			t.Fatal(err)
		}
	}
	var count int
	if err := db.Get(&count, `SELECT count(*) FROM conversation_messages WHERE (source_id='original@example.test' AND content='Original incoming mail' AND status='received') OR (source_id='reply@example.test' AND content='Pending reply' AND status='pending')`); err != nil || count != 2 {
		t.Fatalf("mail preservation count=%d error=%v", count, err)
	}
	if err := db.Get(&count, `SELECT count(*) FROM received_mail_sources WHERE source_id='original@example.test'`); err != nil || count != 1 {
		t.Fatalf("received identity backfill count=%d error=%v", count, err)
	}
	if err := db.Get(&count, `SELECT count(*) FROM conversation_draft_media`); err != nil || count != 1 {
		t.Fatalf("legacy draft retention count=%d error=%v", count, err)
	}
	if err := db.Get(&count, `SELECT count(*) FROM media WHERE uploaded_by IS NOT NULL`); err != nil || count != 0 {
		t.Fatalf("legacy metadata was trusted as ownership count=%d error=%v", count, err)
	}
	var views bool
	if err := db.Get(&views, `SELECT to_regclass('public.views') IS NOT NULL`); err != nil || views {
		t.Fatalf("Views restored=%t error=%v", views, err)
	}
}
