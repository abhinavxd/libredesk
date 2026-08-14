package main

import "testing"

func TestOIDCNames(t *testing.T) {
	tests := []struct {
		name, full, given, family, wantGiven, wantFamily string
	}{
		{name: "structured claims", full: "Ignored Name", given: "Ada", family: "Lovelace", wantGiven: "Ada", wantFamily: "Lovelace"},
		{name: "full name fallback", full: "Grace Brewster Hopper", wantGiven: "Grace", wantFamily: "Brewster Hopper"},
		{name: "empty claim fallback", wantGiven: "User"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			given, family := oidcNames(test.full, test.given, test.family)
			if given != test.wantGiven || family != test.wantFamily {
				t.Fatalf("got %q %q, want %q %q", given, family, test.wantGiven, test.wantFamily)
			}
		})
	}
}
