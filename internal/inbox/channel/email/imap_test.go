package email

import (
	"strings"
	"testing"

	"github.com/emersion/go-message/mail"
	"github.com/jhillyerd/enmime"
	"github.com/zerodha/logf"
)

func TestEmail_extractUUIDFromReplyAddress(t *testing.T) {
	e := &Email{}

	testCases := []struct {
		name     string
		address  string
		expected string
	}{
		{
			name:     "Valid reply address with UUID",
			address:  "support+550e8400-e29b-41d4-a716-446655440000@example.com",
			expected: "550e8400-e29b-41d4-a716-446655440000",
		},
		{
			name:     "Reply address with angle brackets",
			address:  "<support+123e4567-e89b-42d3-a456-426614174000@example.com>",
			expected: "123e4567-e89b-42d3-a456-426614174000",
		},
		{
			name:     "No plus sign in address",
			address:  "support@example.com",
			expected: "",
		},
		{
			name:     "Plus sign but no UUID",
			address:  "support+test@example.com",
			expected: "",
		},
		{
			name:     "Invalid UUID format",
			address:  "support+550e8400-e29b-41d4-a716-44665544000X@example.com",
			expected: "550e8400-e29b-41d4-a716-44665544000X", // extractUUIDFromReplyAddress uses simple format check
		},
		{
			name:     "Empty address",
			address:  "",
			expected: "",
		},
		{
			name:     "UUID too short",
			address:  "support+550e8400-e29b-41d4-a716-4466554400@example.com",
			expected: "",
		},
		{
			name:     "UUID too long",
			address:  "support+550e8400-e29b-41d4-a716-4466554400000@example.com",
			expected: "",
		},
		{
			name:     "Multiple plus signs",
			address:  "support+test+550e8400-e29b-41d4-a716-446655440000@example.com",
			expected: "", // "test+550e8400-e29b-41d4-a716-446655440000" is not 36 chars, so validation fails
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := e.extractUUIDFromReplyAddress(tc.address)
			if result != tc.expected {
				t.Errorf("extractUUIDFromReplyAddress(%q) = %q; expected %q", tc.address, result, tc.expected)
			}
		})
	}
}

func TestEmail_extractConversationUUID(t *testing.T) {
	logger := logf.New(logf.Opts{Level: logf.DebugLevel})
	e := &Email{
		from: "support@example.com",
		lo:   &logger,
	}

	testCases := []struct {
		name     string
		envelope *mockEnvelope
		expected string
	}{
		{
			name: "UUID found in Reply-To address",
			envelope: &mockEnvelope{
				headers: map[string]string{
					"To": "support+550e8400-e29b-41d4-a716-446655440000@example.com",
				},
			},
			expected: "550e8400-e29b-41d4-a716-446655440000",
		},
		{
			name: "UUID found in In-Reply-To header",
			envelope: &mockEnvelope{
				headers: map[string]string{
					"To":          "support@example.com",
					"In-Reply-To": "<123e4567-e89b-42d3-a456-426614174000.1735555200000000000@example.com>",
				},
			},
			expected: "123e4567-e89b-42d3-a456-426614174000",
		},
		{
			name: "UUID found in References header",
			envelope: &mockEnvelope{
				headers: map[string]string{
					"To":          "support@example.com",
					"In-Reply-To": "",
					"References":  "<123e4567-e89b-42d3-a456-426614174000.1735555200000000000@example.com> <another-id@domain.com>",
				},
			},
			expected: "123e4567-e89b-42d3-a456-426614174000",
		},
		{
			name: "No UUID found anywhere",
			envelope: &mockEnvelope{
				headers: map[string]string{
					"To":          "support@example.com",
					"In-Reply-To": "<no-uuid-here@example.com>",
					"References":  "<also-no-uuid@example.com>",
				},
			},
			expected: "",
		},
		{
			name: "Multiple UUIDs, returns first from Reply-To",
			envelope: &mockEnvelope{
				headers: map[string]string{
					"To":          "support+550e8400-e29b-41d4-a716-446655440000@example.com",
					"In-Reply-To": "<123e4567-e89b-42d3-a456-426614174000.1735555200000000000@example.com>",
				},
			},
			expected: "550e8400-e29b-41d4-a716-446655440000", // Reply-To takes precedence
		},
		{
			name: "Empty headers",
			envelope: &mockEnvelope{
				headers: map[string]string{},
			},
			expected: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a real enmime.Envelope with our mock data
			envelope := createMockEnvelope(tc.envelope.headers)
			result := e.extractConversationUUID(envelope)
			if result != tc.expected {
				t.Errorf("extractConversationUUID() = %q; expected %q", result, tc.expected)
			}
		})
	}
}

// TestGoIMAPMessageIDParsing shows how go-imap fails to parse malformed Message-IDs
// and demonstrates the fallback solution.
// go-imap uses mail.Header.MessageID() which strictly follows RFC 5322 and returns
// empty strings for Message-IDs with multiple @ symbols.
//
// This caused emails to be dropped since we require Message-IDs for deduplication.
// References:
// - https://community.mailcow.email/d/701-multiple-at-in-message-id/5
// - https://github.com/emersion/go-message/issues/154#issuecomment-1425634946
func TestGoIMAPMessageIDParsing(t *testing.T) {
	testCases := []struct {
		input            string
		expectedIMAP     string
		expectedFallback string
		name             string
	}{
		{"<normal@example.com>", "normal@example.com", "normal@example.com", "normal message ID"},
		{"<malformed@@example.com>", "", "malformed@@example.com", "double @ - IMAP fails, fallback works"},
		{"<001c01d710db$a8137a50$f83a6ef0$@jones.smith@example.com>", "", "001c01d710db$a8137a50$f83a6ef0$@jones.smith@example.com", "mailcow-style - IMAP fails, fallback works"},
		{"<test@@@domain.com>", "", "test@@@domain.com", "triple @ - IMAP fails, fallback works"},
		{"  <abc123@example.com>  ", "abc123@example.com", "abc123@example.com", "with whitespace - both handle correctly"},
		{"abc123@example.com", "", "abc123@example.com", "no angle brackets - IMAP fails, fallback works"},
		{"", "", "", "empty input"},
		{"<>", "", "", "empty brackets"},
		{"<CAFnQjQFhY8z@mail.example.com@gateway.company.com>", "", "CAFnQjQFhY8z@mail.example.com@gateway.company.com", "gateway-style - IMAP fails, fallback works"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test go-imap parsing behavior
			var h mail.Header
			h.Set("Message-Id", tc.input)
			imapResult, _ := h.MessageID()

			if imapResult != tc.expectedIMAP {
				t.Errorf("IMAP parsing of %q: expected %q, got %q", tc.input, tc.expectedIMAP, imapResult)
			}

			// Test fallback solution
			if tc.input != "" {
				rawEmail := "From: test@example.com\nMessage-ID: " + tc.input + "\n\nBody"
				envelope, err := enmime.ReadEnvelope(strings.NewReader(rawEmail))
				if err != nil {
					t.Fatal(err)
				}

				fallbackResult := extractMessageIDFromHeaders(envelope)
				if fallbackResult != tc.expectedFallback {
					t.Errorf("Fallback extraction of %q: expected %q, got %q", tc.input, tc.expectedFallback, fallbackResult)
				}

				// Critical check: ensure fallback works when IMAP fails
				if imapResult == "" && tc.expectedFallback != "" && fallbackResult == "" {
					t.Errorf("CRITICAL: Both IMAP and fallback failed for %q - would drop email!", tc.input)
				}
			}
		})
	}
}

// TestEdgeCasesMessageID tests additional edge cases for Message-ID extraction.
func TestEdgeCasesMessageID(t *testing.T) {
	tests := []struct {
		name     string
		email    string
		expected string
	}{
		{
			name: "no Message-ID header",
			email: `From: test@example.com
To: inbox@test.com
Subject: Test

Body`,
			expected: "",
		},
		{
			name: "malformed header syntax",
			email: `From: test@example.com
Message-ID: malformed-no-brackets@@domain.com
To: inbox@test.com

Body`,
			expected: "malformed-no-brackets@@domain.com",
		},
		{
			name: "multiple Message-ID headers (first wins)",
			email: `From: test@example.com
Message-ID: <first@example.com>
Message-ID: <second@@example.com>
To: inbox@test.com

Body`,
			expected: "first@example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			envelope, err := enmime.ReadEnvelope(strings.NewReader(tt.email))
			if err != nil {
				t.Fatal(err)
			}

			result := extractMessageIDFromHeaders(envelope)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// mockEnvelope stores test header data
type mockEnvelope struct {
	headers map[string]string
}

// createMockEnvelope creates a minimal enmime.Envelope for testing
func createMockEnvelope(headers map[string]string) *enmime.Envelope {
	// Create a minimal email content with the required headers
	var emailContent strings.Builder
	for key, value := range headers {
		emailContent.WriteString(key + ": " + value + "\r\n")
	}
	emailContent.WriteString("\r\n") // Empty line to separate headers from body
	emailContent.WriteString("Test body content")

	envelope, _ := enmime.ReadEnvelope(strings.NewReader(emailContent.String()))
	return envelope
}
