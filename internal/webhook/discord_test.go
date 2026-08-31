package webhook

import (
	"encoding/json"
	"testing"

	"github.com/abhinavxd/libredesk/internal/webhook/models"
)

func TestIsDiscordWebhookURL(t *testing.T) {
	ok := []string{
		"https://discord.com/api/webhooks/123/abc-token",
		"https://discordapp.com/api/webhooks/123/abc_token",
		"https://ptb.discord.com/api/webhooks/123/token",
		"https://canary.discord.com/api/v10/webhooks/9/tok?wait=true",
	}
	for _, u := range ok {
		if !models.IsDiscordWebhookURL(u) {
			t.Errorf("expected Discord URL: %s", u)
		}
	}
	bad := []string{
		"https://example.com/api/webhooks/123/abc",
		"http://discord.com/api/webhooks/123/abc",
		"https://discord.com/api/webhooks/not-a-snowflake/abc",
		"https://evil.com/?https://discord.com/api/webhooks/1/x",
	}
	for _, u := range bad {
		if models.IsDiscordWebhookURL(u) {
			t.Errorf("did not expect Discord URL: %s", u)
		}
	}
}

func TestNormalizeDelivery(t *testing.T) {
	got, err := normalizeDelivery("https://discord.com/api/webhooks/1/t", "")
	if err != nil || got != models.DeliveryDiscord {
		t.Fatalf("auto-detect: got %q err %v", got, err)
	}
	got, err = normalizeDelivery("https://hooks.example.com/x", "")
	if err != nil || got != models.DeliveryHTTP {
		t.Fatalf("http default: got %q err %v", got, err)
	}
	if _, err := normalizeDelivery("https://example.com/hook", models.DeliveryDiscord); err == nil {
		t.Fatal("expected error for non-Discord URL with discord delivery")
	}
}

func TestBuildDiscordPayloadConversationCreated(t *testing.T) {
	raw, err := buildDiscordPayload(DeliveryTask{
		Event: models.EventConversationCreated,
		Payload: map[string]any{
			"uuid":              "conv-1",
			"subject":           "Billing question",
			"status":            "open",
			"inbox_name":        "support@",
			"reference_number":  "SB-100",
			"contact": map[string]any{
				"first_name": "Ada",
				"last_name":  "Lovelace",
				"email":      "ada@example.com",
			},
		},
	}, "https://support.stackblaze.cloud")
	if err != nil {
		t.Fatal(err)
	}
	var p discordPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		t.Fatal(err)
	}
	if p.Username != "Libredesk" || len(p.Embeds) != 1 {
		t.Fatalf("unexpected payload: %+v", p)
	}
	e := p.Embeds[0]
	if e.Title != "New conversation" || e.Description != "Billing question" {
		t.Fatalf("title/desc: %+v", e)
	}
	if e.URL != "https://support.stackblaze.cloud/inboxes/all/conversation/conv-1" {
		t.Fatalf("url: %s", e.URL)
	}
}

func TestBuildDiscordPayloadMessage(t *testing.T) {
	raw, err := buildDiscordPayload(DeliveryTask{
		Event: models.EventMessageCreated,
		Payload: map[string]any{
			"conversation_uuid": "conv-2",
			"text_content":      "Need help with login",
			"private":           false,
			"author":            map[string]any{"first_name": "Ada", "last_name": "Lovelace"},
		},
	}, "")
	if err != nil {
		t.Fatal(err)
	}
	var p discordPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		t.Fatal(err)
	}
	if p.Embeds[0].Description != "Need help with login" {
		t.Fatalf("desc: %s", p.Embeds[0].Description)
	}
}
