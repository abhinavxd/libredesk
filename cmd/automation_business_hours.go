package main

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	businesshours "github.com/abhinavxd/libredesk/internal/business_hours"
	bhmodels "github.com/abhinavxd/libredesk/internal/business_hours/models"
	"github.com/abhinavxd/libredesk/internal/setting"
	"github.com/abhinavxd/libredesk/internal/team"
)

// automationBusinessHours tells the automation engine whether support is open, using the team's business hours and timezone with the helpdesk defaults as fallback.
type automationBusinessHours struct {
	team          *team.Manager
	settings      *setting.Manager
	businessHours *businesshours.Manager
}

// IsOpen reports whether support is open at the given time for the team.
func (a *automationBusinessHours) IsOpen(teamID int, at time.Time) (bool, error) {
	var (
		hoursID  int
		timezone string
	)
	if teamID != 0 {
		t, err := a.team.Get(teamID)
		if err != nil {
			return false, fmt.Errorf("fetching team %d: %w", teamID, err)
		}
		hoursID, timezone = t.BusinessHoursID.Int, t.Timezone
	}
	// Each value falls back to the helpdesk default on its own.
	if hoursID == 0 || timezone == "" {
		raw, err := a.settings.GetByPrefix("app")
		if err != nil {
			return false, err
		}
		var out map[string]any
		if err := json.Unmarshal([]byte(raw), &out); err != nil {
			return false, fmt.Errorf("parsing settings: %w", err)
		}
		if hoursID == 0 {
			idStr, _ := out["app.business_hours_id"].(string)
			hoursID, _ = strconv.Atoi(idStr)
		}
		if timezone == "" {
			timezone, _ = out["app.timezone"].(string)
		}
	}
	if hoursID == 0 || timezone == "" {
		return false, fmt.Errorf("business hours or timezone not configured")
	}
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return false, err
	}
	hours, err := a.businessHours.Get(hoursID)
	if err != nil {
		return false, err
	}
	return isOpenAt(hours, at.In(loc))
}

// isOpenAt reports whether the business hours are open at local, which must already be in the schedule's timezone.
func isOpenAt(hours bhmodels.BusinessHours, local time.Time) (bool, error) {
	if hours.IsAlwaysOpen {
		return true, nil
	}
	var (
		holidays []bhmodels.Holiday
		working  map[string]bhmodels.WorkingHours
	)
	if len(hours.Holidays) > 0 {
		if err := json.Unmarshal(hours.Holidays, &holidays); err != nil {
			return false, err
		}
	}
	if err := json.Unmarshal(hours.Hours, &working); err != nil {
		return false, err
	}
	isHoliday := func(t time.Time) bool {
		for _, h := range holidays {
			if h.Date == t.Format(time.DateOnly) {
				return true
			}
		}
		return false
	}
	clock := local.Format("15:04")
	if !isHoliday(local) {
		if day, ok := working[local.Weekday().String()]; ok && withinWorkingHours(clock, day) {
			return true, nil
		}
	}
	// A range that crosses midnight keeps the previous day open into this morning.
	prev := local.AddDate(0, 0, -1)
	if day, ok := working[prev.Weekday().String()]; ok && !isHoliday(prev) && day.Open > day.Close && clock < day.Close {
		return true, nil
	}
	return false, nil
}
