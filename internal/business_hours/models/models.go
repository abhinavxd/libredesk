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
	for _, holiday := range holidays {
		if holiday.Date == local.Format(time.DateOnly) {
			return false, nil
		}
	}
	var schedule map[string]WorkingHours
	if err := json.Unmarshal(b.Hours, &schedule); err != nil {
		return false, fmt.Errorf("decoding working hours: %w", err)
	}
	day, ok := schedule[local.Weekday().String()]
	if !ok {
		return false, nil
	}
	open, err := time.Parse("15:04", day.Open)
	if err != nil {
		return false, fmt.Errorf("parsing opening time: %w", err)
	}
	close, err := time.Parse("15:04", day.Close)
	if err != nil {
		return false, fmt.Errorf("parsing closing time: %w", err)
	}
	start, end := open.Hour()*60+open.Minute(), close.Hour()*60+close.Minute()
	minute := local.Hour()*60 + local.Minute()
	switch {
	case start == end:
		return false, nil
	case start < end:
		return minute >= start && minute < end, nil
	default:
		return minute >= start || minute < end, nil
	}
}
