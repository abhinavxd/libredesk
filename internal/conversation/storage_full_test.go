package conversation

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/abhinavxd/libredesk/internal/attachment"
	"github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/inbox"
	"github.com/abhinavxd/libredesk/internal/media"
	fs "github.com/abhinavxd/libredesk/internal/media/stores/localfs"
	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/abhinavxd/libredesk/internal/ws"
	"github.com/volatiletech/null/v9"
	"github.com/zerodha/logf"
)

func TestEmailStorageFullPreservesMessageAndUnavailableAttachments(t *testing.T) {
	db := testutil.NewDB(t, "email_storage_full")
	lo := logf.New(logf.Opts{})
	i18n := testutil.NewI18n(t)
	root := func() string { return "http://localhost" }
	store, _ := fs.New(fs.Opts{UploadPath: t.TempDir(), RootURL: root})
	mediaManager, err := media.New(media.Opts{DB: db, Store: store, Lo: &lo, I18n: i18n, RootURL: root, SigningKey: "test", MaxStorageBytes: 6})
	if err != nil {
		t.Fatal(err)
	}
	var userID, inboxID, conversationID int
	var conversationUUID string
	db.Get(&userID, "INSERT INTO users (type,email,first_name,last_name,avatar_url,availability_status) VALUES ('contact','quota@example.com','Quota','','','online') RETURNING id")
	db.Get(&inboxID, "INSERT INTO inboxes (name,channel) VALUES ('Quota','email') RETURNING id")
	if err := db.Get(&conversationID, "INSERT INTO conversations (contact_id,inbox_id,status_id,subject) VALUES ($1,$2,(SELECT id FROM conversation_statuses LIMIT 1),'Quota') RETURNING id", userID, inboxID); err != nil {
		t.Fatal(err)
	}
	db.Get(&conversationUUID, "SELECT uuid FROM conversations WHERE id=$1", conversationID)
	db.MustExec("INSERT INTO conversation_participants (user_id,conversation_id) VALUES ($1,$2)", userID, conversationID)
	manager, err := New(ws.NewHub(&lo, nil), i18n, nil, nil, nil, nil, nil, nil, mediaManager, stubSettingsStore{}, nil, nil, nil, receiptWebhookStore{}, nil, Opts{DB: db, Lo: &lo})
	if err != nil {
		t.Fatal(err)
	}
	msg := models.Message{ConversationID: conversationID, ConversationUUID: conversationUUID, SenderID: userID, SenderType: models.SenderTypeContact, Type: models.MessageIncoming, Channel: inbox.ChannelEmail, Status: models.MessageStatusReceived, ContentType: models.ContentTypeText, Content: "Email body survives", SourceID: null.StringFrom("quota-message"), Meta: json.RawMessage(`{"to":["support@example.com"]}`), Attachments: attachment.Attachments{
		{Name: "first.txt", ContentType: "text/plain", Content: []byte("1234"), Size: 4, Disposition: "attachment"},
		{Name: "missing.png", ContentType: "image/png", Content: []byte("1234567"), Size: 7, Disposition: "inline"},
		{Name: "last.txt", ContentType: "text/plain", Content: []byte("12"), Size: 2, Disposition: "attachment"},
	}}
	if err := manager.uploadMessageAttachments(&msg); err != nil {
		t.Fatal(err)
	}
	if len(msg.Media) != 2 {
		t.Fatalf("stored attachments=%d", len(msg.Media))
	}
	if err := manager.InsertMessage(&msg); err != nil {
		t.Fatal(err)
	}
	check := func(saved models.Message) {
		t.Helper()
		if saved.Content != "Email body survives" || len(saved.Attachments) != 3 {
			t.Fatalf("message lost data: %+v", saved)
		}
		var missing int
		for _, a := range saved.Attachments {
			if a.Unavailable {
				missing++
				if a.Name != "missing.png" || a.Size != 7 || a.Disposition != "inline" || a.URL != "" || a.ThumbnailURL != "" || a.UnavailableReason != "storage_full" || len(a.Content) != 0 {
					t.Fatalf("bad unavailable attachment: %+v", a)
				}
			} else if a.URL == "" {
				t.Fatal("stored attachment has no URL")
			}
		}
		if missing != 1 {
			t.Fatalf("unavailable count=%d", missing)
		}
		if strings.Contains(string(saved.Meta), "MTIzNDU2Nw==") || !strings.Contains(string(saved.Meta), "support@example.com") {
			t.Fatalf("incorrect persisted metadata: %s", saved.Meta)
		}
	}
	saved, err := manager.GetMessage(msg.UUID)
	if err != nil {
		t.Fatal(err)
	}
	check(saved)
	messages, _, err := manager.GetConversationMessages(conversationUUID, 1, 20, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(messages) != 1 {
		t.Fatalf("messages=%d", len(messages))
	}
	manager.SignAttachmentURLs(messages[0].Attachments)
	check(messages[0])
	// Subsequent mailbox scans deduplicate the saved email instead of reuploading.
	if _, err := manager.ProcessIncomingMessage(models.IncomingMessage{SourceID: null.StringFrom("quota-message")}); err != nil {
		t.Fatal(err)
	}
	usage, err := mediaManager.GetStorageUsage()
	if err != nil || usage.UsedBytes != 6 {
		t.Fatalf("usage=%+v error=%v", usage, err)
	}
}
