package conversation

import (
	"io"
	"testing"

	"github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/media"
	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/zerodha/logf"
)

type attachmentLinkTestStore struct{}

func (attachmentLinkTestStore) Put(name, _ string, _ io.ReadSeeker) (string, error) {
	return name, nil
}

func (attachmentLinkTestStore) Delete(string) error { return nil }

func (attachmentLinkTestStore) GetURL(name, _, _ string) string { return name }

func (attachmentLinkTestStore) GetBlob(string) ([]byte, error) { return nil, nil }

func (attachmentLinkTestStore) Name() string { return "fs" }

func (attachmentLinkTestStore) SignedURLValidator() func(string, string, int64) bool {
	return nil
}

func TestInsertMessageRollsBackWhenNotAllAttachmentsLink(t *testing.T) {
	db := testutil.NewDB(t, "message_attachment_link")
	lo := logf.New(logf.Opts{})
	lang := testutil.NewI18n(t)
	mediaManager, err := media.New(media.Opts{
		Store: attachmentLinkTestStore{},
		Lo:    &lo,
		DB:    db,
		I18n:  lang,
	})
	if err != nil {
		t.Fatalf("creating media manager: %v", err)
	}
	manager, err := New(nil, lang, nil, nil, nil, nil, nil, nil, mediaManager, nil, nil, nil, nil, nil, nil, Opts{
		DB: db,
		Lo: &lo,
	})
	if err != nil {
		t.Fatalf("creating conversation manager: %v", err)
	}

	var contactID, inboxID, conversationID int
	if err := db.Get(&contactID, `INSERT INTO users (type, email, first_name)
		VALUES ('contact', 'attachment-link@example.test', 'Contact') RETURNING id`); err != nil {
		t.Fatalf("creating contact: %v", err)
	}
	if err := db.Get(&inboxID, `INSERT INTO inboxes (channel, name)
		VALUES ('livechat', 'attachment-link') RETURNING id`); err != nil {
		t.Fatalf("creating inbox: %v", err)
	}
	var conversationUUID string
	if err := db.QueryRow(`INSERT INTO conversations (contact_id, inbox_id, status_id)
		VALUES ($1, $2, (SELECT id FROM conversation_statuses WHERE category = 'open' LIMIT 1))
		RETURNING id, uuid`, contactID, inboxID).Scan(&conversationID, &conversationUUID); err != nil {
		t.Fatalf("creating conversation: %v", err)
	}

	attachmentIDs := make([]int, 0, 2)
	for range 2 {
		var id int
		if err := db.Get(&id, `INSERT INTO media (store, filename, content_type, size, model_type, meta)
			VALUES ('fs', 'attachment.txt', 'text/plain', 12, 'messages', jsonb_build_object('widget_contact_id', $1))
			RETURNING id`, contactID); err != nil {
			t.Fatalf("creating staged attachment: %v", err)
		}
		attachmentIDs = append(attachmentIDs, id)
	}

	stagedMedia, err := mediaManager.GetMany(attachmentIDs)
	if err != nil || len(stagedMedia) != len(attachmentIDs) {
		t.Fatalf("loading staged attachments: got %d records, err %v", len(stagedMedia), err)
	}
	if _, err := db.Exec(`UPDATE media SET model_id = 999 WHERE id = $1`, attachmentIDs[1]); err != nil {
		t.Fatalf("simulating an attachment linked after validation: %v", err)
	}

	message := models.Message{
		ConversationID:   conversationID,
		ConversationUUID: conversationUUID,
		SenderID:         contactID,
		Type:             models.MessageIncoming,
		SenderType:       models.SenderTypeContact,
		Status:           models.MessageStatusReceived,
		Content:          "with attachments",
		ContentType:      models.ContentTypeText,
		Media:            stagedMedia,
	}
	if err := manager.InsertMessage(&message); err == nil {
		t.Fatal("InsertMessage succeeded after one attachment became ineligible")
	}

	var messageCount int
	if err := db.Get(&messageCount, `SELECT COUNT(*) FROM conversation_messages WHERE conversation_id = $1`, conversationID); err != nil {
		t.Fatalf("checking for inserted message: %v", err)
	}
	if messageCount != 0 {
		t.Fatalf("message count = %d after failed attachment linking, want 0", messageCount)
	}
	var firstAttachmentModelID, secondAttachmentModelID *int
	if err := db.QueryRow(`SELECT
		(SELECT model_id FROM media WHERE id = $1),
		(SELECT model_id FROM media WHERE id = $2)`, attachmentIDs[0], attachmentIDs[1]).Scan(&firstAttachmentModelID, &secondAttachmentModelID); err != nil {
		t.Fatalf("checking attachment links after rollback: %v", err)
	}
	if firstAttachmentModelID != nil || secondAttachmentModelID == nil || *secondAttachmentModelID != 999 {
		t.Fatalf("model IDs after rollback = %v, %v; want nil, 999", firstAttachmentModelID, secondAttachmentModelID)
	}
}
