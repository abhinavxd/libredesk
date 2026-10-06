package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"

	amodels "github.com/abhinavxd/libredesk/internal/auth/models"
	"github.com/abhinavxd/libredesk/internal/authz"
	"github.com/abhinavxd/libredesk/internal/automation"
	"github.com/abhinavxd/libredesk/internal/conversation"
	cmodels "github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/csat"
	telegramChannel "github.com/abhinavxd/libredesk/internal/inbox/channel/telegram"
	imodels "github.com/abhinavxd/libredesk/internal/inbox/models"
	"github.com/abhinavxd/libredesk/internal/media"
	localfs "github.com/abhinavxd/libredesk/internal/media/stores/localfs"
	"github.com/abhinavxd/libredesk/internal/setting"
	"github.com/abhinavxd/libredesk/internal/sla"
	"github.com/abhinavxd/libredesk/internal/telegram"
	tmpl "github.com/abhinavxd/libredesk/internal/template"
	"github.com/abhinavxd/libredesk/internal/user"
	"github.com/abhinavxd/libredesk/internal/webhook"
	"github.com/abhinavxd/libredesk/internal/ws"
	"github.com/jmoiron/sqlx"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

func TestTelegramWebhookValidation(t *testing.T) {
	app, db := newInboxApp(t)
	app.ctx = t.Context()
	inbox, err := app.inbox.Create(imodels.Inbox{Channel: "telegram", Name: "test", Enabled: true, Config: json.RawMessage(`{"bot_token":"123:secret","secret_token":"webhook-secret"}`)})
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, path, secret, body string
		status                   int
	}{
		{"invalid id", "nope", "", "{}", 400},
		{"missing inbox", "99999", "", "{}", 400},
		{"missing secret", strconv.Itoa(inbox.ID), "", "{}", 403},
		{"wrong secret", strconv.Itoa(inbox.ID), "incorrect", "{}", 403},
		{"malformed update", strconv.Itoa(inbox.ID), "webhook-secret", "{", 400},
		{"other update", strconv.Itoa(inbox.ID), "webhook-secret", `{"update_id":1}`, 200},
		{"group message", strconv.Itoa(inbox.ID), "webhook-secret", `{"message":{"message_id":1,"from":{"id":42},"chat":{"id":-123,"type":"group"},"text":"ignore"}}`, 200},
		{"sender mismatch", strconv.Itoa(inbox.ID), "webhook-secret", `{"message":{"message_id":1,"from":{"id":42},"chat":{"id":43,"type":"private"},"text":"ignore"}}`, 200},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := telegramTestRequest(app, tc.path, tc.secret, tc.body)
			if err := handleTelegramWebhook(r); err != nil {
				t.Fatal(err)
			}
			if got := r.RequestCtx.Response.StatusCode(); got != tc.status {
				t.Fatalf("status=%d body=%s", got, r.RequestCtx.Response.Body())
			}
			if !json.Valid(r.RequestCtx.Response.Body()) {
				t.Fatal("handler omitted JSON envelope")
			}
		})
	}
	db.MustExec(`UPDATE inboxes SET enabled=false WHERE id=$1`, inbox.ID)
	r := telegramTestRequest(app, strconv.Itoa(inbox.ID), "webhook-secret", "{")
	if err := handleTelegramWebhook(r); err != nil || r.RequestCtx.Response.StatusCode() != 200 {
		t.Fatalf("disabled inbox: %v", err)
	}
	db.MustExec(`UPDATE inboxes SET channel='email',config='{}' WHERE id=$1`, inbox.ID)
	r = telegramTestRequest(app, strconv.Itoa(inbox.ID), "webhook-secret", "{}")
	handleTelegramWebhook(r)
	if r.RequestCtx.Response.StatusCode() != 404 {
		t.Fatal("accepted other channel")
	}
	db.MustExec(`UPDATE inboxes SET channel='telegram',config='{"bot_id":"invalid"}' WHERE id=$1`, inbox.ID)
	r = telegramTestRequest(app, strconv.Itoa(inbox.ID), "webhook-secret", "{}")
	handleTelegramWebhook(r)
	if r.RequestCtx.Response.StatusCode() != 500 {
		t.Fatal("invalid config hidden")
	}
	if err := ingestTelegramMessage(t.Context(), app, inbox, telegramChannel.Config{}, telegram.Message{}); err != nil {
		t.Fatal(err)
	}
}

func TestFetchTelegramAttachment(t *testing.T) {
	app := testI18nApp(t)
	app.consts.Store(&constants{MaxFileUploadSizeMB: 100})
	app.telegramClient = telegram.New()
	cfg := telegramChannel.Config{BotToken: "123:secret"}
	for _, tc := range []struct {
		name, payload, response, body, wantName, wantType string
		status                                            int
		notice, wantErr                                   bool
	}{
		{name: "text", payload: `{"text":"hi"}`},
		{name: "unsupported", payload: `{}`, notice: true},
		{name: "oversize metadata", payload: `{"document":{"file_id":"file","file_size":20971521}}`, notice: true},
		{name: "not found", payload: `{"document":{"file_id":"file"}}`, response: `{"ok":false,"error_code":404}`, notice: true},
		{name: "bad file", payload: `{"document":{"file_id":"file"}}`, response: `{"ok":false,"error_code":400}`, notice: true},
		{name: "expired token", payload: `{"document":{"file_id":"file"}}`, response: `{"ok":false,"error_code":401}`, wantErr: true},
		{name: "server failure", payload: `{"document":{"file_id":"file"}}`, response: `{"ok":false,"error_code":500}`, wantErr: true},
		{name: "oversize getFile", payload: `{"document":{"file_id":"file"}}`, response: `{"ok":true,"result":{"file_size":20971521}}`, notice: true},
		{name: "empty file", payload: `{"document":{"file_id":"file"}}`, response: `{"ok":true,"result":{"file_path":"documents/file.txt"}}`, notice: true},
		{name: "original filename", payload: `{"document":{"file_id":"file","file_name":"invoice.txt","mime_type":"text/plain"}}`, body: "test", wantName: "invoice.txt", wantType: "text/plain"},
		{name: "extension MIME", payload: `{"document":{"file_id":"file"}}`, body: "test", wantName: "file.txt", wantType: "text/plain; charset=utf-8"},
		{name: "sniffed MIME", payload: `{"document":{"file_id":"file","file_name":"no-extension"}}`, body: "test", wantName: "no-extension", wantType: "text/plain; charset=utf-8"},
		{name: "name fallback", payload: `{"document":{"file_id":"file","file_name":"."}}`, body: "test", wantName: "document.plain", wantType: "text/plain; charset=utf-8"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var msg telegram.Message
			if err := json.Unmarshal([]byte(tc.payload), &msg); err != nil {
				t.Fatal(err)
			}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/bot123:secret/getFile" {
					response := tc.response
					if response == "" {
						response = `{"ok":true,"result":{"file_path":"documents/file.txt"}}`
					}
					io.WriteString(w, response)
				} else {
					io.WriteString(w, tc.body)
				}
			}))
			defer server.Close()
			app.telegramClient.SetBaseURL(server.URL)
			files, notice, err := fetchTelegramAttachment(t.Context(), app, cfg, msg)
			if (err != nil) != tc.wantErr || (notice != "") != tc.notice {
				t.Fatalf("notice=%q err=%v", notice, err)
			}
			if tc.wantName != "" && (len(files) != 1 || files[0].Name != tc.wantName || files[0].ContentType != tc.wantType || string(files[0].Content) != tc.body) {
				t.Fatalf("files=%+v", files)
			}
		})
	}
	app.consts.Store(&constants{MaxFileUploadSizeMB: 1})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/bot123:secret/getFile" {
			io.WriteString(w, `{"ok":true,"result":{"file_path":"documents/file.txt"}}`)
			return
		}
		w.Write(make([]byte, 1024*1024+1))
	}))
	defer server.Close()
	app.telegramClient.SetBaseURL(server.URL)
	for _, file := range []*telegram.File{{ID: "file", Size: 1024*1024 + 1}, {ID: "file"}} {
		files, notice, err := fetchTelegramAttachment(t.Context(), app, cfg, telegram.Message{Document: file})
		if err != nil || notice == "" || len(files) != 0 {
			t.Fatalf("upload limit ignored: files=%d notice=%q err=%v", len(files), notice, err)
		}
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, _, err := fetchTelegramAttachment(ctx, app, cfg, telegram.Message{Document: &telegram.File{ID: "file"}})
	if err == nil {
		t.Fatal("canceled download succeeded")
	}
}

func TestTelegramCallbackURL(t *testing.T) {
	for _, tc := range []struct{ root, want string }{{"", ""}, {"https://desk.example.com", "https://desk.example.com/webhooks/telegram/7"}, {"https://desk.example.com///", "https://desk.example.com/webhooks/telegram/7"}} {
		if got := telegramCallbackURL(tc.root, 7); got != tc.want {
			t.Fatalf("got %q want %q", got, tc.want)
		}
	}
}

func telegramTestRequest(app *App, id, secret, body string) *fastglue.Request {
	r := &fastglue.Request{Context: app, RequestCtx: &fasthttp.RequestCtx{}}
	r.RequestCtx.SetUserValue("inbox_id", id)
	r.RequestCtx.Request.Header.Set("X-Telegram-Bot-Api-Secret-Token", secret)
	r.RequestCtx.Request.SetBodyString(body)
	return r
}

func TestTelegramIngestionAndReopening(t *testing.T) {
	app, db, rec := newTelegramIntegrationApp(t)
	cfg := telegramChannel.Config{BotToken: "123:secret", SecretToken: "webhook-secret"}
	var msg telegram.Message
	json.Unmarshal([]byte(`{"message_id":1,"from":{"id":4500000000000,"first_name":"First","last_name":"Last"},"chat":{"id":4500000000000,"type":"private"},"text":"First message"}`), &msg)
	deliver := func() {
		t.Helper()
		if err := ingestTelegramMessage(t.Context(), app, rec, cfg, msg); err != nil {
			t.Fatal(err)
		}
	}
	count := func(table string) int {
		t.Helper()
		var n int
		if err := db.Get(&n, "SELECT COUNT(*) FROM "+table); err != nil {
			t.Fatal(err)
		}
		return n
	}
	deliver()
	var original int
	db.Get(&original, `SELECT id FROM conversations WHERE inbox_id=$1`, rec.ID)
	var wg sync.WaitGroup
	for range 10 {
		wg.Go(func() {
			if err := ingestTelegramMessage(t.Context(), app, rec, cfg, msg); err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	if count("conversation_messages") != 1 || count("contact_channel_identities") != 1 || count("conversations") != 1 {
		t.Fatal("duplicate delivery created extra data")
	}
	msg.ID++
	msg.Text = "Second message"
	deliver()
	if count("conversations") != 1 {
		t.Fatal("open conversation not reused")
	}
	resolve := func() {
		db.MustExec(`UPDATE conversations SET status_id=(SELECT id FROM conversation_statuses WHERE category='resolved' LIMIT 1),last_resolved_at=NOW() WHERE inbox_id=$1`, rec.ID)
	}
	resolve()
	msg.ID++
	deliver()
	var status string
	db.Get(&status, `SELECT cs.category FROM conversations c JOIN conversation_statuses cs ON cs.id=c.status_id WHERE c.id=$1`, original)
	if status != "open" || count("conversations") != 1 {
		t.Fatalf("recent conversation was not reopened: status=%s", status)
	}
	resolve()
	db.MustExec(`UPDATE conversations SET last_resolved_at=NOW()-INTERVAL '49 hours' WHERE id=$1`, original)
	msg.ID++
	deliver()
	if count("conversations") != 2 {
		t.Fatal("expired reopen window reused conversation")
	}
	resolve()
	rec.ReopenWindowHours = 0
	msg.ID++
	deliver()
	if count("conversations") != 3 {
		t.Fatal("zero reopen window reused conversation")
	}
	other, err := app.inbox.Create(imodels.Inbox{Name: "second bot", Channel: "telegram", Config: rec.Config})
	if err != nil {
		t.Fatal(err)
	}
	if err := ingestTelegramMessage(t.Context(), app, other, cfg, msg); err != nil {
		t.Fatal(err)
	}
	if count("contact_channel_identities") != 1 || count("conversations") != 4 {
		t.Fatal("cross-bot contact identity or routing incorrect")
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	msg.ID++
	if err := ingestTelegramMessage(ctx, app, rec, cfg, msg); err != context.Canceled {
		t.Fatalf("cancellation=%v", err)
	}

	source := telegram.SourceID(rec.ID, msg.Chat.ID, 1)
	chatID := strconv.FormatInt(msg.Chat.ID, 10)
	if err := app.conversation.UpdateTelegramMessageContent(rec.ID, chatID, source, "Corrected", "Corrected", cmodels.ContentTypeText, 100); err != nil {
		t.Fatal(err)
	}
	if err := app.conversation.UpdateTelegramMessageContent(rec.ID, chatID, source, "Older edit", "Older edit", cmodels.ContentTypeText, 99); err != nil {
		t.Fatal(err)
	}
	var content string
	db.Get(&content, `SELECT content FROM conversation_messages WHERE source_id=$1`, source)
	if content != "Corrected" {
		t.Fatal("older edit overwrote correction")
	}
	if err := app.conversation.UpdateTelegramMessageContent(rec.ID, "1", source, "Other chat", "Other chat", cmodels.ContentTypeText, 200); err != nil {
		t.Fatal(err)
	}
	if err := app.conversation.UpdateTelegramMessageContent(other.ID, chatID, source, "Other inbox", "Other inbox", cmodels.ContentTypeText, 200); err != nil {
		t.Fatal(err)
	}
	db.Get(&content, `SELECT content FROM conversation_messages WHERE source_id=$1`, source)
	if content != "Corrected" {
		t.Fatal("edit from another chat or inbox changed the message")
	}
	if err := app.conversation.UpdateTelegramMessageContent(rec.ID, chatID, "telegram:999:999:999", "missing", "missing", cmodels.ContentTypeText, 100); err != nil {
		t.Fatal(err)
	}
	request := telegramTestRequest(app, strconv.Itoa(rec.ID), "webhook-secret", `{"edited_message":{"message_id":1,"edit_date":101,"from":{"id":4500000000000},"chat":{"id":4500000000000,"type":"private"},"text":"Edited through webhook"}}`)
	handleTelegramWebhook(request)
	if request.RequestCtx.Response.StatusCode() != 200 {
		t.Fatalf("edit failed: %s", request.RequestCtx.Response.Body())
	}
	db.Get(&content, `SELECT content FROM conversation_messages WHERE source_id=$1`, source)
	if content != "Edited through webhook" {
		t.Fatal("webhook edit not saved")
	}
	request = telegramTestRequest(app, strconv.Itoa(rec.ID), "webhook-secret", `{"message":{"message_id":99,"from":{"id":4500000000000},"chat":{"id":4500000000000,"type":"private"},"text":"Through webhook"}}`)
	handleTelegramWebhook(request)
	if request.RequestCtx.Response.StatusCode() != 200 {
		t.Fatalf("webhook failed: %s", request.RequestCtx.Response.Body())
	}
}

func TestTelegramIngestionFailuresAndMedia(t *testing.T) {
	app, db, rec := newTelegramIntegrationApp(t)
	cfg := telegramChannel.Config{BotToken: "123:secret"}
	var msg telegram.Message
	json.Unmarshal([]byte(`{"message_id":1,"from":{"id":42,"first_name":"Test"},"chat":{"id":42,"type":"private"},"text":"Caption"}`), &msg)
	getFileFailure := true
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/bot123:secret/getFile" {
			if getFileFailure {
				io.WriteString(w, `{"ok":false,"error_code":500}`)
			} else {
				io.WriteString(w, `{"ok":true,"result":{"file_path":"documents/file.txt"}}`)
			}
		} else {
			io.WriteString(w, "attachment content")
		}
	}))
	defer server.Close()
	app.telegramClient.SetBaseURL(server.URL)
	msg.Document = &telegram.File{ID: "file", Name: "original.txt", MIME: "text/plain"}
	if err := ingestTelegramMessage(t.Context(), app, rec, cfg, msg); err == nil {
		t.Fatal("transient media failure acknowledged")
	}
	getFileFailure = false
	db.MustExec(`ALTER TABLE media ADD CONSTRAINT telegram_test_reject_media CHECK (false) NOT VALID`)
	if err := ingestTelegramMessage(t.Context(), app, rec, cfg, msg); err == nil {
		t.Fatal("media storage failure ignored")
	}
	db.MustExec(`ALTER TABLE media DROP CONSTRAINT telegram_test_reject_media`)
	if err := ingestTelegramMessage(t.Context(), app, rec, cfg, msg); err != nil {
		t.Fatal(err)
	}
	var name string
	if err := db.Get(&name, `SELECT filename FROM media LIMIT 1`); err != nil {
		t.Fatal(err)
	}
	if name != "original.txt" {
		t.Fatalf("filename=%s", name)
	}
	msg.ID++
	msg.Text = ""
	if err := ingestTelegramMessage(t.Context(), app, rec, cfg, msg); err != nil {
		t.Fatal(err)
	}
	msg.ID++
	msg.Document.Size = telegram.MaxDownloadBytes + 1
	msg.Text = "Unavailable attachment caption"
	if err := ingestTelegramMessage(t.Context(), app, rec, cfg, msg); err != nil {
		t.Fatal(err)
	}
	msg.ID++
	msg.Text = ""
	if err := ingestTelegramMessage(t.Context(), app, rec, cfg, msg); err != nil {
		t.Fatal(err)
	}
	msg.ID++
	msg.Document = nil
	msg.Text = "retry me"
	db.MustExec(`CREATE OR REPLACE FUNCTION reject_telegram_test() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'test failure'; END $$`)
	for _, table := range []string{"users", "conversations", "conversation_messages"} {
		db.MustExec(`CREATE TRIGGER reject_telegram_test BEFORE INSERT ON ` + table + ` FOR EACH ROW EXECUTE FUNCTION reject_telegram_test()`)
		msg.Chat.ID++
		msg.From.ID = msg.Chat.ID
		msg.ID++
		if err := ingestTelegramMessage(t.Context(), app, rec, cfg, msg); err == nil {
			t.Fatalf("%s failure ignored", table)
		}
		if table == "conversation_messages" {
			body, _ := json.Marshal(telegram.Update{Message: &msg})
			r := telegramTestRequest(app, strconv.Itoa(rec.ID), "webhook-secret", string(body))
			handleTelegramWebhook(r)
			if r.RequestCtx.Response.StatusCode() != 503 {
				t.Fatalf("storage failure acknowledged: %s", r.RequestCtx.Response.Body())
			}
		}
		db.MustExec(`DROP TRIGGER reject_telegram_test ON ` + table)
	}
	if err := ingestTelegramMessage(t.Context(), app, rec, cfg, msg); err != nil {
		t.Fatal("retry after recovery failed:", err)
	}
	msg.ID++
	db.MustExec(`ALTER TABLE conversations RENAME COLUMN contact_id TO missing_contact_id`)
	if err := ingestTelegramMessage(t.Context(), app, rec, cfg, msg); err == nil {
		t.Fatal("conversation lookup error ignored")
	}
	db.MustExec(`ALTER TABLE conversations RENAME COLUMN missing_contact_id TO contact_id`)
	db.Close()
	if err := ingestTelegramMessage(t.Context(), app, rec, cfg, msg); err == nil {
		t.Fatal("duplicate lookup database failure ignored")
	}
	if err := app.conversation.UpdateTelegramMessageContent(rec.ID, "42", "source", "text", "text", cmodels.ContentTypeText, 100); err == nil {
		t.Fatal("edit database failure ignored")
	}
}

func TestTelegramBusinessAndTopics(t *testing.T) {
	app, db, rec := newTelegramIntegrationApp(t)
	var count int
	for _, body := range []string{
		`{"business_message":{"business_connection_id":"business-a","message_id":1,"from":{"id":7777,"first_name":"Customer"},"chat":{"id":7777,"type":"private"},"text":"Customer to business A"}}`,
		`{"business_message":{"business_connection_id":"business-b","message_id":1,"from":{"id":7777},"chat":{"id":7777,"type":"private"},"text":"Customer to business B"}}`,
		`{"message":{"message_id":1,"from":{"id":7777},"chat":{"id":7777,"type":"private"},"text":"Customer to bot"}}`,
		`{"message":{"message_id":2,"message_thread_id":44,"from":{"id":7777},"chat":{"id":7777,"type":"private"},"text":"Customer in topic"}}`,
		`{"business_message":{"business_connection_id":"business-a","message_id":2,"from":{"id":8888,"first_name":"Owner"},"chat":{"id":7777,"type":"private","first_name":"Customer"},"text":"Reply from Telegram app"}}`,
		`{"business_message":{"business_connection_id":"business-c","message_id":1,"from":{"id":8888,"first_name":"Owner"},"chat":{"id":7777,"type":"private","first_name":"Customer"},"text":"Owner initiates"}}`,
	} {
		r := telegramTestRequest(app, strconv.Itoa(rec.ID), "webhook-secret", body)
		if err := handleTelegramWebhook(r); err != nil {
			t.Fatal(err)
		}
		if r.RequestCtx.Response.StatusCode() != 200 {
			t.Fatalf("status=%d body=%s", r.RequestCtx.Response.StatusCode(), r.RequestCtx.Response.Body())
		}
	}
	db.Get(&count, `SELECT COUNT(*) FROM conversations WHERE inbox_id=$1`, rec.ID)
	if count != 5 {
		t.Fatalf("separate business accounts, bot chats and topics mixed: %d", count)
	}
	db.Get(&count, `SELECT COUNT(*) FROM conversation_messages WHERE type='outgoing'`)
	if count != 2 {
		t.Fatalf("business owner replies not mirrored: %d", count)
	}
	db.Get(&count, `SELECT COUNT(*) FROM contact_channel_identities WHERE channel='telegram' AND identifier='7777'`)
	if count != 1 {
		t.Fatal("same customer identity duplicated")
	}
	r := telegramTestRequest(app, strconv.Itoa(rec.ID), "webhook-secret", `{"edited_business_message":{"business_connection_id":"business-a","message_id":2,"edit_date":100,"from":{"id":8888},"chat":{"id":7777,"type":"private"},"text":"Corrected owner reply"}}`)
	handleTelegramWebhook(r)
	var content string
	db.Get(&content, `SELECT content FROM conversation_messages WHERE source_id=$1`, telegram.BusinessSourceID(rec.ID, "business-a", 7777, 2))
	if content != "Corrected owner reply" {
		t.Fatal("business edit not applied")
	}
	db.MustExec(`UPDATE inboxes SET config = config || '{"bot_id":123}'::jsonb WHERE id=$1`, rec.ID)
	r = telegramTestRequest(app, strconv.Itoa(rec.ID), "webhook-secret", `{"business_message":{"business_connection_id":"business-a","message_id":99,"sender_business_bot":{"id":123},"from":{"id":8888},"chat":{"id":7777,"type":"private"},"text":"Own echo"}}`)
	handleTelegramWebhook(r)
	db.Get(&count, `SELECT COUNT(*) FROM conversation_messages WHERE source_id=$1`, telegram.BusinessSourceID(rec.ID, "business-a", 7777, 99))
	if count != 0 {
		t.Fatal("own bot echo duplicated")
	}
	r = telegramTestRequest(app, strconv.Itoa(rec.ID), "webhook-secret", `{"edited_business_message":{"business_connection_id":"business-a","message_id":2,"edit_date":101,"sender_business_bot":{"id":123},"from":{"id":8888},"chat":{"id":7777,"type":"private"},"text":"Edited bot reply"}}`)
	handleTelegramWebhook(r)
	db.Get(&content, `SELECT content FROM conversation_messages WHERE source_id=$1`, telegram.BusinessSourceID(rec.ID, "business-a", 7777, 2))
	if content != "Edited bot reply" {
		t.Fatal("own bot message edit ignored")
	}
	db.Get(&content, `SELECT last_message FROM conversations WHERE meta->'telegram'->>'business_connection_id'='business-a'`)
	if content != "Edited bot reply" {
		t.Fatalf("conversation preview stale: %s", content)
	}
	db.MustExec(`DELETE FROM users WHERE email='System'`)
	r = telegramTestRequest(app, strconv.Itoa(rec.ID), "webhook-secret", `{"business_message":{"business_connection_id":"business-a","message_id":100,"from":{"id":8888},"chat":{"id":7777,"type":"private"},"text":"Cannot attribute owner"}}`)
	handleTelegramWebhook(r)
	if r.RequestCtx.Response.StatusCode() != 503 {
		t.Fatal("owner attribution error acknowledged")
	}
}

func TestTelegramContactAvatar(t *testing.T) {
	app, db, rec := newTelegramIntegrationApp(t)
	var contactID int
	if err := db.Get(&contactID, `INSERT INTO users (type,first_name,last_name) VALUES ('contact','Avatar','Test') RETURNING id`); err != nil {
		t.Fatal(err)
	}
	var picture bytes.Buffer
	if err := png.Encode(&picture, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		t.Fatal(err)
	}
	mode := "profile error"
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		switch r.URL.Path {
		case "/bot123:secret/getUserProfilePhotos":
			if mode == "profile error" {
				io.WriteString(w, `{"ok":false,"error_code":401}`)
			} else if mode == "no photo" {
				io.WriteString(w, `{"ok":true,"result":{"photos":[]}}`)
			} else {
				io.WriteString(w, `{"ok":true,"result":{"photos":[[{"file_id":"photo"}]]}}`)
			}
		case "/bot123:secret/getFile":
			if mode == "download error" {
				io.WriteString(w, `{"ok":false,"error_code":400}`)
			} else {
				io.WriteString(w, `{"ok":true,"result":{"file_path":"photos/avatar.png"}}`)
			}
		default:
			if mode == "invalid image" {
				io.WriteString(w, "not an image")
			} else {
				w.Write(picture.Bytes())
			}
		}
	}))
	defer server.Close()
	app.telegramClient.SetBaseURL(server.URL)
	cfg := telegramChannel.Config{BotToken: "123:secret"}
	for _, value := range []string{"profile error", "download error", "invalid image"} {
		mode = value
		if err := fetchTelegramAvatar(t.Context(), app, cfg, contactID, 42); err == nil {
			t.Fatalf("%s ignored", mode)
		}
	}
	mode = "no photo"
	if err := fetchTelegramAvatar(t.Context(), app, cfg, contactID, 42); err != nil {
		t.Fatal(err)
	}
	mode = "valid"
	db.MustExec(`ALTER TABLE media ADD CONSTRAINT telegram_test_avatar CHECK (false) NOT VALID`)
	if err := fetchTelegramAvatar(t.Context(), app, cfg, contactID, 42); err == nil {
		t.Fatal("media error ignored")
	}
	db.MustExec(`ALTER TABLE media DROP CONSTRAINT telegram_test_avatar`)
	db.MustExec(`CREATE OR REPLACE FUNCTION reject_telegram_test() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'test failure'; END $$`)
	db.MustExec(`CREATE TRIGGER reject_telegram_avatar BEFORE UPDATE ON users FOR EACH ROW EXECUTE FUNCTION reject_telegram_test()`)
	if err := fetchTelegramAvatar(t.Context(), app, cfg, contactID, 42); err == nil {
		t.Fatal("avatar update error ignored")
	}
	db.MustExec(`DROP TRIGGER reject_telegram_avatar ON users`)
	if err := fetchTelegramAvatar(t.Context(), app, cfg, contactID, 42); err != nil {
		t.Fatal(err)
	}
	var avatar string
	if err := db.Get(&avatar, `SELECT avatar_url FROM users WHERE id=$1`, contactID); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(avatar, "/uploads/") {
		t.Fatalf("avatar must store a relative path: %s", avatar)
	}
	before := calls
	if err := fetchTelegramAvatar(t.Context(), app, cfg, contactID, 42); err != nil || calls != before {
		t.Fatal("existing avatar overwritten")
	}
	if err := fetchTelegramAvatar(t.Context(), app, cfg, 999999, 42); err == nil {
		t.Fatal("missing contact accepted")
	}
	mode = "no photo"
	r := telegramTestRequest(app, strconv.Itoa(rec.ID), "webhook-secret", `{"message":{"message_id":1,"from":{"id":88888},"chat":{"id":88888,"type":"private"},"reply_to_message":{"message_id":99,"text":"Original"},"text":"Quoted reply"}}`)
	handleTelegramWebhook(r)
	if r.RequestCtx.Response.StatusCode() != 200 {
		t.Fatalf("quoted ingress failed: %s", r.RequestCtx.Response.Body())
	}
	var quote string
	db.Get(&quote, `SELECT meta->'reply_to'->>'content' FROM conversation_messages WHERE source_id=$1`, telegram.SourceID(rec.ID, 88888, 1))
	if quote != "Original" {
		t.Fatal("incoming quote context lost")
	}
	before = calls
	r = telegramTestRequest(app, strconv.Itoa(rec.ID), "webhook-secret", `{"message":{"message_id":2,"from":{"id":88888},"chat":{"id":88888,"type":"private"},"text":"Follow up"}}`)
	handleTelegramWebhook(r)
	if r.RequestCtx.Response.StatusCode() != 200 || calls != before {
		t.Fatalf("avatar fetched for an existing conversation: status=%d calls=%d", r.RequestCtx.Response.StatusCode(), calls-before)
	}
}

func TestTelegramCallbackDelivery(t *testing.T) {
	app, db, rec := newTelegramIntegrationApp(t)
	var callbacks int
	fail := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/answerCallbackQuery") {
			callbacks++
			var body map[string]string
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
			}
			if body["callback_query_id"] != "query-1" {
				t.Errorf("callback=%v", body)
			}
			if fail {
				w.WriteHeader(400)
				io.WriteString(w, `{"ok":false,"error_code":400,"description":"query expired"}`)
				return
			}
			io.WriteString(w, `{"ok":true,"result":true}`)
			return
		}
		io.WriteString(w, `{"ok":true,"result":{"photos":[]}}`)
	}))
	defer server.Close()
	app.telegramClient.SetBaseURL(server.URL)
	for _, body := range []string{
		`{"callback_query":{"id":"query-1","data":"Billing","from":{"id":42},"message":{"message_id":1,"from":{"id":123,"is_bot":true},"chat":{"id":42,"type":"private"}}}}`,
		`{"callback_query":{"id":"query-1","data":"Billing","from":{"id":42},"message":{"message_id":1,"chat":{"id":42,"type":"private"}}}}`,
		`{"callback_query":{"id":"empty","data":""}}`,
	} {
		r := telegramTestRequest(app, strconv.Itoa(rec.ID), "webhook-secret", body)
		if err := handleTelegramWebhook(r); err != nil {
			t.Fatal(err)
		}
		if r.RequestCtx.Response.StatusCode() != 200 {
			t.Fatalf("status=%d", r.RequestCtx.Response.StatusCode())
		}
		fail = true
	}
	var count int
	if err := db.Get(&count, `SELECT COUNT(*) FROM conversation_messages WHERE source_id=$1 AND content='Billing' AND meta->>'telegram_callback'='true'`, fmt.Sprintf("telegram:%d::42:callback:query-1", rec.ID)); err != nil {
		t.Fatal(err)
	}
	if count != 1 || callbacks != 2 {
		t.Fatalf("messages=%d acknowledgements=%d", count, callbacks)
	}
}

func TestTelegramAlbumsAndFormattedEdits(t *testing.T) {
	app, db, rec := newTelegramIntegrationApp(t)
	cfg := telegramChannel.Config{BotToken: "123:secret"}
	var msg telegram.Message
	json.Unmarshal([]byte(`{"message_id":1,"media_group_id":"album","from":{"id":42},"chat":{"id":42,"type":"private"},"text":"Hello","entities":[{"type":"bold","offset":0,"length":5}]}`), &msg)
	if err := ingestTelegramMessage(t.Context(), app, rec, cfg, msg); err != nil {
		t.Fatal(err)
	}
	var row struct{ UUID, Content, Text, Kind, Album string }
	if err := db.Get(&row, `SELECT uuid,content,text_content AS text,content_type AS kind,meta->>'telegram_media_group_id' AS album FROM conversation_messages WHERE source_id=$1`, telegram.SourceID(rec.ID, 42, 1)); err != nil {
		t.Fatal(err)
	}
	if row.Content != "<b>Hello</b>" || row.Kind != "html" || row.Album != "album" || !strings.Contains(row.Text, "Hello") {
		t.Fatalf("row=%+v", row)
	}
	sourceIDs := []string{telegram.SourceID(rec.ID, 42, 10), telegram.SourceID(rec.ID, 42, 11)}
	if err := app.conversation.RecordTelegramSend(row.UUID, sourceIDs); err != nil {
		t.Fatal(err)
	}
	var stored struct {
		Source string
		Meta   json.RawMessage
	}
	if err := db.Get(&stored, `SELECT source_id AS source,meta FROM conversation_messages WHERE uuid=$1`, row.UUID); err != nil {
		t.Fatal(err)
	}
	var meta struct {
		Sources []string `json:"telegram_source_ids"`
		Album   string   `json:"telegram_media_group_id"`
	}
	json.Unmarshal(stored.Meta, &meta)
	if stored.Source != sourceIDs[0] || len(meta.Sources) != 2 || meta.Sources[1] != sourceIDs[1] || meta.Album != "album" {
		t.Fatalf("stored=%+v meta=%+v", stored, meta)
	}
	if err := app.conversation.UpdateTelegramMessageContent(rec.ID, "42", sourceIDs[1], "<i>Corrected</i>", "Corrected", "html", 100); err != nil {
		t.Fatal(err)
	}
	if err := db.Get(&row, `SELECT uuid,content,text_content AS text,content_type AS kind,meta->>'telegram_media_group_id' AS album FROM conversation_messages WHERE uuid=$1`, row.UUID); err != nil {
		t.Fatal(err)
	}
	if row.Content != "<i>Corrected</i>" || row.Text != "Corrected" || row.Kind != "html" {
		t.Fatalf("edit=%+v", row)
	}
	for _, input := range []struct {
		uuid string
		ids  []string
	}{{row.UUID, nil}, {"", sourceIDs}} {
		if err := app.conversation.RecordTelegramSend(input.uuid, input.ids); err != nil {
			t.Fatal(err)
		}
	}
	db.Close()
	if err := app.conversation.RecordTelegramSend(row.UUID, sourceIDs); err == nil {
		t.Fatal("database failure ignored")
	}
}

func TestTelegramNativeRatings(t *testing.T) {
	app, db, rec := newTelegramIntegrationApp(t)
	var msg telegram.Message
	json.Unmarshal([]byte(`{"message_id":1,"from":{"id":42},"chat":{"id":42,"type":"private"},"text":"Hello"}`), &msg)
	if err := ingestTelegramMessage(t.Context(), app, rec, telegramChannel.Config{}, msg); err != nil {
		t.Fatal(err)
	}
	source := telegram.SourceID(rec.ID, 42, 1)
	db.MustExec(`INSERT INTO csat_responses (conversation_id) SELECT conversation_id FROM conversation_messages WHERE source_id=$1`, source)
	db.MustExec(`UPDATE conversation_messages m SET type='outgoing',meta=jsonb_build_object('is_csat',true,'csat_uuid',r.uuid::text) FROM csat_responses r WHERE r.conversation_id=m.conversation_id AND m.source_id=$1`, source)
	deliver := func(chat, owner int64, business, data string) {
		t.Helper()
		body := fmt.Sprintf(`{"callback_query":{"id":"rating","data":%q,"from":{"id":%d},"message":{"message_id":1,"business_connection_id":%q,"chat":{"id":%d,"type":"private"}}}}`, data, owner, business, chat)
		r := telegramTestRequest(app, strconv.Itoa(rec.ID), "webhook-secret", body)
		if err := handleTelegramWebhook(r); err != nil {
			t.Fatal(err)
		}
		if r.RequestCtx.Response.StatusCode() != 200 {
			t.Fatalf("status=%d body=%s", r.RequestCtx.Response.StatusCode(), r.RequestCtx.Response.Body())
		}
	}
	for _, data := range []string{telegram.RatingCallbackPrefix + "0", telegram.RatingCallbackPrefix + "6", telegram.RatingCallbackPrefix + "bad"} {
		deliver(42, 42, "", data)
	}
	deliver(43, 43, "", telegram.RatingCallbackPrefix+"5")
	deliver(42, 77, "business", telegram.RatingCallbackPrefix+"5")
	var count int
	db.Get(&count, `SELECT COUNT(*) FROM csat_responses WHERE response_timestamp IS NOT NULL`)
	if count != 0 {
		t.Fatal("invalid or another chat's rating accepted")
	}
	deliver(42, 42, "", telegram.RatingCallbackPrefix+"4")
	deliver(42, 42, "", telegram.RatingCallbackPrefix+"1")
	var score int
	if err := db.Get(&score, `SELECT rating FROM csat_responses`); err != nil {
		t.Fatal(err)
	}
	if score != 4 {
		t.Fatalf("rating was overwritten: %d", score)
	}
	db.Get(&count, `SELECT COUNT(*) FROM conversation_messages`)
	if count != 1 {
		t.Fatal("rating created a conversation reply")
	}
	db.Close()
	if err := app.conversation.SubmitTelegramRating(source, 5); err == nil {
		t.Fatal("database failure ignored")
	}
}

func TestTelegramContactAttributes(t *testing.T) {
	app, db, rec := newTelegramIntegrationApp(t)
	var msg telegram.Message
	json.Unmarshal([]byte(`{"message_id":1,"from":{"id":42,"username":"customer","language_code":"hi"},"chat":{"id":42,"type":"private"},"text":"Hello"}`), &msg)
	if err := ingestTelegramMessage(t.Context(), app, rec, telegramChannel.Config{}, msg); err != nil {
		t.Fatal(err)
	}
	var attrs json.RawMessage
	if err := db.Get(&attrs, `SELECT custom_attributes FROM users WHERE id=(SELECT contact_id FROM contact_channel_identities WHERE channel='telegram' AND identifier='42')`); err != nil {
		t.Fatal(err)
	}
	var parsed map[string]any
	json.Unmarshal(attrs, &parsed)
	if parsed["telegram_username"] != "customer" || parsed["telegram_language"] != "hi" {
		t.Fatalf("attributes=%s", attrs)
	}
	db.MustExec(`CREATE OR REPLACE FUNCTION reject_telegram_attribute_test() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'test failure'; END $$`)
	db.MustExec(`CREATE TRIGGER reject_telegram_attribute_test BEFORE UPDATE ON users FOR EACH ROW EXECUTE FUNCTION reject_telegram_attribute_test()`)
	t.Cleanup(func() { db.Exec(`DROP TRIGGER IF EXISTS reject_telegram_attribute_test ON users`) })
	msg.ID++
	if err := ingestTelegramMessage(t.Context(), app, rec, telegramChannel.Config{}, msg); err != nil {
		t.Fatal("unchanged attributes were written:", err)
	}
	msg.ID++
	msg.From.Username = "renamed"
	if err := ingestTelegramMessage(t.Context(), app, rec, telegramChannel.Config{}, msg); err == nil {
		t.Fatal("attribute storage failure ignored")
	}
}

func TestTelegramCSATRequest(t *testing.T) {
	app, db, rec := newTelegramIntegrationApp(t)
	var msg telegram.Message
	json.Unmarshal([]byte(`{"message_id":1,"from":{"id":42},"chat":{"id":42,"type":"private"},"text":"Hello"}`), &msg)
	if err := ingestTelegramMessage(t.Context(), app, rec, telegramChannel.Config{}, msg); err != nil {
		t.Fatal(err)
	}
	var conversationID, actorID int
	db.Get(&conversationID, `SELECT id FROM conversations WHERE inbox_id=$1`, rec.ID)
	db.Get(&actorID, `SELECT id FROM users WHERE email='System'`)
	db.MustExec(`UPDATE settings SET value='"https://desk.example.com"' WHERE key='app.root_url'`)
	conv, err := app.conversation.GetConversation(conversationID, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := app.conversation.SendCSATReply(actorID, conv); err != nil {
		t.Fatal(err)
	}
	var raw json.RawMessage
	if err := db.Get(&raw, `SELECT meta FROM conversation_messages WHERE conversation_id=$1 AND type='outgoing'`, conversationID); err != nil {
		t.Fatal(err)
	}
	var meta struct {
		IsCSAT   bool   `json:"is_csat"`
		UUID     string `json:"csat_uuid"`
		Telegram struct {
			Buttons []telegram.Button `json:"buttons"`
		} `json:"telegram"`
	}
	if err := json.Unmarshal(raw, &meta); err != nil {
		t.Fatal(err)
	}
	if !meta.IsCSAT || meta.UUID == "" || len(meta.Telegram.Buttons) != 5 {
		t.Fatalf("meta=%s", raw)
	}
	if err := app.conversation.SendCSATReply(actorID, conv); err != nil {
		t.Fatal(err)
	}
	var count int
	db.Get(&count, `SELECT COUNT(*) FROM conversation_messages WHERE conversation_id=$1 AND type='outgoing'`, conversationID)
	if count != 1 {
		t.Fatal("duplicate survey sent")
	}
}

func newTelegramIntegrationApp(t *testing.T) (*App, *sqlx.DB, imodels.Inbox) {
	t.Helper()
	app, db := newInboxApp(t)
	app.ctx = t.Context()
	app.consts.Store(&constants{MaxFileUploadSizeMB: 100})
	var err error
	app.telegramClient = telegram.New()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, `{"ok":true,"result":{"photos":[]}}`) }))
	t.Cleanup(server.Close)
	app.telegramClient.SetBaseURL(server.URL)
	app.user, err = user.New(app.i18n, user.Opts{DB: db, Lo: app.lo})
	if err != nil {
		t.Fatal(err)
	}
	db.MustExec(`INSERT INTO users (type,email,first_name,last_name) VALUES ('agent','System','System','') ON CONFLICT DO NOTHING`)
	store, _ := localfs.New(localfs.Opts{UploadPath: t.TempDir(), UploadURI: "/uploads", RootURL: func() string { return "http://localhost" }})
	app.media, err = media.New(media.Opts{DB: db, Lo: app.lo, I18n: app.i18n, Store: store, RootURL: func() string { return "http://localhost" }})
	if err != nil {
		t.Fatal(err)
	}
	app.automation, err = automation.New(automation.Opts{DB: db, Lo: app.lo, I18n: app.i18n})
	if err != nil {
		t.Fatal(err)
	}
	app.webhook, err = webhook.New(webhook.Opts{DB: db, Lo: app.lo, I18n: app.i18n})
	if err != nil {
		t.Fatal(err)
	}
	app.sla, err = sla.New(sla.Opts{DB: db, Lo: app.lo, I18n: app.i18n}, nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	app.csat, err = csat.New(csat.Opts{DB: db, Lo: app.lo, I18n: app.i18n})
	if err != nil {
		t.Fatal(err)
	}
	app.setting, err = setting.New(setting.Opts{DB: db, Lo: app.lo})
	if err != nil {
		t.Fatal(err)
	}
	app.tmpl, err = tmpl.New(app.lo, db, nil, nil, nil, app.i18n)
	if err != nil {
		t.Fatal(err)
	}
	app.wsHub = ws.NewHub(app.lo, app.user)
	app.conversation, err = conversation.New(app.wsHub, app.i18n, app.sla, nil, nil, app.inbox, app.user, nil, app.media, app.setting, app.csat, app.automation, app.tmpl, app.webhook, nil, conversation.Opts{DB: db, Lo: app.lo})
	if err != nil {
		t.Fatal(err)
	}
	rec, err := app.inbox.Create(imodels.Inbox{Name: "Telegram integration", Channel: "telegram", Enabled: true, ReopenWindowHours: 48, Config: json.RawMessage(`{"bot_token":"123:secret","secret_token":"webhook-secret"}`)})
	if err != nil {
		t.Fatal(err)
	}
	return app, db, rec
}

func TestTelegramCardsAndReplyReferences(t *testing.T) {
	app, db, rec := newTelegramIntegrationApp(t)
	var message telegram.Message
	if err := json.Unmarshal([]byte(`{"message_id":1,"from":{"id":42},"chat":{"id":42,"type":"private"},"contact":{"first_name":"Shared","last_name":"Contact","phone_number":"+91 90000 00000"}}`), &message); err != nil {
		t.Fatal(err)
	}
	if err := ingestTelegramMessage(t.Context(), app, rec, telegramChannel.Config{}, message); err != nil {
		t.Fatal(err)
	}
	var contactJSON, firstUUID, convUUID string
	if err := db.QueryRow(`SELECT meta->'telegram_contact',uuid, (SELECT uuid FROM conversations WHERE id=conversation_id) FROM conversation_messages WHERE source_id=$1`, message.SourceID(rec.ID)).Scan(&contactJSON, &firstUUID, &convUUID); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(contactJSON, "90000") {
		t.Fatal(contactJSON)
	}
	var wg sync.WaitGroup
	ids := make(chan int, 10)
	for range 10 {
		wg.Go(func() {
			id, err := app.user.SaveSharedContact("Shared", "Contact", "+91 90000 00000")
			if err != nil {
				t.Error(err)
				return
			}
			ids <- id
		})
	}
	wg.Wait()
	close(ids)
	var first int
	for id := range ids {
		if first == 0 {
			first = id
		}
		if first != id {
			t.Fatal("duplicate contacts")
		}
	}
	id, err := app.user.SaveSharedContact("Different", "Name", "919000000000")
	if err != nil || id != first {
		t.Fatalf("reused contact=%d %v", id, err)
	}
	var name string
	db.Get(&name, `SELECT first_name FROM users WHERE id=$1`, id)
	if name != "Shared" {
		t.Fatal("existing name overwritten")
	}
	if _, err := app.user.SaveSharedContact("", "", "no number"); err == nil {
		t.Fatal("empty phone accepted")
	}
	message.ID = 2
	message.Contact = nil
	message.Location = &telegram.Location{Latitude: 12.97, Longitude: 77.59}
	message.ReplyTo = &telegram.Message{ID: 1, Text: "Shared contact"}
	if err := ingestTelegramMessage(t.Context(), app, rec, telegramChannel.Config{}, message); err != nil {
		t.Fatal(err)
	}
	var target, location string
	db.QueryRow(`SELECT meta->'reply_to'->>'uuid',meta->'telegram_location' FROM conversation_messages WHERE source_id=$1`, message.SourceID(rec.ID)).Scan(&target, &location)
	if target != firstUUID || !strings.Contains(location, "12.97") {
		t.Fatalf("target=%s location=%s", target, location)
	}
	foreignID := telegram.SourceID(rec.ID, 42, 99)
	if err := app.conversation.RecordTelegramSend(firstUUID, []string{telegram.SourceID(rec.ID, 42, 1), foreignID}); err != nil {
		t.Fatal(err)
	}
	target, err = app.conversation.GetTelegramReplyUUID(foreignID, convUUID)
	if err != nil || target != firstUUID {
		t.Fatalf("album reference=%s %v", target, err)
	}
	target, err = app.conversation.GetTelegramReplyUUID(foreignID, "00000000-0000-0000-0000-000000000000")
	if err != nil || target != "" {
		t.Fatal("cross-conversation reference")
	}
	db.Close()
	if _, err := app.conversation.GetTelegramReplyUUID(foreignID, convUUID); err == nil {
		t.Fatal("database failure ignored")
	}
	if _, err := app.user.SaveSharedContact("", "", "12345"); err == nil {
		t.Fatal("database failure ignored")
	}
}

func TestTelegramCSATFeedbackAfterRating(t *testing.T) {
	app, db, rec := newTelegramIntegrationApp(t)
	var message telegram.Message
	json.Unmarshal([]byte(`{"message_id":1,"from":{"id":42},"chat":{"id":42,"type":"private"},"text":"Hello"}`), &message)
	if err := ingestTelegramMessage(t.Context(), app, rec, telegramChannel.Config{}, message); err != nil {
		t.Fatal(err)
	}
	source := message.SourceID(rec.ID)
	db.MustExec(`INSERT INTO csat_responses (conversation_id) SELECT conversation_id FROM conversation_messages WHERE source_id=$1`, source)
	db.MustExec(`UPDATE conversation_messages m SET type='outgoing',meta=jsonb_build_object('is_csat',true,'csat_uuid',r.uuid::text) FROM csat_responses r WHERE r.conversation_id=m.conversation_id AND m.source_id=$1`, source)
	var uuid string
	db.Get(&uuid, `SELECT uuid FROM csat_responses`)
	if err := app.conversation.SubmitTelegramRating(source, 4); err != nil {
		t.Fatal(err)
	}
	survey, err := app.csat.Get(uuid)
	if err != nil || !survey.FeedbackPending() {
		t.Fatal("feedback unavailable after rating")
	}
	if err := app.csat.UpdateResponse(uuid, 1, "", nil); err == nil {
		t.Fatal("rating overwrite allowed")
	}
	if err := app.csat.UpdateResponse(uuid, 1, "Helpful reply", nil); err != nil {
		t.Fatal(err)
	}
	survey, err = app.csat.Get(uuid)
	if err != nil || survey.Rating != 4 || survey.Feedback.String != "Helpful reply" || survey.FeedbackPending() {
		t.Fatalf("survey=%+v %v", survey, err)
	}
	if err := app.csat.UpdateResponse(uuid, 5, "Overwrite", nil); err == nil {
		t.Fatal("feedback overwritten")
	}
	if err := app.conversation.SubmitTelegramRating(source, 5); err != nil {
		t.Fatal(err)
	}
	survey, _ = app.csat.Get(uuid)
	if survey.Rating != 4 {
		t.Fatal("callback overwrote completed survey")
	}
	db.MustExec(`UPDATE settings SET value='"https://new.example.com"' WHERE key='app.root_url'`)
	buttons, err := app.conversation.TelegramCSATButtons(uuid)
	if err != nil || len(buttons) != 6 || buttons[5].URL != "https://new.example.com/csat/"+uuid {
		t.Fatalf("buttons=%+v %v", buttons, err)
	}
}

func TestTelegramReplyHTTPContract(t *testing.T) {
	app, db, rec := newTelegramIntegrationApp(t)
	app.authz, _ = authz.NewEnforcer(app.lo, app.i18n)
	var message telegram.Message
	json.Unmarshal([]byte(`{"message_id":1,"from":{"id":42},"chat":{"id":42,"type":"private"},"text":"Question"}`), &message)
	if err := ingestTelegramMessage(t.Context(), app, rec, telegramChannel.Config{}, message); err != nil {
		t.Fatal(err)
	}
	var actorID int
	db.Get(&actorID, `SELECT id FROM users WHERE email='System'`)
	db.MustExec(`INSERT INTO roles (name,permissions) VALUES ('Telegram writer',ARRAY['conversations:read','conversations:read_all','messages:write'])`)
	db.MustExec(`INSERT INTO user_roles (user_id,role_id) SELECT $1,id FROM roles WHERE name='Telegram writer'`, actorID)
	var uuid, cuuid string
	db.QueryRow(`SELECT m.uuid,c.uuid FROM conversation_messages m JOIN conversations c ON c.id=m.conversation_id WHERE m.source_id=$1`, message.SourceID(rec.ID)).Scan(&uuid, &cuuid)
	for _, tc := range []struct {
		name, target string
		buttons      []telegram.Button
		status       int
	}{
		{name: "quoted buttons", target: uuid, buttons: []telegram.Button{{Text: "Yes", Data: "yes"}}, status: 200},
		{name: "missing quote", target: "00000000-0000-0000-0000-000000000000", status: 400},
		{name: "invalid link", buttons: []telegram.Button{{Text: "Unsafe", URL: "javascript:alert(1)"}}, status: 400},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := &fasthttp.RequestCtx{}
			ctx.SetUserValue("user", amodels.User{ID: actorID})
			ctx.SetUserValue("cuuid", cuuid)
			ctx.Request.Header.SetContentType("application/json")
			body, _ := json.Marshal(messageReq{SenderType: "agent", Message: "Reply", ReplyToMessageUUID: tc.target, TelegramButtons: tc.buttons})
			ctx.Request.SetBody(body)
			if err := handleSendMessage(&fastglue.Request{RequestCtx: ctx, Context: app}); err != nil {
				t.Fatal(err)
			}
			if ctx.Response.StatusCode() != tc.status {
				t.Fatalf("status=%d body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
			}
			if tc.status == 200 && !strings.Contains(string(ctx.Response.Body()), "reply_to") {
				t.Fatal("quote metadata missing")
			}
		})
	}
}
