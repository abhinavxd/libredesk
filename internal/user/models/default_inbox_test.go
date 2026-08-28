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

func TestCanAccessDefaultInbox(t *testing.T) {
	restricted := []string{"conversations:read", "conversations:read_assigned"}
	full := []string{
		"conversations:read",
		"conversations:read_assigned",
		"conversations:read_unassigned",
		"conversations:read_all",
	}

	cases := []struct {
		inbox string
		perms []string
		want  bool
	}{
		{DefaultInboxAssigned, restricted, true},
		{DefaultInboxMentioned, restricted, true},
		{DefaultInboxUnassigned, restricted, false},
		{DefaultInboxAll, restricted, false},
		{DefaultInboxAssigned, full, true},
		{DefaultInboxMentioned, full, true},
		{DefaultInboxUnassigned, full, true},
		{DefaultInboxAll, full, true},
		{DefaultInboxAssigned, nil, false},
		{"nope", full, false},
	}
	for _, tt := range cases {
		if got := CanAccessDefaultInbox(tt.perms, tt.inbox); got != tt.want {
			t.Errorf("CanAccessDefaultInbox(%v, %q) = %v, want %v", tt.perms, tt.inbox, got, tt.want)
		}
	}
}
