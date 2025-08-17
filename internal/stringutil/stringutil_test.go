package stringutil

import (
	"testing"
	"time"
)

func TestReverseSlice(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "empty slice",
			input:    []string{},
			expected: []string{},
		},
		{
			name:     "single element",
			input:    []string{"a"},
			expected: []string{"a"},
		},
		{
			name:     "multiple elements",
			input:    []string{"a", "b", "c"},
			expected: []string{"c", "b", "a"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := make([]string, len(tt.input))
			copy(input, tt.input)
			ReverseSlice(input)
			if len(input) != len(tt.expected) {
				t.Errorf("got len %d, want %d", len(input), len(tt.expected))
			}
			for i := range input {
				if input[i] != tt.expected[i] {
					t.Errorf("at index %d got %s, want %s", i, input[i], tt.expected[i])
				}
			}
		})
	}
}

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

func TestGenerateSlug(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple title",
			input:    "Hello World",
			expected: "hello-world",
		},
		{
			name:     "title with special characters",
			input:    "Hello, World! How are you?",
			expected: "hello-world-how-are-you",
		},
		{
			name:     "title with numbers",
			input:    "Article 123: How to Code",
			expected: "article-123-how-to-code",
		},
		{
			name:     "title with underscores",
			input:    "test_article_name",
			expected: "test_article_name",
		},
		{
			name:     "title with multiple spaces",
			input:    "Hello     World",
			expected: "hello-world",
		},
		{
			name:     "title with leading/trailing spaces",
			input:    "  Hello World  ",
			expected: "hello-world",
		},
		{
			name:     "title with multiple hyphens",
			input:    "Hello---World",
			expected: "hello-world",
		},
		{
			name:     "unicode characters",
			input:    "Hello World",
			expected: "hello-world",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateSlug(tt.input, false)
			if result != tt.expected {
				t.Errorf("GenerateSlug(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
