package models

import (
	"maps"
	"testing"
)

func TestResolveExternalSync(t *testing.T) {
	tests := []struct {
		name      string
		current   map[string]string
		synced    map[string]string
		incoming  map[string]string
		wantApply map[string]string
		wantNext  map[string]string
	}{
		{
			name:      "first sync fills the empty fields",
			current:   map[string]string{ExternalSyncFirstName: "Ada"},
			synced:    map[string]string{},
			incoming:  map[string]string{ExternalSyncFirstName: "Ada", ExternalSyncEmail: "ada@example.com", ExternalSyncPhoneNumber: "5550100"},
			wantApply: map[string]string{ExternalSyncEmail: "ada@example.com", ExternalSyncPhoneNumber: "5550100"},
			wantNext:  map[string]string{ExternalSyncFirstName: "Ada", ExternalSyncEmail: "ada@example.com", ExternalSyncPhoneNumber: "5550100"},
		},
		{
			name:      "an unchanged claim writes nothing",
			current:   map[string]string{ExternalSyncFirstName: "Ada", ExternalSyncEmail: "ada@example.com"},
			synced:    map[string]string{ExternalSyncFirstName: "Ada", ExternalSyncEmail: "ada@example.com"},
			incoming:  map[string]string{ExternalSyncFirstName: "Ada", ExternalSyncEmail: "ada@example.com"},
			wantApply: map[string]string{},
			wantNext:  map[string]string{ExternalSyncFirstName: "Ada", ExternalSyncEmail: "ada@example.com"},
		},
		{
			name:      "a manual edit survives the same claim coming round again",
			current:   map[string]string{ExternalSyncFirstName: "Ada Lovelace"},
			synced:    map[string]string{ExternalSyncFirstName: "ada"},
			incoming:  map[string]string{ExternalSyncFirstName: "ada"},
			wantApply: map[string]string{},
			wantNext:  map[string]string{ExternalSyncFirstName: "ada"},
		},
		{
			name:      "a manual edit survives a changed claim, and the record still follows it",
			current:   map[string]string{ExternalSyncFirstName: "Ada Lovelace"},
			synced:    map[string]string{ExternalSyncFirstName: "ada"},
			incoming:  map[string]string{ExternalSyncFirstName: "ADA"},
			wantApply: map[string]string{},
			wantNext:  map[string]string{ExternalSyncFirstName: "ADA"},
		},
		{
			name:      "a field set back to the supplied value follows the claim again",
			current:   map[string]string{ExternalSyncFirstName: "ADA"},
			synced:    map[string]string{ExternalSyncFirstName: "ADA"},
			incoming:  map[string]string{ExternalSyncFirstName: "Ada"},
			wantApply: map[string]string{ExternalSyncFirstName: "Ada"},
			wantNext:  map[string]string{ExternalSyncFirstName: "Ada"},
		},
		{
			name:      "an empty claim clears nothing and leaves the record standing",
			current:   map[string]string{ExternalSyncFirstName: "Ada", ExternalSyncPhoneNumber: "5550100"},
			synced:    map[string]string{ExternalSyncFirstName: "Ada", ExternalSyncPhoneNumber: "5550100"},
			incoming:  map[string]string{ExternalSyncFirstName: "Ada"},
			wantApply: map[string]string{},
			wantNext:  map[string]string{ExternalSyncFirstName: "Ada", ExternalSyncPhoneNumber: "5550100"},
		},
		{
			name:      "every field is decided on its own",
			current:   map[string]string{ExternalSyncFirstName: "Ada Lovelace", ExternalSyncEmail: "ada@example.com"},
			synced:    map[string]string{ExternalSyncFirstName: "Ada", ExternalSyncEmail: "ada@example.com"},
			incoming:  map[string]string{ExternalSyncFirstName: "Augusta", ExternalSyncEmail: "ada@example.org", ExternalSyncPhoneCountryCode: "GB"},
			wantApply: map[string]string{ExternalSyncEmail: "ada@example.org", ExternalSyncPhoneCountryCode: "GB"},
			wantNext:  map[string]string{ExternalSyncFirstName: "Augusta", ExternalSyncEmail: "ada@example.org", ExternalSyncPhoneCountryCode: "GB"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			apply, next := ResolveExternalSync(test.current, test.synced, test.incoming)
			if !maps.Equal(apply, test.wantApply) {
				t.Errorf("apply = %v, want %v", apply, test.wantApply)
			}
			if !maps.Equal(next, test.wantNext) {
				t.Errorf("next = %v, want %v", next, test.wantNext)
			}
		})
	}
}

func TestUnmarshalExternalSync(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want map[string]string
	}{
		{"no record yet", "", map[string]string{}},
		{"an empty record", "{}", map[string]string{}},
		{"a record", `{"first_name":"Ada","email":"ada@example.com"}`, map[string]string{ExternalSyncFirstName: "Ada", ExternalSyncEmail: "ada@example.com"}},
		{"an empty field reads as absent", `{"first_name":"Ada","last_name":""}`, map[string]string{ExternalSyncFirstName: "Ada"}},
		{"an unreadable record", `["nonsense"]`, map[string]string{}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := UnmarshalExternalSync([]byte(test.raw))
			if !maps.Equal(got, test.want) {
				t.Errorf("UnmarshalExternalSync(%q) = %v, want %v", test.raw, got, test.want)
			}
		})
	}
}
