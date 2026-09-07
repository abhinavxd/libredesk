package stringutil

import (
	"testing"
	"time"
)

// frontendPickerTimezones mirrors frontend/apps/main/src/constants/timezones.js (the values the FE can send on save).
var frontendPickerTimezones = []string{
	"UTC",
	"America/New_York",
	"America/Chicago",
	"America/Denver",
	"America/Los_Angeles",
	"America/Toronto",
	"America/Mexico_City",
	"America/Bogota",
	"America/Sao_Paulo",
	"America/Buenos_Aires",
	"America/Santiago",
	"Europe/London",
	"Europe/Berlin",
	"Europe/Paris",
	"Europe/Rome",
	"Europe/Madrid",
	"Europe/Moscow",
	"Europe/Istanbul",
	"Asia/Dubai",
	"Asia/Kolkata",
	"Asia/Bangkok",
	"Asia/Singapore",
	"Asia/Shanghai",
	"Asia/Seoul",
	"Asia/Tokyo",
	"Australia/Sydney",
	"Australia/Melbourne",
	"Australia/Perth",
	"Pacific/Auckland",
	"Pacific/Honolulu",
	"Africa/Cairo",
	"Africa/Lagos",
	"Africa/Nairobi",
	"Africa/Johannesburg",
}

func TestFrontendPickerTimezonesAreValid(t *testing.T) {
	for _, tz := range frontendPickerTimezones {
		// Save-time validation must accept it.
		if !IsValidTimezone(tz) {
			t.Errorf("frontend picker offers %q but IsValidTimezone rejects it", tz)
		}
		// It must actually load in Go, since the date filter relies on it at query time.
		if _, err := time.LoadLocation(tz); err != nil {
			t.Errorf("frontend picker offers %q but time.LoadLocation fails: %v", tz, err)
		}
		if got := NormalizeTimezone(tz); got != tz {
			t.Errorf("NormalizeTimezone(%q) = %q, want it unchanged", tz, got)
		}
	}
}

func TestNormalizeTimezoneFallsBackToUTC(t *testing.T) {
	cases := []string{
		"",
		"   ",
		"Local",                  // Go resolves it, Postgres rejects it
		"asia/kolkata",           // wrong case, IANA is case-sensitive
		"Mars/Olympus",           // not a real zone
		"'; DROP TABLE users;--", // injection attempt
		"UTC+5",                  // not an IANA name
	}
	for _, tz := range cases {
		if IsValidTimezone(tz) {
			t.Errorf("expected %q to be invalid", tz)
		}
		if got := NormalizeTimezone(tz); got != "UTC" {
			t.Errorf("NormalizeTimezone(%q) = %q, want UTC", tz, got)
		}
	}
}

func TestNormalizeTimezoneTrimsWhitespace(t *testing.T) {
	if got := NormalizeTimezone("  Asia/Kolkata  "); got != "Asia/Kolkata" {
		t.Errorf("NormalizeTimezone trimmed = %q, want Asia/Kolkata", got)
	}
}
