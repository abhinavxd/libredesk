package main

import (
	"testing"
	"time"

	bh "github.com/abhinavxd/libredesk/internal/business_hours"
	bhmodels "github.com/abhinavxd/libredesk/internal/business_hours/models"
	"github.com/abhinavxd/libredesk/internal/inbox/channel/livechat/proactive"
	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/jmoiron/sqlx/types"
	"github.com/volatiletech/null/v9"
)

func TestWithinWorkingHours(t *testing.T) {
	tests := []struct {
		name     string
		clock    string
		day      bhmodels.WorkingHours
		previous bhmodels.WorkingHours
		want     bool
	}{
		{name: "normal open", clock: "09:00", day: bhmodels.WorkingHours{Open: "09:00", Close: "17:00"}, want: true},
		{name: "normal close", clock: "17:00", day: bhmodels.WorkingHours{Open: "09:00", Close: "17:00"}},
		{name: "overnight evening", clock: "23:00", day: bhmodels.WorkingHours{Open: "22:00", Close: "02:00"}, want: true},
		{name: "overnight morning", clock: "01:00", previous: bhmodels.WorkingHours{Open: "22:00", Close: "02:00"}, want: true},
		{name: "overnight midnight", clock: "00:00", previous: bhmodels.WorkingHours{Open: "22:00", Close: "02:00"}, want: true},
		{name: "overnight close", clock: "02:00", previous: bhmodels.WorkingHours{Open: "22:00", Close: "02:00"}},
		{name: "today overnight has not opened", clock: "01:00", day: bhmodels.WorkingHours{Open: "22:00", Close: "02:00"}},
		{name: "previous daytime does not carry over", clock: "01:00", previous: bhmodels.WorkingHours{Open: "09:00", Close: "17:00"}},
		{name: "today daytime and previous overnight", clock: "01:00", day: bhmodels.WorkingHours{Open: "09:00", Close: "17:00"}, previous: bhmodels.WorkingHours{Open: "22:00", Close: "02:00"}, want: true},
		{name: "overnight closed", clock: "12:00", day: bhmodels.WorkingHours{Open: "22:00", Close: "02:00"}},
		{name: "same time closed", clock: "09:00", day: bhmodels.WorkingHours{Open: "09:00", Close: "09:00"}},
		{name: "no schedule", clock: "01:00"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := withinWorkingHours(tc.clock, tc.day, tc.previous); got != tc.want {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestCampaignWithinHours(t *testing.T) {
	db := testutil.NewDB(t, "widget_campaign_hours")
	i18n := testutil.NewI18n(t)
	manager, err := bh.New(bh.Opts{DB: db, I18n: i18n})
	if err != nil {
		t.Fatal(err)
	}
	hours, err := manager.Create("Overnight", null.String{} /* description */, false /* isAlwaysOpen */, types.JSONText(`{"Friday":{"open":"22:00","close":"02:00"},"Sunday":{"open":"22:00","close":"02:00"},"Monday":{"open":"09:00","close":"17:00"}}`), types.JSONText(`[{"date":"2026-10-31"},{"date":"2026-11-06"},{"date":"2026-11-08"}]`))
	if err != nil {
		t.Fatal(err)
	}
	app := &App{businessHours: manager, i18n: i18n}
	campaign := proactive.Campaign{BusinessHours: "inside", BusinessHoursID: hours.ID}
	loc, err := time.LoadLocation("Asia/Kolkata")
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name string
		now  string
		want bool
	}{
		{name: "Friday opening", now: "2026-10-23T16:30:00Z", want: true},
		{name: "Saturday early morning without Saturday hours", now: "2026-10-23T19:30:00Z", want: true},
		{name: "Saturday closing", now: "2026-10-23T20:30:00Z"},
		{name: "Friday early morning before opening", now: "2026-10-22T19:30:00Z"},
		{name: "Sunday overnight carries into Monday", now: "2026-10-25T19:30:00Z", want: true},
		{name: "holiday blocks previous overnight hours", now: "2026-10-30T19:30:00Z"},
		{name: "holiday cancels evening shift", now: "2026-11-06T17:30:00Z"},
		{name: "holiday shift cannot carry into midnight", now: "2026-11-06T18:30:00Z"},
		{name: "holiday shift cannot carry into an unscheduled day", now: "2026-11-06T19:30:00Z"},
		{name: "holiday shift cannot carry into a scheduled day", now: "2026-11-08T19:30:00Z"},
		{name: "day after holiday opens at its own hours", now: "2026-11-09T03:30:00Z", want: true},
		{name: "next week overnight shift remains open", now: "2026-11-13T19:30:00Z", want: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			now, err := time.Parse(time.RFC3339, tc.now)
			if err != nil {
				t.Fatal(err)
			}
			got, err := campaignWithinHours(app, campaign, now, loc)
			if err != nil || got != tc.want {
				t.Fatalf("got %v, %v, want %v", got, err, tc.want)
			}
		})
	}
}

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
