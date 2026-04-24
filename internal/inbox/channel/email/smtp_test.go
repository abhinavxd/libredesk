package email

import (
	"strings"
	"testing"

	"github.com/abhinavxd/libredesk/internal/stringutil"
)

func TestResolveReplyTo(t *testing.T) {
	tests := []struct {
		name             string
		perMessage       string
		inboxReplyTo     string
		fromEmail        string
		conversationUUID string
		plusAddressing   bool
		want             string
	}{
		{
			name:             "per-message override wins over everything",
			perMessage:       "override@example.com",
			inboxReplyTo:     "support@example.com",
			fromEmail:        "support@hedwig.example.com",
			conversationUUID: "abc-123",
			plusAddressing:   true,
			want:             "override@example.com",
		},
		{
			name:             "per-message override is passed through literally with display name",
			perMessage:       "Override Name <override@example.com>",
			inboxReplyTo:     "support@example.com",
			fromEmail:        "support@hedwig.example.com",
			conversationUUID: "abc-123",
			plusAddressing:   true,
			want:             "Override Name <override@example.com>",
		},

		{
			name:             "plus-addressing on + inbox reply_to: base taken from reply_to",
			inboxReplyTo:     "support@example.com",
			fromEmail:        "support@hedwig.example.com",
			conversationUUID: "abc-123",
			plusAddressing:   true,
			want:             "support+conv-abc-123@example.com",
		},
		{
			name:             "plus-addressing on + inbox reply_to with different local part",
			inboxReplyTo:     "replies@example.com",
			fromEmail:        "support@hedwig.example.com",
			conversationUUID: "uuid-x",
			plusAddressing:   true,
			want:             "replies+conv-uuid-x@example.com",
		},
		{
			name:             "plus-addressing on + inbox reply_to with display name: strips name",
			inboxReplyTo:     "example Support <support@example.com>",
			fromEmail:        "support@hedwig.example.com",
			conversationUUID: "id-1",
			plusAddressing:   true,
			want:             "support+conv-id-1@example.com",
		},
		{
			name:             "plus-addressing on + reply_to with surrounding whitespace: trimmed by parser",
			inboxReplyTo:     "  support@example.com  ",
			fromEmail:        "support@hedwig.example.com",
			conversationUUID: "uuid-w",
			plusAddressing:   true,
			want:             "support+conv-uuid-w@example.com",
		},
		{
			name:             "plus-addressing on + reply_to equals From: plus-addressed on that address",
			inboxReplyTo:     "support@hedwig.example.com",
			fromEmail:        "support@hedwig.example.com",
			conversationUUID: "uuid-same",
			plusAddressing:   true,
			want:             "support+conv-uuid-same@hedwig.example.com",
		},
		{
			name:             "plus-addressing on + reply_to already contains +: concatenates (no normalization)",
			inboxReplyTo:     "support+help@example.com",
			fromEmail:        "support@hedwig.example.com",
			conversationUUID: "uuid-p",
			plusAddressing:   true,
			want:             "support+help+conv-uuid-p@example.com",
		},
		{
			name:             "plus-addressing on + invalid inbox reply_to: silently falls back to From",
			inboxReplyTo:     "not-an-email",
			fromEmail:        "support@hedwig.example.com",
			conversationUUID: "id-2",
			plusAddressing:   true,
			want:             "support+conv-id-2@hedwig.example.com",
		},
		{
			name:             "plus-addressing on + reply_to missing domain: falls back to From",
			inboxReplyTo:     "support@",
			fromEmail:        "support@hedwig.example.com",
			conversationUUID: "id-3",
			plusAddressing:   true,
			want:             "support+conv-id-3@hedwig.example.com",
		},

		{
			name:             "plus-addressing on + no inbox reply_to: base taken from From",
			fromEmail:        "support@hedwig.example.com",
			conversationUUID: "uuid-2",
			plusAddressing:   true,
			want:             "support+conv-uuid-2@hedwig.example.com",
		},
		{
			name:             "plus-addressing on + subdomain-chained From",
			fromEmail:        "support@mail.hedwig.example.com",
			conversationUUID: "uuid-sub",
			plusAddressing:   true,
			want:             "support+conv-uuid-sub@mail.hedwig.example.com",
		},
		{
			name:             "plus-addressing on + UUID with typical hyphenated format",
			fromEmail:        "support@example.com",
			conversationUUID: "550e8400-e29b-41d4-a716-446655440000",
			plusAddressing:   true,
			want:             "support+conv-550e8400-e29b-41d4-a716-446655440000@example.com",
		},

		{
			name:           "plus-addressing on + no conversation UUID + inbox reply_to: literal reply_to",
			inboxReplyTo:   "support@example.com",
			fromEmail:      "support@hedwig.example.com",
			plusAddressing: true,
			want:           "support@example.com",
		},
		{
			name:           "plus-addressing on + no conversation UUID + no inbox reply_to: empty",
			fromEmail:      "support@hedwig.example.com",
			plusAddressing: true,
			want:           "",
		},
		{
			name:           "plus-addressing on + no conversation UUID + invalid inbox reply_to: falls back to literal From",
			inboxReplyTo:   "invalid",
			fromEmail:      "support@hedwig.example.com",
			plusAddressing: true,
			want:           "support@hedwig.example.com",
		},

		{
			name:         "plus-addressing off + inbox reply_to: literal reply_to",
			inboxReplyTo: "support@example.com",
			fromEmail:    "support@hedwig.example.com",
			want:         "support@example.com",
		},
		{
			name:         "plus-addressing off + inbox reply_to with display name: stripped",
			inboxReplyTo: "example Support <support@example.com>",
			fromEmail:    "support@hedwig.example.com",
			want:         "support@example.com",
		},
		{
			name:             "plus-addressing off + inbox reply_to + conversation UUID: still literal (no plus-addressing)",
			inboxReplyTo:     "support@example.com",
			fromEmail:        "support@hedwig.example.com",
			conversationUUID: "abc-123",
			want:             "support@example.com",
		},
		{
			name:      "plus-addressing off + no inbox reply_to: empty (customer replies to From)",
			fromEmail: "support@hedwig.example.com",
			want:      "",
		},
		{
			name:             "plus-addressing off + no inbox reply_to + conversation UUID: still empty",
			fromEmail:        "support@hedwig.example.com",
			conversationUUID: "abc-123",
			want:             "",
		},
		{
			name: "everything empty: empty result",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveReplyTo(tt.perMessage, tt.inboxReplyTo, tt.fromEmail, tt.conversationUUID, tt.plusAddressing)
			if got != tt.want {
				t.Errorf("resolveReplyTo() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBuildPlusAddress(t *testing.T) {
	tests := []struct {
		name             string
		email            string
		conversationUUID string
		want             string
	}{
		{
			name:             "standard address",
			email:            "support@example.com",
			conversationUUID: "abc-123",
			want:             "support+conv-abc-123@example.com",
		},
		{
			name:             "subdomain preserved",
			email:            "support@hedwig.example.com",
			conversationUUID: "id-1",
			want:             "support+conv-id-1@hedwig.example.com",
		},
		{
			name:             "no @ in input: returned unchanged",
			email:            "not-an-email",
			conversationUUID: "abc",
			want:             "not-an-email",
		},
		{
			name:             "empty email: returned unchanged",
			email:            "",
			conversationUUID: "abc",
			want:             "",
		},
		{
			name:             "empty UUID still produces a valid-shaped address",
			email:            "support@example.com",
			conversationUUID: "",
			want:             "support+conv-@example.com",
		},
		{
			name:             "multiple @ in input: splits only on first",
			email:            "weird@part@example.com",
			conversationUUID: "abc",
			want:             "weird+conv-abc@part@example.com",
		},
		{
			name:             "uuid containing plus: passed through verbatim",
			email:            "support@example.com",
			conversationUUID: "uuid+tag",
			want:             "support+conv-uuid+tag@example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildPlusAddress(tt.email, tt.conversationUUID)
			if got != tt.want {
				t.Errorf("buildPlusAddress() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestResolveReplyTo_PlusAddressIsRoundTrippable(t *testing.T) {
	const uuid = "550e8400-e29b-41d4-a716-446655440000"

	cases := []struct {
		name         string
		inboxReplyTo string
		fromEmail    string
	}{
		{"reply_to set", "support@example.com", "support@hedwig.example.com"},
		{"reply_to with display name", "example Support <support@example.com>", "support@hedwig.example.com"},
		{"no reply_to", "", "support@hedwig.example.com"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := resolveReplyTo("", tc.inboxReplyTo, tc.fromEmail, uuid, true)
			if !strings.Contains(got, "+conv-") {
				t.Errorf("expected plus-addressed reply-to, got %q", got)
			}
			if extracted := stringutil.ExtractConvUUID(got); extracted != uuid {
				t.Errorf("UUID round-trip failed: got %q from %q, want %q", extracted, got, uuid)
			}
		})
	}
}
