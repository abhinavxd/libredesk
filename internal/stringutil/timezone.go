package stringutil

import (
	"strings"
	"time"
)

// IsValidTimezone reports whether tz is an IANA timezone name. "Local" is rejected because Go
// resolves it to the host zone but Postgres' AT TIME ZONE does not accept it.
func IsValidTimezone(tz string) bool {
	tz = strings.TrimSpace(tz)
	if tz == "" || tz == "Local" {
		return false
	}
	_, err := time.LoadLocation(tz)
	return err == nil
}

// NormalizeTimezone returns tz if valid, otherwise UTC.
func NormalizeTimezone(tz string) string {
	tz = strings.TrimSpace(tz)
	if IsValidTimezone(tz) {
		return tz
	}
	return "UTC"
}
