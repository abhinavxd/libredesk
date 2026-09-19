package main

import (
	"testing"

	bhmodels "github.com/abhinavxd/libredesk/internal/business_hours/models"
)

func TestWithinWorkingHours(t *testing.T) {
	tests := []struct {
		name  string
		clock string
		day   bhmodels.WorkingHours
		want  bool
	}{
		{name: "normal open", clock: "09:00", day: bhmodels.WorkingHours{Open: "09:00", Close: "17:00"}, want: true},
		{name: "normal close", clock: "17:00", day: bhmodels.WorkingHours{Open: "09:00", Close: "17:00"}},
		{name: "overnight evening", clock: "23:00", day: bhmodels.WorkingHours{Open: "22:00", Close: "02:00"}, want: true},
		{name: "overnight morning", clock: "01:00", day: bhmodels.WorkingHours{Open: "22:00", Close: "02:00"}, want: true},
		{name: "overnight closed", clock: "12:00", day: bhmodels.WorkingHours{Open: "22:00", Close: "02:00"}},
		{name: "same time closed", clock: "09:00", day: bhmodels.WorkingHours{Open: "09:00", Close: "09:00"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := withinWorkingHours(tc.clock, tc.day); got != tc.want {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}
