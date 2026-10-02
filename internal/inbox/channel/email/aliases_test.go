package email

import (
	"testing"

	imodels "github.com/jakedolan443/fernmail/internal/inbox/models"
)

func testAliases() []imodels.EmailAlias {
	return []imodels.EmailAlias{
		{Address: "director@example.com", Enabled: true, Default: true},
		{Address: "billing@example.com", Enabled: true},
		{Address: "support@example.com", Enabled: true},
	}
}

func TestResolveRecipientAliasPrefersEnvelopeRecipient(t *testing.T) {
	got := resolveRecipientAlias(testAliases(), map[string]string{
		"To":            "Support <support@example.com>",
		"Delivered-To":  "billing@example.com",
		"X-Original-To": "director@example.com",
	})
	if got != "billing@example.com" {
		t.Fatalf("alias = %q, want billing@example.com", got)
	}
}

func TestResolveRecipientAliasStripsPlusAddressing(t *testing.T) {
	got := resolveRecipientAlias(testAliases(), map[string]string{
		"Delivered-To": "support+conv-123@example.com",
	})
	if got != "support@example.com" {
		t.Fatalf("alias = %q, want support@example.com", got)
	}
}

func TestResolveRecipientAliasFallsBackToDefault(t *testing.T) {
	got := resolveRecipientAlias(testAliases(), map[string]string{"To": "unknown@example.net"})
	if got != "director@example.com" {
		t.Fatalf("alias = %q, want director@example.com", got)
	}
}

func TestRetiredAliasRetainsRecipientOwnership(t *testing.T) {
	aliases := testAliases()
	aliases[1].Enabled = false
	if got := resolveRecipientAlias(aliases, map[string]string{"Delivered-To": "billing@example.com", "To": "director@example.com"}); got != "billing@example.com" {
		t.Fatalf("retired alias rerouted to primary: %s", got)
	}
}
