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

	"github.com/abhinavxd/libredesk/internal/whatsapp"
	"github.com/knadh/go-i18n"
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
