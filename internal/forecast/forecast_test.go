package forecast

import (
	"testing"
	"time"
)

func TestIsLockedAtForecastDeadline(t *testing.T) {
	loc, err := time.LoadLocation("Europe/Paris")
	if err != nil {
		t.Skip("tzdata not available")
	}
	deadline := time.Date(2026, time.June, 14, 23, 59, 0, 0, loc)

	tests := []struct {
		name string
		now  time.Time
		want bool
	}{
		{name: "1ns before deadline is editable", now: deadline.Add(-time.Nanosecond), want: false},
		{name: "at deadline is locked", now: deadline, want: true},
		{name: "after deadline stays locked", now: deadline.Add(time.Hour), want: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := isLocked(tc.now, time.Time{}); got != tc.want {
				t.Fatalf("isLocked() = %v, want %v", got, tc.want)
			}
		})
	}
}
