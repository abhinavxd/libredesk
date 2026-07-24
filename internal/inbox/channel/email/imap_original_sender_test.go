package email

import (
	"strings"
	"testing"

	"github.com/jhillyerd/enmime"
)

func TestExtractSenderFromHeader(t *testing.T) {
	const header = "X-Original-Sender"

	testCases := []struct {
		name         string
		rawEmail     string
		headerName   string
		expectedAddr string
		expectedName string
	}{
		{
			name:         "address with display name",
			rawEmail:     "From: group@lists.example.com\r\nX-Original-Sender: Jane Doe <Jane.Doe@Example.com>\r\nSubject: hi\r\n\r\nbody",
			headerName:   header,
			expectedAddr: "jane.doe@example.com",
			expectedName: "Jane Doe",
		},
		{
			name:         "bare address is lowercased",
			rawEmail:     "From: group@lists.example.com\r\nX-Original-Sender: USER@Example.com\r\nSubject: hi\r\n\r\nbody",
			headerName:   header,
			expectedAddr: "user@example.com",
			expectedName: "",
		},
		{
			name:         "header absent falls back to empty",
			rawEmail:     "From: group@lists.example.com\r\nSubject: hi\r\n\r\nbody",
			headerName:   header,
			expectedAddr: "",
			expectedName: "",
		},
		{
			name:         "unparseable header falls back to empty",
			rawEmail:     "From: group@lists.example.com\r\nX-Original-Sender: not-an-email\r\nSubject: hi\r\n\r\nbody",
			headerName:   header,
			expectedAddr: "",
			expectedName: "",
		},
		{
			name:         "empty header name is a no-op",
			rawEmail:     "From: group@lists.example.com\r\nX-Original-Sender: user@example.com\r\nSubject: hi\r\n\r\nbody",
			headerName:   "",
			expectedAddr: "",
			expectedName: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			env, err := enmime.ReadEnvelope(strings.NewReader(tc.rawEmail))
			if err != nil {
				t.Fatalf("reading envelope: %v", err)
			}
			addr, name := extractSenderFromHeader(env, tc.headerName)
			if addr != tc.expectedAddr {
				t.Errorf("addr = %q, want %q", addr, tc.expectedAddr)
			}
			if name != tc.expectedName {
				t.Errorf("name = %q, want %q", name, tc.expectedName)
			}
		})
	}
}

func TestIsValidHeaderName(t *testing.T) {
	testCases := []struct {
		name  string
		input string
		valid bool
	}{
		{"typical header", "X-Original-Sender", true},
		{"simple token", "From", true},
		{"empty", "", false},
		{"contains space", "X Original Sender", false},
		{"contains colon", "X-Original-Sender:", false},
		{"contains control char", "X-Original\tSender", false},
		{"non-ascii", "X-Originál-Sender", false},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isValidHeaderName(tc.input); got != tc.valid {
				t.Errorf("isValidHeaderName(%q) = %v, want %v", tc.input, got, tc.valid)
			}
		})
	}
}
