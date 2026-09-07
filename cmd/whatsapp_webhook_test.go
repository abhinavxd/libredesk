package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"strings"

	"github.com/abhinavxd/libredesk/internal/inbox"
	"github.com/abhinavxd/libredesk/internal/testdb"
	"github.com/abhinavxd/libredesk/internal/whatsapp"
	"github.com/alicebob/miniredis/v2"
	"github.com/jmoiron/sqlx"
	"github.com/knadh/go-i18n"
	"github.com/redis/go-redis/v9"
	"github.com/zerodha/logf"
)

func TestTextPreview(t *testing.T) {
	tests := []struct {
		name string
		msg  whatsapp.ParsedMessage
		want string
	}{
		{"text wins", whatsapp.ParsedMessage{Type: "text", Text: "hello"}, "hello"},
		{"caption when there is no text", whatsapp.ParsedMessage{Type: "image", Caption: "the box"}, "the box"},
		{"image placeholder", whatsapp.ParsedMessage{Type: "image"}, "[image]"},
		{"video placeholder", whatsapp.ParsedMessage{Type: "video"}, "[video]"},
		{"audio placeholder", whatsapp.ParsedMessage{Type: "audio"}, "[audio]"},
		{"voice note placeholder", whatsapp.ParsedMessage{Type: "voice"}, "[audio]"},
		{"document placeholder", whatsapp.ParsedMessage{Type: "document"}, "[document]"},
		{"sticker placeholder", whatsapp.ParsedMessage{Type: "sticker"}, "[sticker]"},
		{"unsupported explains itself", whatsapp.ParsedMessage{Type: "unsupported"}, "[unsupported message: not delivered by WhatsApp]"},
		{"unknown type falls back", whatsapp.ParsedMessage{Type: "order"}, "[whatsapp message]"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := textPreview(tc.msg); got != tc.want {
				t.Fatalf("expected %q, got %q", tc.want, got)
			}
		})
	}
}

func TestDefaultMediaFilename(t *testing.T) {
	tests := []struct {
		messageType string
		mime        string
		want        string
	}{
		{"image", "image/jpeg", "image.jpeg"},
		{"video", "video/mp4", "video.mp4"},
		{"audio", "audio/ogg; codecs=opus", "audio.ogg"},
		{"voice", "audio/ogg", "audio.ogg"},
		{"document", "application/pdf", "document.pdf"},
		{"sticker", "image/webp", "sticker.webp"},
		{"unknown", "application/pdf", "attachment.pdf"},
		{"image", "", "image.bin"},
	}
	for _, tc := range tests {
		t.Run(tc.messageType+"/"+tc.mime, func(t *testing.T) {
			if got := defaultMediaFilename(tc.messageType, tc.mime); got != tc.want {
				t.Fatalf("expected %q, got %q", tc.want, got)
			}
		})
	}
}

func TestSplitName(t *testing.T) {
	tests := []struct {
		in    string
		first string
		last  string
	}{
		{"Ravi Kumar", "Ravi", "Kumar"},
		{"Ravi", "Ravi", ""},
		{"Ravi Kumar Singh", "Ravi", "Kumar Singh"},
		{"", whatsAppDefaultContactName, ""},
	}
	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			first, last := splitName(tc.in)
			if first != tc.first || last != tc.last {
				t.Fatalf("expected (%q, %q), got (%q, %q)", tc.first, tc.last, first, last)
			}
		})
	}
}

// A permanent error stores a placeholder, anything transient is retried.
func TestIsPermanentMediaError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"404 from meta", &whatsapp.MetaAPIError{StatusCode: http.StatusNotFound}, true},
		{"400 from meta", &whatsapp.MetaAPIError{StatusCode: http.StatusBadRequest}, true},
		{"401 is retryable, a replaced token recovers the media", &whatsapp.MetaAPIError{StatusCode: http.StatusUnauthorized}, false},
		{"403 is retryable", &whatsapp.MetaAPIError{StatusCode: http.StatusForbidden}, false},
		{"408 is retryable", &whatsapp.MetaAPIError{StatusCode: http.StatusRequestTimeout}, false},
		{"429 is retryable", &whatsapp.MetaAPIError{StatusCode: http.StatusTooManyRequests}, false},
		{"500 is retryable", &whatsapp.MetaAPIError{StatusCode: http.StatusInternalServerError}, false},
		{"wrapped meta error", fmt.Errorf("downloading: %w", &whatsapp.MetaAPIError{StatusCode: http.StatusGone}), true},
		{"deadline exceeded is retryable", context.DeadlineExceeded, false},
		{"truncated body is retryable", io.ErrUnexpectedEOF, false},
		{"network error is retryable", &net.DNSError{IsTimeout: true}, false},
		{"decode failure is permanent", errors.New("decoding media info"), true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := isPermanentMediaError(tc.err); got != tc.want {
				t.Fatalf("expected %v, got %v", tc.want, got)
			}
		})
	}
}

func TestLocalPhoneNumber(t *testing.T) {
	app := testI18nApp(t)
	tests := []struct {
		name     string
		phone    string
		dialCode string
		want     string
		wantErr  bool
	}{
		{"bare local number", "9876543210", "91", "9876543210", false},
		{"local number with spaces", "98765 43210", "91", "9876543210", false},
		{"plus prefixed", "+919876543210", "91", "9876543210", false},
		{"plus prefixed with spaces", "+91 98765 43210", "91", "9876543210", false},
		{"double zero prefixed", "00919876543210", "91", "9876543210", false},
		{"plus with the wrong country", "+15550001111", "91", "", true},
		{"empty", "", "91", "", true},
		{"punctuation only", "+ - ", "91", "", true},
		// A local number that happens to start with the dial code must not be trimmed.
		{"local number starting with the dial code", "9198765432", "91", "9198765432", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := localPhoneNumber(app, tc.phone, tc.dialCode)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected an error, got %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("expected %q, got %q", tc.want, got)
			}
		})
	}
}

func TestWhatsAppCallbackURLFromRoot(t *testing.T) {
	tests := []struct {
		root string
		want string
	}{
		{"https://desk.example.com", "https://desk.example.com/webhooks/whatsapp/7"},
		{"https://desk.example.com/", "https://desk.example.com/webhooks/whatsapp/7"},
		{"", ""},
	}
	for _, tc := range tests {
		if got := whatsAppCallbackURLFromRoot(tc.root, 7); got != tc.want {
			t.Errorf("%q: expected %q, got %q", tc.root, tc.want, got)
		}
	}
}

// Meta has to reach the callback URL, so anything local or plain HTTP must not be auto-registered.
func TestIsPublicWebhookURL(t *testing.T) {
	tests := map[string]bool{
		"https://desk.example.com":  true,
		"https://desk.example.com/": true,
		" https://desk.example.com": true,
		"http://desk.example.com":   false,
		"https://localhost:9000":    false,
		"https://127.0.0.1:9000":    false,
		"https://[::1]:9000":        false,
		"https://":                  false,
		"":                          false,
		"not a url":                 false,
	}
	for root, want := range tests {
		if got := isPublicWebhookURL(root); got != want {
			t.Errorf("%q: expected %v, got %v", root, want, got)
		}
	}
}

// testI18nApp carries the real language file, so a renamed i18n key fails the test.
func testI18nApp(t *testing.T) *App {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "i18n", "en-US.json"))
	if err != nil {
		t.Fatalf("reading the language file: %v", err)
	}
	lang, err := i18n.New(raw)
	if err != nil {
		t.Fatalf("loading i18n: %v", err)
	}
	return &App{i18n: lang}
}

// GetAll fails wholesale when any single inbox row cannot be decoded, and reporting that as "no inbox" drops the message.
func TestResolveWhatsAppInboxSeparatesLookupFailureFromUnroutable(t *testing.T) {
	app, db := testInboxApp(t)

	var okID int
	if err := db.QueryRow(`INSERT INTO inboxes (channel, config, "name", enabled, "from")
		VALUES ('whatsapp', '{"phone_number_id":"PN-A"}'::jsonb, 'wa-ok', true, '') RETURNING id`).Scan(&okID); err != nil {
		t.Fatalf("seeding the routable inbox: %v", err)
	}

	// An unknown phone number id with every inbox readable is genuinely unroutable: drop it.
	if _, _, err := resolveWhatsAppInbox(app, okID, "PN-UNKNOWN"); !errors.Is(err, errNoEnabledWhatsAppInbox) {
		t.Fatalf("expected an unroutable message to stay droppable, got %v", err)
	}

	// One undecodable row makes GetAll fail for every caller, and the message is still deliverable on a retry.
	if _, err := db.Exec(`INSERT INTO inboxes (channel, config, "name", enabled, "from")
		VALUES ('whatsapp', '"corrupt"'::jsonb, 'wa-corrupt', true, '')`); err != nil {
		t.Fatalf("seeding the undecodable inbox: %v", err)
	}
	_, _, err := resolveWhatsAppInbox(app, okID, "PN-B")
	if err == nil {
		t.Fatal("expected the lookup failure to surface")
	}
	if errors.Is(err, errNoEnabledWhatsAppInbox) {
		t.Fatal("a failed inbox lookup was reported as unroutable, so the queue acks and drops the customer's message")
	}
}

func testInboxApp(t *testing.T) (*App, *sqlx.DB) {
	app, db, _ := testInboxAppWithRedis(t, "")
	return app, db
}

// addr "" points the app at a redis that is not listening.
func testInboxAppWithRedis(t *testing.T, addr string) (*App, *sqlx.DB, *miniredis.Miniredis) {
	t.Helper()
	var mr *miniredis.Miniredis
	if addr == "" {
		addr = "127.0.0.1:1"
	} else if addr == "live" {
		mr = miniredis.RunT(t)
		addr = mr.Addr()
	}
	app, db := newInboxApp(t)
	app.redis = redis.NewClient(&redis.Options{Addr: addr})
	t.Cleanup(func() { app.redis.Close() })
	return app, db, mr
}

func newInboxApp(t *testing.T) (*App, *sqlx.DB) {
	t.Helper()
	testdb.New(t, "cmd")
	db, err := sqlx.Connect("postgres", strings.Replace(os.Getenv("LIBREDESK_TEST_DB_DSN"), "/libredesk?", "/libredesk_test_cmd?", 1))
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	if _, err := db.Exec(`DELETE FROM inboxes`); err != nil {
		t.Fatalf("clearing inboxes: %v", err)
	}

	lo := logf.New(logf.Opts{Level: logf.FatalLevel})
	raw, err := os.ReadFile(filepath.Join("..", "i18n", "en-US.json"))
	if err != nil {
		t.Fatalf("reading the language file: %v", err)
	}
	lang, err := i18n.New(raw)
	if err != nil {
		t.Fatalf("loading i18n: %v", err)
	}
	mgr, err := inbox.New(&lo, db, lang, "0123456789abcdef0123456789abcdef")
	if err != nil {
		t.Fatalf("inbox.New: %v", err)
	}
	return &App{lo: &lo, i18n: lang, inbox: mgr}, db
}

// An install with no WhatsApp inbox must boot with Redis unreachable.
func TestWhatsAppIngesterStaysOffWithoutAWhatsAppInbox(t *testing.T) {
	app, db, _ := testInboxAppWithRedis(t, "")

	if _, err := db.Exec(`INSERT INTO inboxes (channel, config, "name", enabled, "from")
		VALUES ('email', '{}'::jsonb, 'mail', true, 'a@b.test')`); err != nil {
		t.Fatalf("seeding an email inbox: %v", err)
	}

	if err := ensureWhatsAppIngester(app); err != nil {
		t.Fatalf("an install with no whatsapp inbox must boot with redis down, got %v", err)
	}
	if app.ingester() != nil {
		t.Fatal("expected no ingester workers on an install with no whatsapp inbox")
	}
}

func TestWhatsAppIngesterStartsForAWhatsAppInbox(t *testing.T) {
	app, db, mr := testInboxAppWithRedis(t, "live")

	if err := ensureWhatsAppIngester(app); err != nil {
		t.Fatalf("ensure with no whatsapp inbox: %v", err)
	}
	if app.ingester() != nil {
		t.Fatal("expected the ingester to stay off before a whatsapp inbox exists")
	}

	if _, err := db.Exec(`INSERT INTO inboxes (channel, config, "name", enabled, "from")
		VALUES ('whatsapp', '{"phone_number_id":"PN-A"}'::jsonb, 'wa', true, '')`); err != nil {
		t.Fatalf("seeding a whatsapp inbox: %v", err)
	}
	if err := ensureWhatsAppIngester(app); err != nil {
		t.Fatalf("ensure after the inbox exists: %v", err)
	}
	started := app.ingester()
	if started == nil {
		t.Fatal("expected the ingester to start once a whatsapp inbox exists")
	}
	t.Cleanup(started.Close)
	if !mr.Exists(whatsAppStream) {
		t.Fatal("expected the durable stream to be created")
	}

	// Every inbox save calls this, so it must not replace a running ingester.
	if err := ensureWhatsAppIngester(app); err != nil {
		t.Fatalf("repeat ensure: %v", err)
	}
	if app.ingester() != started {
		t.Fatal("a second call replaced the running ingester")
	}
}
