package email

import (
	"strings"
	"testing"

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
					"To":         "support@example.com",
					"In-Reply-To": "",
					"References": "<123e4567-e89b-42d3-a456-426614174000.1735555200000000000@example.com> <another-id@domain.com>",
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

// BenchmarkExtractConversationUUID benchmarks the conversation UUID extraction
func BenchmarkExtractConversationUUID(b *testing.B) {
	logger := logf.New(logf.Opts{Level: logf.DebugLevel})
	e := &Email{
		from: "support@example.com",
		lo:   &logger,
	}

	headers := map[string]string{
		"To":          "support+550e8400-e29b-41d4-a716-446655440000@example.com",
		"In-Reply-To": "<123e4567-e89b-42d3-a456-426614174000.1735555200000000000@example.com>",
		"References":  "<abc12345-e89b-42d3-a456-426614174000.1735555200000000000@example.com>",
	}
	envelope := createMockEnvelope(headers)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		e.extractConversationUUID(envelope)
	}
}