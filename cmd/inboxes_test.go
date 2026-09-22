package main

import (
	"strings"
	"testing"
)

func TestValidateQuickReplies(t *testing.T) {
	app := newValidatorTestApp(t)
	tests := []struct {
		name    string
		replies []string
		wantErr bool
	}{
		{name: "unicode limit", replies: []string{strings.Repeat("界", 120)}},
		{name: "unicode over limit", replies: []string{strings.Repeat("界", 121)}, wantErr: true},
		{name: "blank", replies: []string{"  "}, wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateQuickReplies(app, tc.replies)
			if (err != nil) != tc.wantErr {
				t.Fatalf("got error %v, want error %v", err, tc.wantErr)
			}
		})
	}
}

func TestValidateLiveChatSessionDuration(t *testing.T) {
	app := newValidatorTestApp(t)
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{name: "legacy empty"},
		{name: "one hour", value: "1h"},
		{name: "one hour in minutes", value: "60m"},
		{name: "one hour in seconds", value: "3600s"},
		{name: "combined duration", value: "1h30m"},
		{name: "zero", value: "0", wantErr: true},
		{name: "below one hour", value: "59m59s", wantErr: true},
		{name: "below one hour in seconds", value: "3599s", wantErr: true},
		{name: "negative", value: "-1h", wantErr: true},
		{name: "invalid", value: "soon", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateLiveChatSessionDuration(app, tc.value)
			if (err != nil) != tc.wantErr {
				t.Fatalf("got error %v, want error %v", err, tc.wantErr)
			}
		})
	}
}
