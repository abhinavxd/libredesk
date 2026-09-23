package proactive

import (
	"testing"
	"time"
)

func TestEligibilityAndSuppression(t *testing.T) {
	now := time.Date(2026, 9, 12, 12, 0, 0, 0, time.UTC)
	campaign := Campaign{ID: "a", Enabled: true, Audience: "all", Desktop: true, Mobile: true, BusinessHours: "any", DelaySeconds: 10, Repeat: "once", IncludeURLs: []string{"https://example.com/pricing*"}, ExcludeURLs: []string{"*/private*"}}
	ctx := Context{URL: "https://example.com/pricing?plan=team", Visitor: true, Now: now, ActiveSeconds: 10, SessionKey: "session"}
	tests := []struct {
		name   string
		modify func(*Context)
		want   string
	}{
		{"matches", func(*Context) {}, ""},
		{"delay", func(c *Context) { c.ActiveSeconds = 9 }, "delay"},
		{"wrong host", func(c *Context) { c.URL = "https://example.com.evil/pricing" }, "url"},
		{"excluded", func(c *Context) { c.URL = "https://example.com/pricing/private" }, "excludedUrl"},
		{"script URL", func(c *Context) { c.URL = "javascript:alert(1)" }, "url"},
		{"path pattern ignores query", func(c *Context) { c.URL = "https://example.com/pricing?utm_source=ads" }, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := ctx
			tt.modify(&c)
			if got := campaign.IneligibleReason(c, true, func() bool { return true }); got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
	for pattern, want := range map[string]bool{"/pricing": true, "example.com/pricing": true, "https://example.com/pricing": true, "/pricing?plan=team": true, "/pricing?plan=solo": false, "/pric": false} {
		if got := matchesURL([]string{pattern}, ctx.URL); got != want {
			t.Fatalf("pattern %q: got %v, want %v", pattern, got, want)
		}
	}
	history := []Delivery{{CampaignID: "a", Displayed: true, CreatedAt: now.Add(-48 * time.Hour), SessionKey: "session"}}
	if got := Suppression(campaign, ctx, history, 24*time.Hour); got != "repeat" {
		t.Fatalf("once: %q", got)
	}
	campaign.Repeat = "session"
	if got := Suppression(campaign, ctx, history, 24*time.Hour); got != "repeat" {
		t.Fatalf("session: %q", got)
	}
	ctx.SessionKey = "new"
	if got := Suppression(campaign, ctx, history, 24*time.Hour); got != "" {
		t.Fatalf("new session: %q", got)
	}
	campaign.Repeat = "interval"
	campaign.RepeatHours = 72
	if got := Suppression(campaign, ctx, history, 24*time.Hour); got != "repeat" {
		t.Fatalf("interval: %q", got)
	}
	history[0].CampaignID = "other"
	history[0].CreatedAt = now.Add(-time.Hour)
	if got := Suppression(campaign, ctx, history, 90*time.Minute); got != "cooldown" {
		t.Fatalf("shared cooldown: %q", got)
	}
	history[0].Displayed = false
	if got := Suppression(campaign, ctx, history, 90*time.Minute); got != "" {
		t.Fatalf("expired reservation suppressed: %q", got)
	}
	history[0].CreatedAt = now.Add(-time.Second)
	if got := Suppression(campaign, ctx, history, 10*time.Minute); got != "cooldown" {
		t.Fatalf("concurrent reservation: %q", got)
	}
}

func TestEligibilityGates(t *testing.T) {
	now := time.Date(2026, 9, 12, 12, 0, 0, 0, time.UTC)
	open := func() bool { return true }
	closed := func() bool { return false }
	c := Campaign{Enabled: true, Audience: "users", Desktop: true, BusinessHours: "inside", Event: "checkout"}
	ctx := Context{Now: now, Event: "checkout"}
	if got := c.IneligibleReason(ctx, true, open); got != "" {
		t.Fatal(got)
	}
	ctx.Event = ""
	if got := c.IneligibleReason(ctx, true, open); got != "event" {
		t.Fatal(got)
	}
	ctx.Event = "checkout"
	if got := c.IneligibleReason(ctx, true, closed); got != "businessHours" {
		t.Fatal(got)
	}
	if got := c.IneligibleReason(ctx, false, open); got != "attributes" {
		t.Fatal(got)
	}
	ctx.Visitor = true
	if got := c.IneligibleReason(ctx, true, open); got != "audience" {
		t.Fatal(got)
	}
}

func TestCampaignValidateRequiresTargetDevice(t *testing.T) {
	campaign := Campaign{
		ID:            "8a3660e6-e29b-461c-924f-314c7576f75a",
		Name:          "Pricing invitation",
		Message:       "Need help choosing a plan?",
		Audience:      "all",
		BusinessHours: "any",
		Repeat:        "once",
		RepeatHours:   24,
	}
	if err := campaign.Validate(); err == nil || err.Error() != "device" {
		t.Fatalf("got %v, want device validation error", err)
	}
	campaign.Desktop = true
	if err := campaign.Validate(); err != nil {
		t.Fatalf("valid desktop campaign: %v", err)
	}
}

func TestCooldownAppliesToRepeatingCampaign(t *testing.T) {
	now := time.Date(2026, 9, 12, 12, 0, 0, 0, time.UTC)
	c := Campaign{ID: "a", Enabled: true, Repeat: "interval", RepeatHours: 1}
	ctx := Context{Now: now, SessionKey: "session"}
	history := []Delivery{{CampaignID: "a", Displayed: true, CreatedAt: now.Add(-2 * time.Hour), SessionKey: "session"}}
	if got := Suppression(c, ctx, history, 24*time.Hour); got != "cooldown" {
		t.Fatalf("got %q, want cooldown", got)
	}
	if got := Suppression(c, ctx, history, 90*time.Minute); got != "" {
		t.Fatalf("past both the repeat interval and the cooldown: %q", got)
	}
	if got := Suppression(c, ctx, history, 0); got != "" {
		t.Fatalf("no cooldown configured: %q", got)
	}
	history[0].CreatedAt = now.Add(-30 * time.Minute)
	if got := Suppression(c, ctx, history, 0); got != "repeat" {
		t.Fatalf("inside the repeat interval: %q", got)
	}
}

func TestMatchesURLHostOnlyPattern(t *testing.T) {
	tests := []struct {
		pattern string
		url     string
		want    bool
	}{
		{"shop.example.com", "https://shop.example.com/", true},
		{"shop.example.com", "https://shop.example.com", true},
		{"shop.example.com", "https://shop.example.com/pricing/pro", true},
		{"shop.example.com", "https://shop.example.com:8443/pricing", true},
		{"shop.example.com", "https://other.example.com/pricing", false},
		{"*.example.com", "https://shop.example.com/a/b", true},
		{"*.example.com", "https://example.org/a", false},
		{"shop.example.com/*", "https://shop.example.com/a", true},
		{"shop.example.com/pricing", "https://shop.example.com/pricing", true},
		{"/pricing", "https://shop.example.com/pricing", true},
	}
	for _, tt := range tests {
		if got := matchesURL([]string{tt.pattern}, tt.url); got != tt.want {
			t.Errorf("pattern %q against %q: got %v, want %v", tt.pattern, tt.url, got, tt.want)
		}
	}
}
