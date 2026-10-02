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
		if t, err := a.team.Get(teamID); err == nil {
			hoursID, timezone = t.BusinessHoursID.Int, t.Timezone
		}
	}
	if hoursID == 0 || timezone == "" {
		raw, err := a.settings.GetByPrefix("app")
		if err != nil {
			return false, err
		}
		var out map[string]any
		if err := json.Unmarshal([]byte(raw), &out); err != nil {
			return false, fmt.Errorf("parsing settings: %w", err)
		}
		idStr, _ := out["app.business_hours_id"].(string)
		hoursID, _ = strconv.Atoi(idStr)
		timezone, _ = out["app.timezone"].(string)
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
	local := at.In(loc)
	for _, h := range holidays {
		if h.Date == local.Format(time.DateOnly) {
			return false, nil
		}
	}
	day, ok := working[local.Weekday().String()]
	if !ok {
		return false, nil
	}
	return withinWorkingHours(local.Format("15:04"), day), nil
}
