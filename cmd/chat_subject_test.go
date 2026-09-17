package main

import (
	"strings"
	"testing"
)

func TestNormalizeChatSubject(t *testing.T) {
	ascii := strings.Repeat("a", maxChatSubjectLength)
	cyrillic := strings.Repeat("ж", maxChatSubjectLength) // 2 bytes per character: byte-counting would reject it

	tests := []struct {
		name string
		in   string
		want string
		ok   bool
	}{
		{"empty stays empty", "", "", true},
		{"whitespace collapsed and trimmed", "  Login \t fails\n after  update  ", "Login fails after update", true},
		{"ascii at the limit", ascii, ascii, true},
		{"ascii one over the limit", ascii + "a", "", false},
		{"non-ascii at the limit counts characters, not bytes", cyrillic, cyrillic, true},
		{"non-ascii one over the limit", cyrillic + "ж", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := normalizeChatSubject(tt.in)
			if ok != tt.ok || got != tt.want {
				t.Fatalf("normalizeChatSubject(%q) = (%q, %v), want (%q, %v)", tt.in, got, ok, tt.want, tt.ok)
			}
		})
	}
}
