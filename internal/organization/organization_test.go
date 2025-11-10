package organization

import (
	"testing"
)

func TestDomainValidation(t *testing.T) {
	tests := []struct {
		name     string
		domain   string
		expected bool
	}{
		{
			name:     "valid domain",
			domain:   "example.com",
			expected: true,
		},
		{
			name:     "valid subdomain",
			domain:   "mail.example.com",
			expected: true,
		},
		{
			name:     "valid multi-level subdomain",
			domain:   "api.v2.example.com",
			expected: true,
		},
		{
			name:     "invalid - no TLD",
			domain:   "example",
			expected: false,
		},
		{
			name:     "invalid - starts with dot",
			domain:   ".example.com",
			expected: false,
		},
		{
			name:     "invalid - ends with dot",
			domain:   "example.com.",
			expected: false,
		},
		{
			name:     "invalid - contains space",
			domain:   "exam ple.com",
			expected: false,
		},
		{
			name:     "invalid - empty",
			domain:   "",
			expected: false,
		},
		{
			name:     "valid - with hyphen",
			domain:   "my-example.com",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := domainRegex.MatchString(tt.domain)
			if result != tt.expected {
				t.Errorf("domain %s: got %v, want %v", tt.domain, result, tt.expected)
			}
		})
	}
}
