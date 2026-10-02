package main

import (
	"testing"
	"time"

	bhmodels "github.com/abhinavxd/libredesk/internal/business_hours/models"
	"github.com/jmoiron/sqlx/types"
	"github.com/stretchr/testify/assert"
)

// Test: isOpenAt matches working hours and holidays, including ranges that cross midnight
func TestIsOpenAt(t *testing.T) {
	hours := bhmodels.BusinessHours{
		Hours: types.JSONText(`{
			"Monday": {"open": "09:00", "close": "18:00"},
			"Friday": {"open": "22:00", "close": "02:00"}
		}`),
		Holidays: types.JSONText(`[{"name": "Holiday", "date": "2026-10-12"}]`),
	}
	// 2026-10-05 is a Monday, 2026-10-09 a Friday, 2026-10-10 a Saturday.
	at := func(date, clock string) time.Time {
		ts, err := time.Parse("2006-01-02 15:04", date+" "+clock)
		assert.NoError(t, err)
		return ts
	}
	tests := []struct {
		name string
		when time.Time
		want bool
	}{
		{"inside weekday range", at("2026-10-05", "10:00"), true},
		{"before opening", at("2026-10-05", "08:59"), false},
		{"at closing time", at("2026-10-05", "18:00"), false},
		{"day without hours", at("2026-10-06", "10:00"), false},
		{"overnight range, evening", at("2026-10-09", "23:00"), true},
		{"overnight range spills into next day", at("2026-10-10", "01:00"), true},
		{"overnight range ends", at("2026-10-10", "02:00"), false},
		{"holiday", at("2026-10-12", "10:00"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := isOpenAt(hours, tt.when)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}

	t.Run("always open", func(t *testing.T) {
		got, err := isOpenAt(bhmodels.BusinessHours{IsAlwaysOpen: true}, at("2026-10-06", "03:00"))
		assert.NoError(t, err)
		assert.True(t, got)
	})
}
