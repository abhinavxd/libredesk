package models

import "testing"

func TestNormalizeDefaultInbox(t *testing.T) {
	tests := []struct {
		in     string
		want   string
		wantOK bool
	}{
		{"", DefaultInboxAssigned, true},
		{"assigned", DefaultInboxAssigned, true},
		{"mentioned", DefaultInboxMentioned, true},
		{"unassigned", DefaultInboxUnassigned, true},
		{"all", DefaultInboxAll, true},
		{"All", "", false},
		{"mine", "", false},
	}
	for _, tt := range tests {
		got, ok := NormalizeDefaultInbox(tt.in)
		if ok != tt.wantOK || got != tt.want {
			t.Errorf("NormalizeDefaultInbox(%q) = (%q, %v), want (%q, %v)", tt.in, got, ok, tt.want, tt.wantOK)
		}
	}
}
