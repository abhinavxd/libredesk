package conversation

import (
	"context"
	"errors"
	"testing"

	"github.com/abhinavxd/libredesk/internal/conversation/models"
	mmodels "github.com/abhinavxd/libredesk/internal/media/models"
	"github.com/abhinavxd/libredesk/internal/testutil"
	wmodels "github.com/abhinavxd/libredesk/internal/webhook/models"
	"github.com/abhinavxd/libredesk/internal/ws"
	"github.com/jmoiron/sqlx"
	"github.com/zerodha/logf"
)

type receiptMediaStore struct{ mediaStore }

func (receiptMediaStore) LinkMessageMediaTx(*sqlx.Tx, int, []mmodels.Media, []string) error {
	return nil
}

type receiptWebhookStore struct{ webhookStore }

func (receiptWebhookStore) TriggerEvent(wmodels.WebhookEvent, any) {}

func TestIncomingImageCacheRunsAfterCommit(t *testing.T) {
	db := testutil.NewDB(t, "receipt_ingestion")
	lo := logf.New(logf.Opts{})
	var userID, inboxID, conversationID int
	var conversationUUID string
	if err := db.Get(&userID, "INSERT INTO users (type, email, first_name, last_name, avatar_url, availability_status) VALUES ('contact', 'receipt@example.com', 'Receipt', '', '', 'online') RETURNING id"); err != nil {
		t.Fatal(err)
	}
	if err := db.Get(&inboxID, "INSERT INTO inboxes (name, channel) VALUES ('Receipt', 'email') RETURNING id"); err != nil {
		t.Fatal(err)
	}
	if err := db.Get(&conversationID, "INSERT INTO conversations (contact_id, inbox_id, status_id, subject) VALUES ($1, $2, (SELECT id FROM conversation_statuses LIMIT 1), 'Receipt') RETURNING id", userID, inboxID); err != nil {
		t.Fatal(err)
	}
	if err := db.Get(&conversationUUID, "SELECT uuid FROM conversations WHERE id = $1", conversationID); err != nil {
		t.Fatal(err)
	}
	db.MustExec("INSERT INTO conversation_participants (user_id, conversation_id) VALUES ($1, $2)", userID, conversationID)
	calls := 0
	manager, err := New(ws.NewHub(&lo, nil), testutil.NewI18n(t), nil, nil, nil, nil, nil, nil, receiptMediaStore{}, stubSettingsStore{}, nil, nil, nil, receiptWebhookStore{}, nil, Opts{
		DB: db, Lo: &lo,
		CacheIncomingImages: func(ctx context.Context, id int, content string) error {
			calls++
			var saved string
			if err := db.GetContext(ctx, &saved, "SELECT content FROM conversation_messages WHERE id = $1", id); err != nil || saved != content {
				t.Fatalf("message not committed before caching: %q, %v", saved, err)
			}
			return errors.New("image host unavailable")
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		kind, contentType string
		private           bool
	}{
		{models.MessageIncoming, models.ContentTypeHTML, false},
		{models.MessageIncoming, models.ContentTypeText, false},
		{models.MessageOutgoing, models.ContentTypeHTML, false},
		{models.MessageIncoming, models.ContentTypeHTML, true},
	} {
		message := models.Message{ConversationID: conversationID, ConversationUUID: conversationUUID, SenderID: userID, SenderType: models.SenderTypeContact, Type: tc.kind, Status: models.MessageStatusReceived, ContentType: tc.contentType, Content: `<img src="https://images.example/a">`, Private: tc.private}
		if err := manager.InsertMessage(&message); err != nil {
			t.Fatalf("image failure rejected message: %v", err)
		}
		if message.ID == 0 {
			t.Fatal("message was not saved")
		}
	}
	if calls != 1 {
		t.Fatalf("expected only public incoming HTML to trigger caching, got %d calls", calls)
	}
}
