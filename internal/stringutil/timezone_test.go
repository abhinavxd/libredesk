package stringutil

import (
	"encoding/json/v2"
	"os"
	"slices"
	"testing"
	"time"
)

var legacyPickerTimezones = []string{
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
	"Asia/Kathmandu",
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
	data, err := os.ReadFile("../../frontend/shared-ui/constants/timezones.json")
	if err != nil {
		t.Fatal(err)
	}
	var timezones []string
	if err := json.Unmarshal(data, &timezones); err != nil {
		t.Fatal(err)
	}
	if len(timezones) < 400 {
		t.Fatalf("expected the complete timezone list, got %d entries", len(timezones))
	}
	for _, tz := range legacyPickerTimezones {
		if !slices.Contains(timezones, tz) {
			t.Errorf("previously offered timezone %q is missing", tz)
		}
	}
	for _, tz := range timezones {
		if !IsValidTimezone(tz) {
			t.Errorf("frontend picker offers %q but IsValidTimezone rejects it", tz)
		}
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
