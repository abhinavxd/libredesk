package main

import (
	"strings"
	"testing"
)

func TestNormalizeConversationSubject(t *testing.T) {
	tests := []struct {
		name   string
		in     string
		want   string
		wantOK bool
	}{
		{name: "plain", in: "Refund request", want: "Refund request", wantOK: true},
		{name: "collapses whitespace", in: "  Refund\n\trequest   #42 ", want: "Refund request #42", wantOK: true},
		{name: "empty clears", in: "", want: "", wantOK: true},
		{name: "blank clears", in: " \n\t ", want: "", wantOK: true},
		{name: "ascii at limit", in: strings.Repeat("a", maxConversationSubjectLength), want: strings.Repeat("a", maxConversationSubjectLength), wantOK: true},
		{name: "ascii over limit", in: strings.Repeat("a", maxConversationSubjectLength+1)},
		{name: "non-ascii at limit counts characters", in: strings.Repeat("é", maxConversationSubjectLength), want: strings.Repeat("é", maxConversationSubjectLength), wantOK: true},
		{name: "non-ascii over limit", in: strings.Repeat("é", maxConversationSubjectLength+1)},
		{name: "limit applies after collapsing", in: strings.Repeat("a", maxConversationSubjectLength) + "   \n", want: strings.Repeat("a", maxConversationSubjectLength), wantOK: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := normalizeConversationSubject(tc.in)
			if ok != tc.wantOK || got != tc.want {
				t.Fatalf("normalizeConversationSubject(%q) = (%q, %v), want (%q, %v)", tc.in, got, ok, tc.want, tc.wantOK)
			}
		})
	}
}
