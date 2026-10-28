package media

import (
	"errors"
	"testing"

	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/media/models"
	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/volatiletech/null/v9"
	"github.com/zerodha/logf"
)

func TestDeletedMessageMediaUnavailable(t *testing.T) {
	db := testutil.NewDB(t, "deleted_message_media")
	lo := logf.New(logf.Opts{})
	m := &Manager{lo: &lo, i18n: testutil.NewI18n(t), rootURL: func() string { return "https://support.example.com" }}
	if err := dbutil.ScanSQLFile("queries.sql", &m.queries, db, efs); err != nil {
		t.Fatal(err)
	}
	var contactID, inboxID, conversationID, messageID int
	if err := db.Get(&contactID, `INSERT INTO users (type, first_name, last_name) VALUES ('contact', 'Customer', '') RETURNING id`); err != nil {
		t.Fatal(err)
	}
	if err := db.Get(&inboxID, `INSERT INTO inboxes (name, channel) VALUES ('Chat', 'livechat') RETURNING id`); err != nil {
		t.Fatal(err)
	}
	if err := db.Get(&conversationID, `INSERT INTO conversations (contact_id, inbox_id, status_id)
		VALUES ($1, $2, (SELECT id FROM conversation_statuses LIMIT 1)) RETURNING id`, contactID, inboxID); err != nil {
		t.Fatal(err)
	}
	if err := db.Get(&messageID, `INSERT INTO conversation_messages (conversation_id, type, status, sender_id, sender_type)
		VALUES ($1, 'incoming', 'sent', $2, 'contact') RETURNING id`, conversationID, contactID); err != nil {
		t.Fatal(err)
	}
	var attachments []models.Media
	for _, modelID := range []null.Int{{}, null.IntFrom(0), null.IntFrom(messageID)} {
		var med models.Media
		if err := db.Get(&med, `INSERT INTO media (store, filename, content_type, content_id, size, model_type, model_id, private)
			VALUES ('fs', 'receipt.png', 'image/png', '', 1, 'messages', $1, false) RETURNING id, uuid`, modelID); err != nil {
			t.Fatal(err)
		}
		attachments = append(attachments, med)
		if _, err := m.Get(med.ID, "" /** uuid **/); err != nil {
			t.Fatal(err)
		}
		if _, err := m.Get(0 /** id **/, med.UUID); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.Exec(`DELETE FROM users WHERE id = $1`, contactID); err != nil {
		t.Fatal(err)
	}
	for i, med := range attachments {
		for _, lookup := range []func() (models.Media, error){
			func() (models.Media, error) { return m.Get(med.ID, "" /** uuid **/) },
			func() (models.Media, error) { return m.Get(0 /** id **/, med.UUID) },
		} {
			_, err := lookup()
			if i < 2 {
				if err != nil {
					t.Fatalf("unattached draft became unavailable: %v", err)
				}
				continue
			}
			var missing envelope.Error
			if !errors.As(err, &missing) || missing.ErrorType != envelope.NotFoundError {
				t.Fatalf("deleted contact attachment remained available: %v", err)
			}
		}
	}
}

func TestPendingMediaOwnership(t *testing.T) {
	db := testutil.NewDB(t, "media_ownership")
	lo := logf.New(logf.Opts{})
	m := &Manager{lo: &lo, i18n: testutil.NewI18n(t)}
	if err := dbutil.ScanSQLFile("queries.sql", &m.queries, db, efs); err != nil {
		t.Fatal(err)
	}
	var owner, other int
	for _, id := range []*int{&owner, &other} {
		if err := db.Get(id, `INSERT INTO users (type, first_name, last_name) VALUES ('agent', 'Agent', '') RETURNING id`); err != nil {
			t.Fatal(err)
		}
	}
	for _, uploader := range []int{owner, other, 0} {
		var med models.Media
		if err := db.Get(&med, `INSERT INTO media (store, filename, content_type, content_id, size, model_type, uploaded_by) VALUES ('fs', 'draft.png', 'image/png', '', 1, 'messages', NULLIF($1, 0)) RETURNING id, uuid`, uploader); err != nil {
			t.Fatal(err)
		}
		_, err := m.GetDraftInlineMedia(med.UUID, 0 /** conversationID **/, owner)
		if (err == nil) != (uploader == owner) {
			t.Fatalf("draft access for uploader %d: %v", uploader, err)
		}
		tx, err := db.Beginx()
		if err != nil {
			t.Fatal(err)
		}
		if err := m.LinkMessageMediaTx(tx, 10 /** messageID **/, []models.Media{med, med}, []string{med.UUID, med.UUID}, owner); err != nil {
			tx.Rollback()
			t.Fatal(err)
		}
		var linked null.Int
		if err := tx.Get(&linked, `SELECT model_id FROM media WHERE id = $1`, med.ID); err != nil {
			tx.Rollback()
			t.Fatal(err)
		}
		tx.Rollback()
		if linked.Valid != (uploader == owner) {
			t.Fatalf("media linked for uploader %d: %v", uploader, linked)
		}
	}
}
