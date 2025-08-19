package email

import (
	"strings"
	"testing"

	"github.com/emersion/go-message/mail"
	"github.com/jhillyerd/enmime"
)


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
