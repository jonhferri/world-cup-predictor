package forecast

import (
	"testing"
	"time"
)

func TestIsLockedAtTournamentStartBoundary(t *testing.T) {
	start := time.Date(2026, time.June, 11, 19, 0, 0, 0, time.UTC)

	tests := []struct {
		name string
		now  time.Time
		want bool
	}{
		{name: "before first kickoff remains editable", now: start.Add(-time.Nanosecond), want: false},
		{name: "at first kickoff still editable", now: start, want: false},
		{name: "just before 24h deadline still editable", now: start.Add(24*time.Hour - time.Nanosecond), want: false},
		{name: "at 24h after first kickoff locks forecast", now: start.Add(24 * time.Hour), want: true},
		{name: "after deadline stays locked", now: start.Add(25 * time.Hour), want: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := isLocked(tc.now, start); got != tc.want {
				t.Fatalf("isLocked() = %v, want %v", got, tc.want)
			}
		})
	}
}
