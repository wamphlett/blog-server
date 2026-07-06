package scheduler

import (
	"testing"
	"time"
)

// verifies that the next execution time is calculated in the location of the
// configured time (e.g. UTC) rather than the server's local timezone, and is
// always in the future
func TestCalculateNextTime_RespectsConfiguredLocation(t *testing.T) {
	s := &Scheduler{timeToExecute: time.Date(0, 0, 0, 0, 1, 0, 0, time.UTC)}

	next := s.calculateNextTime()

	utcNext := next.In(time.UTC)
	if utcNext.Hour() != 0 || utcNext.Minute() != 1 {
		t.Fatalf("expected next run at 00:01 UTC, got %s", utcNext)
	}

	if !next.After(time.Now()) {
		t.Fatalf("expected next run to be in the future, got %s", next)
	}

	if time.Until(next) > 24*time.Hour {
		t.Fatalf("expected next run within 24 hours, got %s", next)
	}
}
