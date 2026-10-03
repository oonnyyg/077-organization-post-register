package org

import (
	"fmt"
	"time"
)

// ParseDate parses a strict YYYY-MM-DD calendar date.
func ParseDate(s string) (time.Time, error) {
	if len(s) != 10 {
		return time.Time{}, &ValidationError{msg: "invalid date: " + s + " (want YYYY-MM-DD)"}
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return time.Time{}, &ValidationError{msg: "invalid date: " + s + " (want YYYY-MM-DD)"}
	}
	return t, nil
}

// ValidationError marks an input error: the state is never modified.
type ValidationError struct{ msg string }

func (e *ValidationError) Error() string { return e.msg }

func failf(format string, args ...any) error {
	return &ValidationError{msg: fmt.Sprintf(format, args...)}
}

// intervalsOverlap reports whether two half-open intervals overlap.
// An empty end means open-ended. Adjacent intervals (b == c) do not overlap.
func intervalsOverlap(startA, endA, startB, endB time.Time) bool {
	return startA.Before(endOrMax(endB)) && startB.Before(endOrMax(endA))
}

func endOrMax(end time.Time) time.Time {
	if end.IsZero() {
		return farFuture
	}
	return end
}

var farFuture = time.Date(9999, 12, 31, 0, 0, 0, 0, time.UTC)
