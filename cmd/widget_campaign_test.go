package main

import (
	"testing"
	"time"
)

func TestParseCampaignCooldown(t *testing.T) {
	tests := []struct {
		value string
		want  time.Duration
		valid bool
	}{
		{value: "0s", valid: true},
		{value: "10m", want: 10 * time.Minute, valid: true},
		{value: "1h30m", want: 90 * time.Minute, valid: true},
		{value: ""},
		{value: "tomorrow"},
		{value: "-1m"},
	}

	for _, tc := range tests {
		t.Run(tc.value, func(t *testing.T) {
			got, err := parseCampaignCooldown(tc.value)
			if tc.valid && (err != nil || got != tc.want) {
				t.Fatalf("got %v, %v; want %v", got, err, tc.want)
			}
			if !tc.valid && err == nil {
				t.Fatalf("accepted %q", tc.value)
			}
		})
	}
}
