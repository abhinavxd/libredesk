package telegram

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/abhinavxd/libredesk/internal/attachment"
	"github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/telegram"
	"github.com/zerodha/logf"
)

type sourceStore struct {
	uuid, source string
	sources      []string
	err          error
}

func (s *sourceStore) RecordTelegramSend(uuid string, sources []string) error {
	source := sources[0]
	s.uuid, s.source, s.sources = uuid, source, sources
	return s.err
}

func TestChannelLifecycle(t *testing.T) {
	opts := Opts{ID: 7, Name: "Support", Config: Config{BotToken: "token", SecretToken: "secret"}, Client: telegram.New(), SourceUpdater: &sourceStore{}, Lo: testLogger()}
	for _, remove := range []func(*Opts){func(o *Opts) { o.Config.BotToken = "" }, func(o *Opts) { o.Config.SecretToken = "" }, func(o *Opts) { o.Client = nil }, func(o *Opts) { o.SourceUpdater = nil }, func(o *Opts) { o.Lo = nil }} {
		invalid := opts
		remove(&invalid)
		if _, err := New(invalid); err == nil {
			t.Fatal("missing dependency accepted")
		}
	}
	channel, err := New(opts)
	if err != nil {
		t.Fatal(err)
	}
	if channel.Identifier() != 7 || channel.Name() != "Support" || channel.Channel() != ChannelTelegram {
		t.Fatal("incorrect channel identity")
	}
	if channel.FromAddress() != "" || channel.ReplyToAddress() != "" || channel.FromNameTemplate() != "" {
		t.Fatal("email metadata must be empty")
	}
	if err := channel.Receive(t.Context()); err != nil {
		t.Fatal(err)
	}
	if err := channel.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestSendValidation(t *testing.T) {
	channel, _ := New(Opts{Config: Config{BotToken: "token", SecretToken: "secret"}, Client: telegram.New(), SourceUpdater: &sourceStore{}, Lo: testLogger()})
	for _, tc := range []struct {
		name    string
		message models.OutboundMessage
	}{
		{"invalid metadata", models.OutboundMessage{Meta: json.RawMessage(`{`)}},
		{"no recipient", models.OutboundMessage{Meta: json.RawMessage(`{}`)}},
		{"zero recipient", models.OutboundMessage{Meta: json.RawMessage(`{"telegram":{"chat_id":0}}`)}},
		{"negative recipient", models.OutboundMessage{Meta: json.RawMessage(`{"telegram":{"chat_id":-1}}`)}},
		{"empty message", models.OutboundMessage{Meta: json.RawMessage(`{"telegram":{"chat_id":1}}`), Content: "  "}},
		{"text limit", models.OutboundMessage{Meta: json.RawMessage(`{"telegram":{"chat_id":1}}`), Content: strings.Repeat("😀", telegram.MaxTextLength+1)}},
		{"caption limit", models.OutboundMessage{Meta: json.RawMessage(`{"telegram":{"chat_id":1}}`), Content: strings.Repeat("x", telegram.MaxCaptionLength+1), Attachments: attachment.Attachments{{}}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := channel.Send(tc.message); err == nil {
				t.Fatal("invalid message accepted")
			}
		})
	}
}

func TestSendContentAndFailureHandling(t *testing.T) {
	for _, tc := range []struct {
		name, content, text, contentType, response, want string
		status                                           int
		storeError, authHook                             bool
	}{
		{name: "HTML converted", content: "<p>Hello <b>there</b></p>", contentType: models.ContentTypeHTML, response: `{"ok":true,"result":{"message_id":9}}`, want: "Hello <b>there</b>", status: 200, authHook: true},
		{name: "text preferred", content: "ignored", text: "plain", response: `{"ok":true,"result":{"message_id":9}}`, want: "plain", status: 200},
		{name: "plain content", content: "<literal>", contentType: models.ContentTypeText, response: `{"ok":true,"result":{"message_id":9}}`, want: "<literal>", status: 200},
		{name: "source store failure does not resend", content: "hi", response: `{"ok":true,"result":{"message_id":9}}`, want: "hi", status: 200, storeError: true},
		{name: "unauthorized", content: "hi", response: `{"ok":false,"error_code":401,"description":"Unauthorized"}`, want: "hi", status: 401, authHook: true},
		{name: "blocked", content: "hi", response: `{"ok":false,"error_code":403,"description":"Blocked"}`, want: "hi", status: 403, authHook: true},
		{name: "invalid response", content: "hi", response: `not JSON`, want: "hi", status: 200},
		{name: "missing provider id", content: "hi", response: `{"ok":true,"result":{}}`, want: "hi", status: 200},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				var body map[string]any
				json.NewDecoder(r.Body).Decode(&body)
				if body["text"] != tc.want || body["chat_id"] != float64(123) {
					t.Errorf("body=%v", body)
				}
				w.WriteHeader(tc.status)
				io.WriteString(w, tc.response)
			}))
			defer server.Close()
			client := telegram.New()
			client.SetBaseURL(server.URL)
			store := &sourceStore{}
			if tc.storeError {
				store.err = errors.New("database offline")
			}
			opts := Opts{ID: 7, Config: Config{BotToken: "token", SecretToken: "secret"}, Client: client, SourceUpdater: store, Lo: testLogger()}
			var auth []bool
			if tc.authHook {
				opts.AuthStatus = func(id int, ok bool) {
					if id != 7 {
						t.Errorf("wrong inbox %d", id)
					}
					auth = append(auth, ok)
				}
			}
			channel, _ := New(opts)
			err := channel.Send(models.OutboundMessage{UUID: "uuid", Meta: json.RawMessage(`{"telegram":{"chat_id":123}}`), Content: tc.content, TextContent: tc.text, ContentType: tc.contentType})
			success := strings.Contains(tc.response, `"message_id":9`)
			if (err == nil) != success {
				t.Fatalf("err=%v", err)
			}
			if calls != 1 {
				t.Fatalf("must not duplicate a send: %d calls", calls)
			}
			if success && (store.uuid != "uuid" || store.source != "telegram:7:123:9") {
				t.Errorf("wrong source: %+v", store)
			}
			if tc.authHook && tc.status == 401 && (len(auth) != 1 || auth[0]) {
				t.Errorf("auth=%v", auth)
			}
			if tc.authHook && success && (len(auth) != 1 || !auth[0]) {
				t.Errorf("auth=%v", auth)
			}
			if tc.status == 403 && len(auth) != 0 {
				t.Error("blocking a chat must not invalidate the bot token")
			}
		})
	}
}

func TestBusinessChannelSend(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		json.NewDecoder(r.Body).Decode(&payload)
		if payload["business_connection_id"] != "business-a" || payload["message_thread_id"] != float64(12) {
			t.Fatalf("routing lost: %v", payload)
		}
		io.WriteString(w, `{"ok":true,"result":{"message_id":9}}`)
	}))
	defer server.Close()
	client := telegram.New()
	client.SetBaseURL(server.URL)
	store := &sourceStore{}
	channel, err := New(Opts{ID: 7, Config: Config{BotToken: "123:secret", SecretToken: "hook"}, Client: client, SourceUpdater: store, Lo: testLogger()})
	if err != nil {
		t.Fatal(err)
	}
	if err := channel.Send(models.OutboundMessage{UUID: "uuid", Content: "Hello", Meta: json.RawMessage(`{"telegram":{"chat_id":123,"business_connection_id":"business-a","message_thread_id":12,"reply_to_message_id":7}}`)}); err != nil {
		t.Fatal(err)
	}
	if store.source != telegram.BusinessSourceID(7, "business-a", 123, 9) {
		t.Fatal("business source not scoped")
	}
}

func TestChannelAlbums(t *testing.T) {
	for _, tc := range []struct {
		name, response string
		wantErr        bool
	}{
		{"success", `{"ok":true,"result":[{"message_id":9},{"message_id":10}]}`, false},
		{"partial result", `{"ok":true,"result":[{"message_id":9}]}`, true},
		{"missing id", `{"ok":true,"result":[{"message_id":9},{"message_id":0}]}`, true},
		{"failure", `{"ok":false,"error_code":400,"description":"Bad request"}`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, tc.response) }))
			defer server.Close()
			client := telegram.New()
			client.SetBaseURL(server.URL)
			store := &sourceStore{}
			channel, _ := New(Opts{ID: 7, Config: Config{BotToken: "token", SecretToken: "secret"}, Client: client, SourceUpdater: store, Lo: testLogger()})
			err := channel.Send(models.OutboundMessage{UUID: "uuid", Meta: json.RawMessage(`{"telegram":{"chat_id":123,"business_connection_id":"business"}}`), Attachments: attachment.Attachments{{Name: "1.txt"}, {Name: "2.txt"}}})
			if (err != nil) != tc.wantErr {
				t.Fatalf("err=%v", err)
			}
			if !tc.wantErr && (len(store.sources) != 2 || store.sources[0] != telegram.BusinessSourceID(7, "business", 123, 9) || store.sources[1] != telegram.BusinessSourceID(7, "business", 123, 10)) {
				t.Fatalf("sources=%v", store.sources)
			}
		})
	}
}

func testLogger() *logf.Logger { lo := logf.New(logf.Opts{Writer: io.Discard}); return &lo }

func TestCSATButtonsResolvedAtDelivery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			ReplyMarkup telegram.Keyboard `json:"reply_markup"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if len(body.ReplyMarkup.Rows) != 1 || body.ReplyMarkup.Rows[0][0].URL != "https://current.example/csat/survey" {
			t.Errorf("keyboard=%+v", body.ReplyMarkup)
		}
		io.WriteString(w, `{"ok":true,"result":{"message_id":1}}`)
	}))
	defer server.Close()
	client := telegram.New()
	client.SetBaseURL(server.URL)
	message := models.OutboundMessage{Content: "Survey", Meta: json.RawMessage(`{"telegram":{"chat_id":42,"csat_uuid":"survey"}}`)}
	for _, tc := range []struct {
		name     string
		resolver func(string) ([]telegram.Button, error)
		fail     bool
	}{
		{name: "missing resolver", fail: true},
		{name: "settings failure", resolver: func(string) ([]telegram.Button, error) { return nil, errors.New("settings unavailable") }, fail: true},
		{name: "current URL", resolver: func(uuid string) ([]telegram.Button, error) {
			return []telegram.Button{{Text: "Feedback", URL: "https://current.example/csat/" + uuid}}, nil
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			channel, _ := New(Opts{ID: 7, Config: Config{BotToken: "token", SecretToken: "secret"}, Client: client, SourceUpdater: &sourceStore{}, Lo: testLogger(), CSATButtons: tc.resolver})
			if err := channel.Send(message); (err != nil) != tc.fail {
				t.Fatal(err)
			}
		})
	}
}
