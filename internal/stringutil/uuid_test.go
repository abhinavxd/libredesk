package stringutil

import "testing"

func TestExtractUUID(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Valid UUID v4 in Message-ID format",
			input:    "<550e8400-e29b-41d4-a716-446655440000.1735555200000000000@example.com>",
			expected: "550e8400-e29b-41d4-a716-446655440000",
		},
		{
			name:     "Valid UUID v4 in plain text",
			input:    "some text 123e4567-e89b-42d3-a456-426614174000 more text",
			expected: "123e4567-e89b-42d3-a456-426614174000",
		},
		{
			name:     "Valid UUID v4 with uppercase",
			input:    "550E8400-E29B-41D4-A716-446655440000",
			expected: "550E8400-E29B-41D4-A716-446655440000",
		},
		{
			name:     "Valid UUID v4 mixed case",
			input:    "550e8400-E29B-41d4-A716-446655440000",
			expected: "550e8400-E29B-41d4-A716-446655440000",
		},
		{
			name:     "Multiple UUIDs returns first valid one",
			input:    "first: 550e8400-e29b-41d4-a716-446655440000 second: 123e4567-e89b-42d3-a456-426614174000",
			expected: "550e8400-e29b-41d4-a716-446655440000",
		},
		{
			name:     "No UUID in text",
			input:    "no uuid here just random text",
			expected: "",
		},
		{
			name:     "Empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "Invalid UUID - wrong length",
			input:    "550e8400-e29b-41d4-a716-44665544000", // missing last char
			expected: "",
		},
		{
			name:     "Invalid UUID - wrong version (not v4)",
			input:    "550e8400-e29b-31d4-a716-446655440000", // version 3, not 4
			expected: "",
		},
		{
			name:     "Invalid UUID - wrong variant",
			input:    "550e8400-e29b-41d4-c716-446655440000", // variant C instead of A/B/8/9
			expected: "",
		},
		{
			name:     "Invalid UUID - non-hex characters",
			input:    "550e8400-e29b-41d4-a716-44665544000g", // 'g' is not hex
			expected: "",
		},
		{
			name:     "UUID-like but wrong format",
			input:    "550e8400_e29b_41d4_a716_446655440000", // underscores instead of hyphens
			expected: "",
		},
		{
			name:     "Valid UUID in email References header",
			input:    "<550e8400-e29b-41d4-a716-446655440000.12345@domain.com> <another-id@domain.com>",
			expected: "550e8400-e29b-41d4-a716-446655440000",
		},
		{
			name:     "Valid UUID with variant 8",
			input:    "550e8400-e29b-41d4-8716-446655440000", // variant 8
			expected: "550e8400-e29b-41d4-8716-446655440000",
		},
		{
			name:     "Valid UUID with variant 9",
			input:    "550e8400-e29b-41d4-9716-446655440000", // variant 9
			expected: "550e8400-e29b-41d4-9716-446655440000",
		},
		{
			name:     "Valid UUID with variant A",
			input:    "550e8400-e29b-41d4-A716-446655440000", // variant A
			expected: "550e8400-e29b-41d4-A716-446655440000",
		},
		{
			name:     "Valid UUID with variant B",
			input:    "550e8400-e29b-41d4-B716-446655440000", // variant B
			expected: "550e8400-e29b-41d4-B716-446655440000",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := ExtractUUID(tc.input)
			if result != tc.expected {
				t.Errorf("ExtractUUID(%q) = %q; expected %q", tc.input, result, tc.expected)
			}
		})
	}
}

// BenchmarkExtractUUID benchmarks the UUID extraction function
func BenchmarkExtractUUID(b *testing.B) {
	testString := "<550e8400-e29b-41d4-a716-446655440000.1735555200000000000@example.com>"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ExtractUUID(testString)
	}
}

// BenchmarkExtractUUIDNoMatch benchmarks when no UUID is found
func BenchmarkExtractUUIDNoMatch(b *testing.B) {
	testString := "no uuid in this string at all, just random text and numbers 12345"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ExtractUUID(testString)
	}
}