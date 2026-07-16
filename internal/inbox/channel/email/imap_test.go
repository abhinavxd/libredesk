package email

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/abhinavxd/libredesk/internal/attachment"
	"github.com/emersion/go-message/mail"
	"github.com/jhillyerd/enmime"
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

// TestCollectAttachments_OutlookInlineImages verifies that inline images embedded
// with a Content-ID but no Content-Disposition header (as Outlook does) are collected
// as inline attachments so their cid: references resolve in the message body. enmime
// routes such parts to envelope.OtherParts, which must not be dropped.
//
// The fixture testdata/outlook-inline-images.eml is a synthetic message that
// reproduces the exact MIME structure Outlook produces.
func TestCollectAttachments_OutlookInlineImages(t *testing.T) {
	const wantCID = "report-chart@example.com"

	f, err := os.Open(filepath.Join("testdata", "outlook-inline-images.eml"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	envelope, err := enmime.ReadEnvelope(f)
	if err != nil {
		t.Fatal(err)
	}

	// Sanity: enmime puts the image in OtherParts, not Inlines/Attachments,
	// because it carries no Content-Disposition header.
	if len(envelope.Inlines) != 0 || len(envelope.Attachments) != 0 {
		t.Fatalf("expected image to be unclassified by enmime, got inlines=%d attachments=%d",
			len(envelope.Inlines), len(envelope.Attachments))
	}
	if len(envelope.OtherParts) != 1 {
		t.Fatalf("expected 1 other part, got %d", len(envelope.OtherParts))
	}

	// The HTML body references the image via cid:.
	if !strings.Contains(envelope.HTML, "cid:"+wantCID) {
		t.Fatalf("fixture HTML does not reference cid:%s", wantCID)
	}

	got := collectAttachments(envelope)

	// The cid referenced in the HTML must resolve to a collected, non-empty inline attachment.
	var att *attachment.Attachment
	for i := range got {
		if got[i].ContentID == wantCID {
			att = &got[i]
			break
		}
	}
	if att == nil {
		t.Fatalf("cid:%s referenced in HTML but not collected as an attachment (got %d attachments)", wantCID, len(got))
	}
	if att.Disposition != attachment.DispositionInline {
		t.Errorf("expected inline disposition, got %q", att.Disposition)
	}
	if att.ContentType != "image/png" {
		t.Errorf("expected content type image/png, got %q", att.ContentType)
	}
	if att.Size == 0 || len(att.Content) == 0 {
		t.Errorf("expected non-empty attachment content, got size=%d len=%d", att.Size, len(att.Content))
	}
}

// TestCollectAttachments_Dispositions verifies disposition handling across the three
// enmime buckets: real attachments, inline parts, and inline-less other parts.
func TestCollectAttachments_Dispositions(t *testing.T) {
	env := &enmime.Envelope{
		Attachments: []*enmime.Part{
			{FileName: "doc.pdf", ContentType: "application/pdf", ContentID: "", Content: []byte("x")},
		},
		Inlines: []*enmime.Part{
			{FileName: "logo.png", ContentType: "image/png", ContentID: "logo@id", Content: []byte("x")},
		},
		OtherParts: []*enmime.Part{
			{FileName: "inline.png", ContentType: "image/png", ContentID: "inline@id", Content: []byte("x")},
			{FileName: "noid.png", ContentType: "image/png", ContentID: "", Content: []byte("x")},
		},
	}

	got := collectAttachments(env)
	if len(got) != 4 {
		t.Fatalf("expected 4 attachments, got %d", len(got))
	}

	byName := map[string]attachment.Attachment{}
	for _, a := range got {
		byName[a.Name] = a
	}

	cases := map[string]string{
		"doc.pdf":    attachment.DispositionAttachment, // real attachment
		"logo.png":   attachment.DispositionInline,     // inline w/ cid
		"inline.png": attachment.DispositionInline,     // other part w/ cid -> inline
		"noid.png":   attachment.DispositionAttachment, // other part w/o cid -> attachment
	}
	for name, want := range cases {
		if got := byName[name].Disposition; got != want {
			t.Errorf("%s: expected disposition %q, got %q", name, want, got)
		}
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
