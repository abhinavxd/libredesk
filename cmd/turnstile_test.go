package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/abhinavxd/libredesk/internal/inbox/channel/livechat"
)

func TestChatInitNeedsTurnstile(t *testing.T) {
	tests := []struct {
		name      string
		secret    string
		contactID int
		visitor   bool
		want      bool
	}{
		{name: "off when secret is empty", secret: "  ", contactID: 0, visitor: true, want: false},
		{name: "new visitor", secret: "sek", contactID: 0, visitor: true, want: true},
		{name: "returning visitor", secret: "sek", contactID: 4, visitor: true, want: true},
		{name: "signed-in user", secret: "sek", contactID: 4, visitor: false, want: false},
		{name: "no session is a visitor even if the flag is false", secret: "sek", contactID: 0, visitor: false, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := chatInitNeedsTurnstile(tt.secret, tt.contactID, tt.visitor); got != tt.want {
				t.Fatalf("chatInitNeedsTurnstile() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChatSettingsHidesTurnstileSecret(t *testing.T) {
	raw, err := json.Marshal(chatSettingsResponse{
		Config: livechat.Config{
			TurnstileSiteKey:   "site-key",
			TurnstileSecretKey: "secret-key",
			TrustedDomains:     []string{"example.com"},
			Theme:              "light",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatal(err)
	}
	if body["turnstile_site_key"] != "site-key" {
		t.Fatalf("site key = %#v", body["turnstile_site_key"])
	}
	if body["theme"] != "light" {
		t.Fatalf("theme = %#v", body["theme"])
	}
	if _, ok := body["turnstile_secret_key"]; ok {
		t.Fatal("turnstile secret leaked in widget settings")
	}
	if _, ok := body["trusted_domains"]; ok {
		t.Fatal("trusted domains leaked in widget settings")
	}
}

func TestVerifyTurnstileToken(t *testing.T) {
	var gotSecret, gotToken, gotIP string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Errorf("parse form: %v", err)
		}
		gotSecret = r.PostForm.Get("secret")
		gotToken = r.PostForm.Get("response")
		gotIP = r.PostForm.Get("remoteip")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"success": true})
	}))
	t.Cleanup(srv.Close)

	err := verifyTurnstileToken(context.Background(), srv.Client(), srv.URL, "sek", "tok", "203.0.113.8")
	if err != nil {
		t.Fatal(err)
	}
	if gotSecret != "sek" || gotToken != "tok" || gotIP != "203.0.113.8" {
		t.Fatalf("posted secret=%q token=%q ip=%q", gotSecret, gotToken, gotIP)
	}

	if err := verifyTurnstileToken(context.Background(), srv.Client(), srv.URL, "sek", "  ", "203.0.113.8"); !errors.Is(err, errTurnstileRejected) {
		t.Fatalf("empty token error = %v", err)
	}
	if gotToken != "tok" {
		t.Fatal("empty token was sent to siteverify")
	}
}

func TestVerifyTurnstileTokenRejects(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"success": false, "error-codes": []string{"invalid-input-response"}})
	}))
	t.Cleanup(srv.Close)

	err := verifyTurnstileToken(context.Background(), srv.Client(), srv.URL, "sek", "tok", "")
	if !errors.Is(err, errTurnstileRejected) {
		t.Fatalf("error = %v", err)
	}
	if !strings.Contains(err.Error(), "invalid-input-response") {
		t.Fatalf("error = %v, want cloudflare error code", err)
	}
}

func TestVerifyTurnstileTokenUnavailable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	t.Cleanup(srv.Close)

	err := verifyTurnstileToken(context.Background(), srv.Client(), srv.URL, "sek", "tok", "")
	if !errors.Is(err, errTurnstileUnavailable) {
		t.Fatalf("error = %v", err)
	}
}
