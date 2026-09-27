package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	telegramChannel "github.com/abhinavxd/libredesk/internal/inbox/channel/telegram"
	imodels "github.com/abhinavxd/libredesk/internal/inbox/models"
	"github.com/abhinavxd/libredesk/internal/setting"
	"github.com/abhinavxd/libredesk/internal/stringutil"
	"github.com/abhinavxd/libredesk/internal/telegram"
)

func TestPrepareTelegramInbox(t *testing.T) {
	app, db := newInboxApp(t)
	app.ctx = t.Context()
	result := `{"ok":true,"result":{"id":123,"is_bot":true,"username":"support_bot"}}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, result) }))
	defer server.Close()
	app.telegramClient = telegram.New()
	app.telegramClient.SetBaseURL(server.URL)
	for _, tc := range []struct {
		name, config string
		hours        int
		want         string
	}{
		{"invalid JSON", `{`, 48, "configuration"},
		{"wrong config shape", `[]`, 48, "configuration"},
		{"empty token", `{}`, 48, "token"},
		{"malformed token", `{"bot_token":"bad"}`, 48, "token"},
		{"negative hours", `{"bot_token":"123:secret"}`, -1, "hours"},
		{"overflow hours", `{"bot_token":"123:secret"}`, 2147483648, "hours"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := imodels.Inbox{Config: json.RawMessage(tc.config), ReopenWindowHours: tc.hours}
			err := prepareTelegramInbox(t.Context(), app, &rec, 0)
			if err == nil || !strings.Contains(strings.ToLower(err.Error()), tc.want) {
				t.Fatalf("error=%v", err)
			}
		})
	}
	rec := imodels.Inbox{Channel: "telegram", Config: json.RawMessage(`{"bot_token":" 123:secret ","bot_id":999,"secret_token":"client-forged"}`), ReopenWindowHours: 48}
	if err := prepareTelegramInbox(t.Context(), app, &rec, 0); err != nil {
		t.Fatal(err)
	}
	var cfg telegramChannel.Config
	json.Unmarshal(rec.Config, &cfg)
	if cfg.BotToken != "123:secret" || cfg.BotID != 123 || cfg.BotUsername != "support_bot" || cfg.SecretToken == "" || cfg.SecretToken == "client-forged" {
		t.Fatalf("untrusted setup fields accepted: %+v", cfg)
	}
	rec.Name = "test"
	created, err := app.inbox.Create(rec)
	if err != nil {
		t.Fatal(err)
	}
	id := created.ID
	if err := prepareTelegramInbox(t.Context(), app, &rec, 0); err == nil {
		t.Fatal("duplicate bot allowed")
	}
	for _, token := range []string{"", stringutil.PasswordDummy} {
		rec.Config = json.RawMessage(`{"bot_token":"` + token + `","secret_token":"forged"}`)
		if err := prepareTelegramInbox(t.Context(), app, &rec, id); err != nil {
			t.Fatal(err)
		}
		var got telegramChannel.Config
		json.Unmarshal(rec.Config, &got)
		if got.BotToken != cfg.BotToken || got.SecretToken != cfg.SecretToken {
			t.Fatal("edit discarded credentials")
		}
	}
	result = `{"ok":true,"result":{"id":456,"is_bot":true}}`
	if err := prepareTelegramInbox(t.Context(), app, &rec, id); err == nil {
		t.Fatal("changing bot allowed")
	}
	for _, response := range []string{`{"ok":false,"error_code":401,"description":"Unauthorized"}`, `{"ok":true,"result":{"id":123,"is_bot":false}}`, `{"ok":true,"result":{"id":0,"is_bot":true}}`} {
		result = response
		if err := prepareTelegramInbox(t.Context(), app, &rec, 0); err == nil {
			t.Fatal("invalid bot accepted")
		}
	}
	result = `{"ok":true,"result":{"id":123,"is_bot":true}}`
	if err := prepareTelegramInbox(t.Context(), app, &rec, id+1000); err == nil {
		t.Fatal("missing inbox accepted")
	}
	db.MustExec(`UPDATE inboxes SET channel='email' WHERE id=$1`, id)
	if err := prepareTelegramInbox(t.Context(), app, &rec, id); err == nil {
		t.Fatal("channel change accepted")
	}
	db.MustExec(`UPDATE inboxes SET channel='telegram',config='{"bot_id":"invalid"}' WHERE id=$1`, id)
	if err := prepareTelegramInbox(t.Context(), app, &rec, id); err == nil {
		t.Fatal("invalid stored config accepted")
	}
	if err := prepareTelegramInbox(t.Context(), app, &rec, 0); err == nil {
		t.Fatal("invalid other inbox ignored")
	}
	db.MustExec(`UPDATE inboxes SET channel='email',config='{}' WHERE id=$1`, id)
	if err := prepareTelegramInbox(t.Context(), app, &rec, 0); err != nil {
		t.Fatal("other channel blocked setup:", err)
	}
	db.Close()
	if err := prepareTelegramInbox(t.Context(), app, &rec, 0); err == nil {
		t.Fatal("database failure ignored")
	}
}

func TestConfigureTelegramWebhook(t *testing.T) {
	app, db := newInboxApp(t)
	app.ctx = t.Context()
	app.telegramClient = telegram.New()
	var err error
	app.setting, err = setting.New(setting.Opts{DB: db, Lo: app.lo})
	if err != nil {
		t.Fatal(err)
	}
	var paths []string
	fail := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Error(err)
		}
		if strings.HasSuffix(r.URL.Path, "setWebhook") && (payload["url"] != "https://desk.example.com/webhooks/telegram/7" || payload["secret_token"] != "hook-secret") {
			t.Errorf("wrong registration: %v", payload)
		}
		if fail {
			io.WriteString(w, `{"ok":false,"error_code":401,"description":"Unauthorized"}`)
		} else {
			io.WriteString(w, `{"ok":true,"result":true}`)
		}
	}))
	defer server.Close()
	app.telegramClient.SetBaseURL(server.URL)
	rec := imodels.Inbox{ID: 7, Enabled: true, Channel: "telegram", Config: json.RawMessage(`{"bot_token":"123:secret","secret_token":"hook-secret"}`)}
	for _, root := range []string{"", "http://localhost:9000", "http://desk.example.com", "https://127.0.0.1"} {
		db.MustExec(`UPDATE settings SET value=$1 WHERE key='app.root_url'`, `"`+root+`"`)
		configureTelegramWebhook(app, rec)
		if _, ok := app.telegramHookErrors.Load(7); !ok {
			t.Errorf("invalid root %q accepted", root)
		}
	}
	db.MustExec(`UPDATE settings SET value='"https://desk.example.com/"' WHERE key='app.root_url'`)
	configureTelegramWebhook(app, rec)
	if _, ok := app.telegramHookErrors.Load(7); ok {
		t.Fatal("success retained webhook error")
	}
	fail = true
	configureTelegramWebhook(app, rec)
	if _, ok := app.telegramHookErrors.Load(7); !ok {
		t.Fatal("failed registration hidden")
	}
	fail = false
	rec.Enabled = false
	configureTelegramWebhook(app, rec)
	if paths[len(paths)-1] != "/bot123:secret/deleteWebhook" {
		t.Fatal("disabled bot webhook not removed")
	}
	db.MustExec(`INSERT INTO inboxes (id,channel,name,config,enabled) VALUES (7,'telegram','test','{}',true),(8,'email','email','{}',true),(9,'telegram','disabled','{}',false)`)
	db.MustExec(`UPDATE inboxes SET "from"=''`)
	if err := app.inbox.UpdateConfig(7, rec.Config); err != nil {
		t.Fatal(err)
	}
	paths = nil
	reconcileTelegramWebhooks(app)
	if len(paths) != 1 || paths[0] != "/bot123:secret/setWebhook" {
		t.Fatalf("reconcile paths=%v", paths)
	}
	rec.Config = json.RawMessage(`{`)
	configureTelegramWebhook(app, rec)
	if _, ok := app.telegramHookErrors.Load(7); !ok {
		t.Fatal("bad configuration hidden")
	}
	db.Close()
	reconcileTelegramWebhooks(app)
}
