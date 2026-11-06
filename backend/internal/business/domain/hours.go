package domain

import (
	"errors"
	"fmt"
	"sort"
)

// Validation errors for opening hours.
var (
	ErrClockInvalid = errors.New("business: time of day is not valid")
	ErrSlotInvalid  = errors.New("business: time slot is not valid")
	ErrSlotsOverlap = errors.New("business: time slots overlap")
)

// Clock is a time of day as minutes since midnight, from 00:00 to 23:59.
type Clock int

const maxClock = 23*60 + 59

// ParseClock reads a strict "HH:MM" 24-hour time.
func ParseClock(raw string) (Clock, error) {
	if len(raw) != 5 || raw[2] != ':' {
		return 0, ErrClockInvalid
	}
	var hours, minutes int
	if _, err := fmt.Sscanf(raw, "%02d:%02d", &hours, &minutes); err != nil {
		return 0, ErrClockInvalid
	}
	if hours > 23 || minutes > 59 {
		return 0, ErrClockInvalid
	}
	return Clock(hours*60 + minutes), nil
}

// String renders the clock as "HH:MM".
func (c Clock) String() string { return fmt.Sprintf("%02d:%02d", int(c)/60, int(c)%60) }

// Slot is one opening interval on a weekday (0 = Sunday .. 6 = Saturday).
type Slot struct {
	Weekday int
	Start   Clock
	End     Clock
}

// ValidateSlots checks that every slot is a well-formed interval and that no two
// slots of the same weekday overlap. Touching slots (one ends when the next
// starts) are allowed, and an empty list means the branch is closed.
func ValidateSlots(slots []Slot) error {
	for _, s := range slots {
		if s.Weekday < 0 || s.Weekday > 6 || s.Start < 0 || s.End > maxClock || s.End <= s.Start {
			return ErrSlotInvalid
		}
	}

	sorted := append([]Slot(nil), slots...)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Weekday != sorted[j].Weekday {
			return sorted[i].Weekday < sorted[j].Weekday
		}
		return sorted[i].Start < sorted[j].Start
	})
	for i := 1; i < len(sorted); i++ {
		if sorted[i].Weekday == sorted[i-1].Weekday && sorted[i].Start < sorted[i-1].End {
			return ErrSlotsOverlap
		}
	}
	return nil
}
