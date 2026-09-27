// Package models contains the data models for the businesshours package.
package models

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx/types"
	"github.com/volatiletech/null/v9"
)

// BusinessHours represents the business in the database.
type BusinessHours struct {
	ID           int            `db:"id" json:"id"`
	CreatedAt    time.Time      `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time      `db:"updated_at" json:"updated_at"`
	Name         string         `db:"name" json:"name"`
	Description  null.String    `db:"description" json:"description"`
	IsAlwaysOpen bool           `db:"is_always_open" json:"is_always_open"`
	Holidays     types.JSONText `db:"holidays" json:"holidays"`
	Hours        types.JSONText `db:"hours" json:"hours"`
}

// WorkingHours represents the working hours for a specific day.
type WorkingHours struct {
	Open  string `json:"open"`
	Close string `json:"close"`
}

// Holiday represents a holiday.
type Holiday struct {
	Name string `json:"name"`
	Date string `json:"date"`
}

// IsOpen reports whether now falls inside the schedule. A day whose close time is before its open time runs past midnight.
func (b BusinessHours) IsOpen(now time.Time, loc *time.Location) (bool, error) {
	if b.IsAlwaysOpen {
		return true, nil
	}
	local := now.In(loc)
	var holidays []Holiday
	if len(b.Holidays) > 0 {
		if err := json.Unmarshal(b.Holidays, &holidays); err != nil {
			return false, fmt.Errorf("decoding holidays: %w", err)
		}
	}
	var schedule map[string]WorkingHours
	if err := json.Unmarshal(b.Hours, &schedule); err != nil {
		return false, fmt.Errorf("decoding working hours: %w", err)
	}
	minute := local.Hour()*60 + local.Minute()
	start, end, err := dayWindow(schedule, holidays, local)
	if err != nil {
		return false, err
	}
	if start < end && minute >= start && minute < end || start > end && minute >= start {
		return true, nil
	}
	start, end, err = dayWindow(schedule, holidays, local.AddDate(0, 0, -1))
	if err != nil {
		return false, err
	}
	return start > end && minute < end, nil
}

// dayWindow returns the opening and closing minute of the interval that opens on day, or equal values when it has none.
func dayWindow(schedule map[string]WorkingHours, holidays []Holiday, day time.Time) (int, int, error) {
	for _, holiday := range holidays {
		if holiday.Date == day.Format(time.DateOnly) {
			return 0, 0, nil
		}
	}
	hours, ok := schedule[day.Weekday().String()]
	if !ok {
		return 0, 0, nil
	}
	open, err := time.Parse("15:04", hours.Open)
	if err != nil {
		return 0, 0, fmt.Errorf("parsing opening time: %w", err)
	}
	close, err := time.Parse("15:04", hours.Close)
	if err != nil {
		return 0, 0, fmt.Errorf("parsing closing time: %w", err)
	}
	return open.Hour()*60 + open.Minute(), close.Hour()*60 + close.Minute(), nil
}
