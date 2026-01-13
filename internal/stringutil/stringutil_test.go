package stringutil

import (
	"testing"
	"time"
)

func TestRemoveItemByValue(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		remove   string
		expected []string
	}{
		{
			name:     "empty slice",
			input:    []string{},
			remove:   "a",
			expected: []string{},
		},
		{
			name:     "no matches",
			input:    []string{"b", "c"},
			remove:   "a",
			expected: []string{"b", "c"},
		},
		{
			name:     "single match",
			input:    []string{"a", "b", "c"},
			remove:   "b",
			expected: []string{"a", "c"},
		},
		{
			name:     "multiple matches",
			input:    []string{"a", "b", "a", "c", "a"},
			remove:   "a",
			expected: []string{"b", "c"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RemoveItemByValue(tt.input, tt.remove)
			if len(result) != len(tt.expected) {
				t.Errorf("got len %d, want %d", len(result), len(tt.expected))
			}
			for i := range result {
				if result[i] != tt.expected[i] {
					t.Errorf("at index %d got %s, want %s", i, result[i], tt.expected[i])
				}
			}
		})
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name           string
		duration       time.Duration
		includeSeconds bool
		expected       string
	}{
		{
			name:           "zero duration with seconds",
			duration:       0,
			includeSeconds: true,
			expected:       "0 minutes",
		},
		{
			name:           "hours only",
			duration:       2 * time.Hour,
			includeSeconds: false,
			expected:       "2 hours 0 minutes",
		},
		{
			name:           "hours and minutes",
			duration:       2*time.Hour + 30*time.Minute,
			includeSeconds: false,
			expected:       "2 hours 30 minutes",
		},
		{
			name:           "full duration with seconds",
			duration:       2*time.Hour + 30*time.Minute + 15*time.Second,
			includeSeconds: true,
			expected:       "2 hours 30 minutes 15 seconds",
		},
		{
			name:           "full duration without seconds",
			duration:       2*time.Hour + 30*time.Minute + 15*time.Second,
			includeSeconds: false,
			expected:       "2 hours 30 minutes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatDuration(tt.duration, tt.includeSeconds)
			if result != tt.expected {
				t.Errorf("got %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestStripConvUUID(t *testing.T) {
	tests := []struct {
		name     string
		email    string
		expected string
	}{
		{
			name:     "no plus addressing",
			email:    "support@domain.com",
			expected: "support@domain.com",
		},
		{
			name:     "with valid UUID v4",
			email:    "support+conv-13216cf7-6626-4b0d-a938-46ce65a20701@domain.com",
			expected: "support@domain.com",
		},
		{
			name:     "short non-UUID preserved (user email)",
			email:    "support+conv-21321@domain.com",
			expected: "support+conv-21321@domain.com",
		},
		{
			name:     "non-conv plus addressing unchanged",
			email:    "support+other@domain.com",
			expected: "support+other@domain.com",
		},
		{
			name:     "empty string",
			email:    "",
			expected: "",
		},
		{
			name:     "uppercase UUID v4",
			email:    "support+conv-13216CF7-6626-4B0D-A938-46CE65A20701@domain.com",
			expected: "support@domain.com",
		},
		{
			name:     "invalid UUID format preserved",
			email:    "support+conv-abc123-def456@domain.com",
			expected: "support+conv-abc123-def456@domain.com",
		},
		{
			name:     "missing 4 in UUID preserved",
			email:    "support+conv-13216cf7-6626-ab0d-a938-46ce65a20701@domain.com",
			expected: "support+conv-13216cf7-6626-ab0d-a938-46ce65a20701@domain.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := StripConvUUID(tt.email)
			if result != tt.expected {
				t.Errorf("StripConvUUID(%q) = %q, want %q", tt.email, result, tt.expected)
			}
		})
	}
}

func TestExtractConvUUID(t *testing.T) {
	tests := []struct {
		name     string
		email    string
		expected string
	}{
		{
			name:     "valid UUID v4",
			email:    "support+conv-13216cf7-6626-4b0d-a938-46ce65a20701@domain.com",
			expected: "13216cf7-6626-4b0d-a938-46ce65a20701",
		},
		{
			name:     "uppercase UUID v4",
			email:    "support+conv-13216CF7-6626-4B0D-A938-46CE65A20701@domain.com",
			expected: "13216CF7-6626-4B0D-A938-46CE65A20701",
		},
		{
			name:     "no plus addressing",
			email:    "support@domain.com",
			expected: "",
		},
		{
			name:     "non-conv plus addressing",
			email:    "support+other@domain.com",
			expected: "",
		},
		{
			name:     "short non-UUID (user email)",
			email:    "support+conv-21321@domain.com",
			expected: "",
		},
		{
			name:     "invalid UUID format",
			email:    "support+conv-abc123-def456@domain.com",
			expected: "",
		},
		{
			name:     "missing 4 in UUID (invalid v4)",
			email:    "support+conv-13216cf7-6626-ab0d-a938-46ce65a20701@domain.com",
			expected: "",
		},
		{
			name:     "empty string",
			email:    "",
			expected: "",
		},
		{
			name:     "missing @ symbol",
			email:    "support+conv-13216cf7-6626-4b0d-a938-46ce65a20701",
			expected: "",
		},
		{
			name:     "UUID with extra chars",
			email:    "support+conv-13216cf7-6626-4b0d-a938-46ce65a20701-extra@domain.com",
			expected: "",
		},
		{
			name:     "valid UUID different local part",
			email:    "inbox+conv-a1b2c3d4-e5f6-4a7b-8c9d-0e1f2a3b4c5d@example.org",
			expected: "a1b2c3d4-e5f6-4a7b-8c9d-0e1f2a3b4c5d",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractConvUUID(tt.email)
			if result != tt.expected {
				t.Errorf("ExtractConvUUID(%q) = %q, want %q", tt.email, result, tt.expected)
			}
		})
	}
}

func TestDedupAndExcludePlusVariants(t *testing.T) {
	tests := []struct {
		name      string
		list      []string
		baseEmail string
		expected  []string
	}{
		{
			name:      "removes exact match",
			list:      []string{"other@domain.com", "support@domain.com"},
			baseEmail: "support@domain.com",
			expected:  []string{"other@domain.com"},
		},
		{
			name:      "removes valid UUID v4 plus-addressed variant",
			list:      []string{"other@domain.com", "support+conv-13216cf7-6626-4b0d-a938-46ce65a20701@domain.com"},
			baseEmail: "support@domain.com",
			expected:  []string{"other@domain.com"},
		},
		{
			name:      "removes both exact and UUID v4 plus variant",
			list:      []string{"support@domain.com", "other@domain.com", "support+conv-a1b2c3d4-e5f6-4a7b-8c9d-0e1f2a3b4c5d@domain.com"},
			baseEmail: "support@domain.com",
			expected:  []string{"other@domain.com"},
		},
		{
			name:      "keeps non-conv plus addresses",
			list:      []string{"support+other@domain.com", "other@domain.com"},
			baseEmail: "support@domain.com",
			expected:  []string{"support+other@domain.com", "other@domain.com"},
		},
		{
			name:      "keeps non-UUID conv addresses (user email)",
			list:      []string{"support+conv-21321@domain.com", "other@domain.com"},
			baseEmail: "support@domain.com",
			expected:  []string{"support+conv-21321@domain.com", "other@domain.com"},
		},
		{
			name:      "deduplicates",
			list:      []string{"other@domain.com", "other@domain.com", "another@domain.com"},
			baseEmail: "support@domain.com",
			expected:  []string{"other@domain.com", "another@domain.com"},
		},
		{
			name:      "removes empty strings",
			list:      []string{"", "other@domain.com", ""},
			baseEmail: "support@domain.com",
			expected:  []string{"other@domain.com"},
		},
		{
			name:      "case insensitive base email match",
			list:      []string{"SUPPORT@domain.com", "other@domain.com"},
			baseEmail: "support@domain.com",
			expected:  []string{"other@domain.com"},
		},
		{
			name:      "empty list",
			list:      []string{},
			baseEmail: "support@domain.com",
			expected:  []string{},
		},
		{
			name:      "empty inboxEmail preserves all non-empty emails",
			list:      []string{"user@example.com", "other@domain.com"},
			baseEmail: "",
			expected:  []string{"user@example.com", "other@domain.com"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DedupAndExcludePlusVariants(tt.list, tt.baseEmail)
			if len(result) != len(tt.expected) {
				t.Errorf("got len %d, want %d", len(result), len(tt.expected))
				return
			}
			for i := range result {
				if result[i] != tt.expected[i] {
					t.Errorf("at index %d got %s, want %s", i, result[i], tt.expected[i])
				}
			}
		})
	}
}

func TestExtractReferenceNumber(t *testing.T) {
	tests := []struct {
		name     string
		subject  string
		expected string
	}{
		{
			name:     "simple reference number",
			subject:  "Test - #392",
			expected: "392",
		},
		{
			name:     "with RE prefix",
			subject:  "RE: Test - #392",
			expected: "392",
		},
		{
			name:     "multiple hashes picks last",
			subject:  "Order #123 - #392",
			expected: "392",
		},
		{
			name:     "no reference number",
			subject:  "Just a regular subject",
			expected: "",
		},
		{
			name:     "hash without number",
			subject:  "Test #abc",
			expected: "",
		},
		{
			name:     "empty string",
			subject:  "",
			expected: "",
		},
		{
			name:     "number without hash",
			subject:  "Test 392",
			expected: "",
		},
		{
			name:     "multiple RE prefixes",
			subject:  "RE: RE: Test - #100",
			expected: "100",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractReferenceNumber(tt.subject)
			if result != tt.expected {
				t.Errorf("ExtractReferenceNumber(%q) = %q, want %q", tt.subject, result, tt.expected)
			}
		})
	}
}
