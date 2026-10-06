package telegram

import (
	bhmodels "github.com/abhinavxd/libredesk/internal/business_hours/models"
	"github.com/jmoiron/sqlx/types"
	"testing"
	"time"
)

func TestWithinBusinessHours(t *testing.T) {
	schedule := types.JSONText(`{"Tuesday":{"open":"09:00","close":"17:00"},"Sunday":{"open":"01:00","close":"04:00"}}`)
	for _, tc := range []struct {
		name, instant, timezone, hours, holidays string
		always, want, fail                       bool
	}{
		{name: "opening boundary", instant: "2026-09-08T03:30:00Z", timezone: "Asia/Kolkata", want: true},
		{name: "before opening", instant: "2026-09-08T03:29:59Z", timezone: "Asia/Kolkata"},
		{name: "closing boundary", instant: "2026-09-08T11:30:00Z", timezone: "Asia/Kolkata"},
		{name: "before closing", instant: "2026-09-08T11:29:59Z", timezone: "Asia/Kolkata", want: true},
		{name: "closed weekday", instant: "2026-09-09T10:00:00Z", timezone: "UTC"},
		{name: "holiday", instant: "2026-09-08T10:00:00Z", timezone: "UTC", holidays: `[{"date":"2026-09-08"}]`},
		{name: "other holiday", instant: "2026-09-08T10:00:00Z", timezone: "UTC", holidays: `[{"date":"2026-09-09"}]`, want: true},
		{name: "DST spring", instant: "2026-03-08T07:30:00Z", timezone: "America/New_York", want: true},
		{name: "DST fall first hour", instant: "2026-11-01T05:30:00Z", timezone: "America/New_York", want: true},
		{name: "DST fall repeated hour", instant: "2026-11-01T06:30:00Z", timezone: "America/New_York", want: true},
		{name: "always open", always: true, want: true},
		{name: "invalid timezone", timezone: "bad", fail: true},
		{name: "invalid schedule", timezone: "UTC", hours: `{`, fail: true},
		{name: "invalid holidays", timezone: "UTC", holidays: `{`, fail: true},
		{name: "invalid opening", timezone: "UTC", hours: `{"Tuesday":{"open":"bad","close":"17:00"}}`, fail: true},
		{name: "invalid closing", timezone: "UTC", hours: `{"Tuesday":{"open":"09:00","close":"bad"}}`, fail: true},
		{name: "reversed hours", timezone: "UTC", hours: `{"Tuesday":{"open":"17:00","close":"09:00"}}`, fail: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			now := time.Date(2026, 9, 8, 10, 0, 0, 0, time.UTC)
			if tc.instant != "" {
				now, _ = time.Parse(time.RFC3339, tc.instant)
			}
			hours := schedule
			if tc.hours != "" {
				hours = types.JSONText(tc.hours)
			}
			got, err := WithinBusinessHours(now, bhmodels.BusinessHours{IsAlwaysOpen: tc.always, Hours: hours, Holidays: types.JSONText(tc.holidays)}, tc.timezone)
			if (err != nil) != tc.fail || got != tc.want {
				t.Fatalf("open=%v err=%v", got, err)
			}
		})
	}
}
