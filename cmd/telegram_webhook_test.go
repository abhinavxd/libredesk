package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	telegramChannel "github.com/abhinavxd/libredesk/internal/inbox/channel/telegram"
	imodels "github.com/abhinavxd/libredesk/internal/inbox/models"
	"github.com/abhinavxd/libredesk/internal/telegram"
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
		{name: "name fallback", payload: `{"document":{"file_id":"file","file_name":"."}}`, body: "test", wantName: "document", wantType: "text/plain; charset=utf-8"},
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
