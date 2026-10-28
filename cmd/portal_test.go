package main

import "testing"

func TestPortalAPIKeyMatches(t *testing.T) {
	configured := "01234567890123456789012345678901"
	tests := []struct {
		name     string
		provided string
		want     bool
	}{
		{"matching", configured, true},
		{"wrong key", "11234567890123456789012345678901", false},
		{"short key", configured[:31], false},
		{"empty key", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := portalAPIKeyMatches(tt.provided, configured); got != tt.want {
				t.Fatalf("portalAPIKeyMatches() = %v, want %v", got, tt.want)
			}
		})
	}
	if portalAPIKeyMatches(configured, "short") {
		t.Fatal("accepted configured API key shorter than 32 bytes")
	}
}
