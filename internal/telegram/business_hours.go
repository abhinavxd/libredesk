package telegram

import (
	"encoding/json"
	"fmt"
	"time"

	bhmodels "github.com/abhinavxd/libredesk/internal/business_hours/models"
)

func WithinBusinessHours(now time.Time, hours bhmodels.BusinessHours, timezone string) (bool, error) {
	if hours.IsAlwaysOpen {
		return true, nil
	}
	location, err := time.LoadLocation(timezone)
	if err != nil {
		return false, err
	}
	local := now.In(location)
	var holidays []bhmodels.Holiday
	if len(hours.Holidays) > 0 {
		if err := json.Unmarshal(hours.Holidays, &holidays); err != nil {
			return false, err
		}
	}
	for _, holiday := range holidays {
		if holiday.Date == local.Format(time.DateOnly) {
			return false, nil
		}
	}
	var schedule map[string]bhmodels.WorkingHours
	if err := json.Unmarshal(hours.Hours, &schedule); err != nil {
		return false, err
	}
	day, ok := schedule[local.Weekday().String()]
	if !ok {
		return false, nil
	}
	open, err := time.Parse("15:04", day.Open)
	if err != nil {
		return false, err
	}
	close, err := time.Parse("15:04", day.Close)
	if err != nil {
		return false, err
	}
	start, end := open.Hour()*60+open.Minute(), close.Hour()*60+close.Minute()
	if end <= start {
		return false, fmt.Errorf("business hours must close after opening")
	}
	minute := local.Hour()*60 + local.Minute()
	return minute >= start && minute < end, nil
}
