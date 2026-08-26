package main

import (
	"testing"

	"github.com/abhinavxd/libredesk/internal/testutil"
)

func TestPortalStatusLabel(t *testing.T) {
	lcl := testutil.NewI18n(t)
	tests := []struct {
		category   string
		lastSender string
		wantLabel  string
		wantClass  string
	}{
		{"open", "contact", "In progress", "open"},
		{"open", "", "In progress", "open"},
		{"open", "agent", "Awaiting your reply", "waiting"},
		{"waiting", "agent", "Awaiting your reply", "waiting"},
		{"waiting", "contact", "In progress", "open"},
		{"resolved", "agent", "Resolved", "resolved"},
		{"resolved", "contact", "Resolved", "resolved"},
	}
	for _, tt := range tests {
		label, class := portalStatusLabel(lcl, tt.category, tt.lastSender)
		if label != tt.wantLabel || class != tt.wantClass {
			t.Errorf("portalStatusLabel(%q, %q) = (%q, %q), want (%q, %q)",
				tt.category, tt.lastSender, label, class, tt.wantLabel, tt.wantClass)
		}
	}
}

func TestPortalContactFirstName(t *testing.T) {
	tests := []struct{ email, want string }{
		{"jane@example.com", "jane"},
		{"@example.com", "@example.com"},
	}
	for _, tt := range tests {
		if got := portalContactFirstName(tt.email); got != tt.want {
			t.Errorf("portalContactFirstName(%q) = %q, want %q", tt.email, got, tt.want)
		}
	}
}

func TestPortalReturnTo(t *testing.T) {
	const base = "https://support.example.com"
	tests := []struct{ raw, want string }{
		{"", "/portal"},
		{"/portal", "/portal"},
		{"/portal/tickets/42", "/portal/tickets/42"},
		{"/portal/tickets/new?article=refunds", "/portal/tickets/new?article=refunds"},
		{"  /portal  ", "/portal"},
		{"https://support.example.com/portal/tickets/42", "/portal/tickets/42"},
		// Anything outside /portal is not a portal page, so it is not a redirect target.
		{"/hc/docs/en", "/portal"},
		{"https://support.example.com/hc/docs/en", "/portal"},
		{"/portalish", "/portal"},
		{"//evil.example.com/portal", "/portal"},
		{"/\\evil.example.com", "/portal"},
		{"https://support.example.com.evil.com/portal", "/portal"},
		{"https://evil.example.com/portal", "/portal"},
		{"javascript:alert(1)", "/portal"},
		{"/portal/x\r\nSet-Cookie: a=b", "/portal"},
	}
	for _, tt := range tests {
		if got := portalReturnTo(base, tt.raw); got != tt.want {
			t.Errorf("portalReturnTo(%q) = %q, want %q", tt.raw, got, tt.want)
		}
	}
	if got := portalReturnTo("", "https://support.example.com/portal"); got != "/portal" {
		t.Errorf("portalReturnTo with no base URL = %q, want /portal", got)
	}
}

func TestNormalizePortalEmail(t *testing.T) {
	if got := normalizePortalEmail("  Jane@Example.COM "); got != "jane@example.com" {
		t.Errorf("normalizePortalEmail = %q", got)
	}
}

func TestMatchAcceptLanguage(t *testing.T) {
	allowed := []string{"en-US", "fr", "de"}
	tests := []struct {
		header string
		want   string
		wantOK bool
	}{
		{"fr", "fr", true},
		{"FR", "fr", true},
		{"fr-CA", "fr", true},
		{"en-GB,en;q=0.9", "en-US", true},
		{"de;q=0.2,fr;q=0.9", "fr", true},
		{"de;q=0,fr;q=0.1", "fr", true},
		{"ja,ko;q=0.8", "", false},
		{"", "", false},
		{"*", "", false},
	}
	for _, tt := range tests {
		got, ok := matchAcceptLanguage(tt.header, allowed)
		if got != tt.want || ok != tt.wantOK {
			t.Errorf("matchAcceptLanguage(%q) = (%q, %v), want (%q, %v)", tt.header, got, ok, tt.want, tt.wantOK)
		}
	}
}
