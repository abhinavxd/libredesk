package conversation

import (
	"encoding/json"
	"strings"
	"testing"

	imodels "github.com/abhinavxd/libredesk/internal/inbox/models"
	mmodels "github.com/abhinavxd/libredesk/internal/media/models"
	"github.com/abhinavxd/libredesk/internal/telegram"
	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/abhinavxd/libredesk/internal/user"
	"github.com/zerodha/logf"
)

func TestPrepareTelegramOutbound(t *testing.T) {
	db := testutil.NewDB(t, "telegram_outbound")
	lang := testutil.NewI18n(t)
	lo := logf.New(logf.Opts{Level: logf.FatalLevel})
	users, err := user.New(lang, user.Opts{DB: db, Lo: &lo})
	if err != nil {
		t.Fatal(err)
	}
	m, err := New(nil, lang, nil, nil, nil, nil, users, nil, nil, nil, nil, nil, nil, nil, nil, Opts{DB: db, Lo: &lo})
	if err != nil {
		t.Fatal(err)
	}
	var inboxID, contactID int
	var uuid string
	db.Get(&inboxID, `INSERT INTO inboxes (name,channel) VALUES ('Telegram','telegram') RETURNING id`)
	db.Get(&contactID, `INSERT INTO users (type,first_name,last_name) VALUES ('contact','Telegram','Contact') RETURNING id`)
	if err := db.Get(&uuid, `INSERT INTO conversations (inbox_id,contact_id,status_id) VALUES ($1,$2,(SELECT id FROM conversation_statuses WHERE category='open' LIMIT 1)) RETURNING uuid`, inboxID, contactID); err != nil {
		t.Fatal(err)
	}
	rec := imodels.Inbox{ID: inboxID, Channel: "telegram"}
	if err := m.prepareTelegramOutbound(rec, uuid, "Hello", nil, map[string]any{}); err == nil {
		t.Fatal("missing identity accepted")
	}
	db.MustExec(`INSERT INTO contact_channel_identities (contact_id,channel,identifier) VALUES ($1,'telegram','4500000000000')`, contactID)
	for _, tc := range []struct {
		name, content string
		media         []mmodels.Media
		wantErr       bool
	}{
		{name: "text", content: "<p>Hello</p>"},
		{name: "text limit", content: strings.Repeat("a", telegram.MaxTextLength)},
		{name: "text over limit", content: strings.Repeat("a", telegram.MaxTextLength+1), wantErr: true},
		{name: "Unicode characters", content: strings.Repeat("😀", telegram.MaxTextLength)},
		{name: "formatting does not consume limit", content: "<b>" + strings.Repeat("a", telegram.MaxTextLength) + "</b>"},
		{name: "escaped characters", content: strings.Repeat("&amp;", telegram.MaxTextLength)},
		{name: "empty", wantErr: true},
		{name: "blank HTML", content: "<p><br></p>", wantErr: true},
		{name: "attachment only", media: []mmodels.Media{{Size: 1}}},
		{name: "caption limit", content: strings.Repeat("a", telegram.MaxCaptionLength), media: []mmodels.Media{{Size: 1}}},
		{name: "caption over limit", content: strings.Repeat("a", telegram.MaxCaptionLength+1), media: []mmodels.Media{{Size: 1}}, wantErr: true},
		{name: "upload limit", media: []mmodels.Media{{Size: telegram.MaxUploadBytes}}},
		{name: "upload over limit", media: []mmodels.Media{{Size: telegram.MaxUploadBytes + 1}}, wantErr: true},
		{name: "album", media: []mmodels.Media{{Size: 1}, {Size: 1}}},
		{name: "too many attachments", media: make([]mmodels.Media, 11), wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			meta := map[string]any{"telegram": map[string]any{"chat_id": 999}, "echo_id": "preserve"}
			err := m.prepareTelegramOutbound(rec, uuid, tc.content, tc.media, meta)
			if (err != nil) != tc.wantErr {
				t.Fatalf("err=%v", err)
			}
			if err == nil {
				var target struct {
					ChatID int64 `json:"chat_id"`
				}
				if err := json.Unmarshal(meta["telegram"].(json.RawMessage), &target); err != nil {
					t.Fatal(err)
				}
				if target.ChatID != 4500000000000 || meta["echo_id"] != "preserve" {
					t.Fatalf("untrusted recipient or lost metadata: %v", meta)
				}
			}
		})
	}
	for _, tc := range []struct {
		name    string
		buttons any
		media   []mmodels.Media
		wantErr bool
	}{
		{name: "reply", buttons: []telegram.Button{{Text: "Yes", Data: "yes"}}},
		{name: "link", buttons: []telegram.Button{{Text: "Docs", URL: "https://example.com"}}},
		{name: "empty", buttons: []telegram.Button{}},
		{name: "invalid action", buttons: []telegram.Button{{Text: "No"}}, wantErr: true},
		{name: "invalid JSON shape", buttons: "bad", wantErr: true},
		{name: "unencodable", buttons: make(chan int), wantErr: true},
		{name: "album buttons", buttons: []telegram.Button{{Text: "Yes", Data: "yes"}}, media: make([]mmodels.Media, 2), wantErr: true},
	} {
		t.Run("buttons/"+tc.name, func(t *testing.T) {
			meta := map[string]any{"telegram_buttons": tc.buttons}
			err := m.prepareTelegramOutbound(rec, uuid, "Choose", tc.media, meta)
			if (err != nil) != tc.wantErr {
				t.Fatalf("err=%v", err)
			}
			if err == nil {
				var routing struct {
					Buttons []telegram.Button `json:"buttons"`
				}
				if err := json.Unmarshal(meta["telegram"].(json.RawMessage), &routing); err != nil {
					t.Fatal(err)
				}
				if len(routing.Buttons) != len(tc.buttons.([]telegram.Button)) {
					t.Fatal("buttons lost")
				}
			}
		})
	}
	var replyUUID string
	if err := db.Get(&replyUUID, `INSERT INTO conversation_messages (conversation_id,sender_id,sender_type,type,status,content,content_type,source_id,text_content) VALUES ((SELECT id FROM conversations WHERE uuid=$1),$2,'contact','incoming','received','Original','text',$3,'Original') RETURNING uuid`, uuid, contactID, telegram.SourceID(inboxID, 4500000000000, 99)); err != nil {
		t.Fatal(err)
	}
	meta := map[string]any{"reply_to_message_uuid": replyUUID}
	if err := m.prepareTelegramOutbound(rec, uuid, "Reply", nil, meta); err != nil {
		t.Fatal(err)
	}
	var send struct {
		ReplyToMessageID int64 `json:"reply_to_message_id"`
	}
	json.Unmarshal(meta["telegram"].(json.RawMessage), &send)
	if send.ReplyToMessageID != 99 || meta["reply_to"].(map[string]any)["content"] != "Original" {
		t.Fatal("reply reference lost")
	}
	db.MustExec(`UPDATE conversation_messages SET private=true WHERE uuid=$1`, replyUUID)
	if err := m.prepareTelegramOutbound(rec, uuid, "Reply", nil, map[string]any{"reply_to_message_uuid": replyUUID}); err == nil {
		t.Fatal("quoted private note leaked")
	}
	db.MustExec(`UPDATE conversation_messages SET private=false, source_id='telegram:999:1:1' WHERE uuid=$1`, replyUUID)
	if err := m.prepareTelegramOutbound(rec, uuid, "Reply", nil, map[string]any{"reply_to_message_uuid": replyUUID}); err == nil {
		t.Fatal("quoted another chat")
	}
	db.MustExec(`UPDATE conversations SET meta='[]' WHERE uuid=$1`, uuid)
	if err := m.prepareTelegramOutbound(rec, uuid, "Reply", nil, map[string]any{}); err == nil {
		t.Fatal("invalid routing accepted")
	}
	db.MustExec(`UPDATE conversations SET meta='{"telegram":{"business_connection_id":"business","message_thread_id":12}}' WHERE uuid=$1`, uuid)
	meta = map[string]any{}
	if err := m.prepareTelegramOutbound(rec, uuid, "Reply", nil, meta); err != nil {
		t.Fatal(err)
	}
	var route map[string]any
	json.Unmarshal(meta["telegram"].(json.RawMessage), &route)
	if route["business_connection_id"] != "business" || route["message_thread_id"] != float64(12) {
		t.Fatal("business routing lost")
	}
	for _, identity := range []string{"0", "-1", "invalid", "9223372036854775808"} {
		db.MustExec(`UPDATE contact_channel_identities SET identifier=$1 WHERE contact_id=$2`, identity, contactID)
		if err := m.prepareTelegramOutbound(rec, uuid, "Hello", nil, map[string]any{}); err == nil {
			t.Fatalf("invalid identity %q accepted", identity)
		}
	}
	if err := m.prepareTelegramOutbound(imodels.Inbox{ID: inboxID + 1}, uuid, "Hello", nil, map[string]any{}); err == nil {
		t.Fatal("wrong inbox accepted")
	}
	if err := m.prepareTelegramOutbound(rec, "00000000-0000-0000-0000-000000000000", "Hello", nil, map[string]any{}); err == nil {
		t.Fatal("missing conversation accepted")
	}
	db.Close()
	if err := m.prepareTelegramOutbound(rec, uuid, "Hello", nil, map[string]any{}); err == nil {
		t.Fatal("database failure ignored")
	}
}
