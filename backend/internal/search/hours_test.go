package search_test

import (
	"testing"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/search"
)

// at is an instant on the given weekday (0 = Sunday) and wall-clock time in Córdoba,
// in the week of Monday 2026-10-26.
func at(t *testing.T, weekday int, hhmm string) time.Time {
	t.Helper()
	zone, err := time.LoadLocation(search.ZoneName)
	if err != nil {
		t.Fatalf("load zone: %v", err)
	}
	clock, err := time.Parse("15:04", hhmm)
	if err != nil {
		t.Fatalf("parse %q: %v", hhmm, err)
	}
	return time.Date(2026, 10, 25+weekday, clock.Hour(), clock.Minute(), 0, 0, zone)
}

func TestStatusAt(t *testing.T) {
	t.Parallel()

	split := []search.Slot{{Weekday: 1, Start: "09:00", End: "13:00"}, {Weekday: 1, Start: "16:00", End: "20:00"}}
	touching := []search.Slot{{Weekday: 1, Start: "09:00", End: "12:00"}, {Weekday: 1, Start: "12:00", End: "18:00"}}
	oneSlot := []search.Slot{{Weekday: 1, Start: "09:00", End: "12:00"}}
	laterInWeek := []search.Slot{{Weekday: 3, Start: "09:00", End: "13:00"}}
	twoDays := []search.Slot{{Weekday: 1, Start: "09:00", End: "20:00"}, {Weekday: 2, Start: "08:00", End: "12:00"}}

	tests := []struct {
		name  string
		slots []search.Slot
		now   time.Time
		want  search.OpenStatus
	}{
		{"inside a slot", split, at(t, 1, "10:30"), search.OpenStatus{Open: true, ClosesAt: "13:00"}},
		{"second slot of a split day", split, at(t, 1, "17:00"), search.OpenStatus{Open: true, ClosesAt: "20:00"}},
		{"the instant it opens", split, at(t, 1, "09:00"), search.OpenStatus{Open: true, ClosesAt: "13:00"}},
		{"the instant it closes is already closed", oneSlot, at(t, 1, "12:00"), search.OpenStatus{HasNext: true, OpensAt: "09:00", OpensWeekday: 1}},
		{"between the slots of a split day", split, at(t, 1, "14:00"), search.OpenStatus{HasNext: true, OpensAt: "16:00", OpensWeekday: 1}},
		{"before opening", split, at(t, 1, "07:00"), search.OpenStatus{HasNext: true, OpensAt: "09:00", OpensWeekday: 1}},
		{"touching slots are one stretch", touching, at(t, 1, "12:00"), search.OpenStatus{Open: true, ClosesAt: "18:00"}},
		{"after the last slot it opens the next day", twoDays, at(t, 1, "21:00"), search.OpenStatus{HasNext: true, OpensAt: "08:00", OpensWeekday: 2}},
		{"a day without slots looks for the next opening", laterInWeek, at(t, 2, "10:00"), search.OpenStatus{HasNext: true, OpensAt: "09:00", OpensWeekday: 3}},
		{"after the only slot of the week it opens next week", oneSlot, at(t, 1, "15:00"), search.OpenStatus{HasNext: true, OpensAt: "09:00", OpensWeekday: 1}},
		{"no slots at all is closed with no next opening", nil, at(t, 1, "10:00"), search.OpenStatus{}},
		{"the week wraps from Saturday to Sunday", []search.Slot{{Weekday: 0, Start: "10:00", End: "14:00"}}, at(t, 6, "12:00"), search.OpenStatus{HasNext: true, OpensAt: "10:00", OpensWeekday: 0}},
		{"an instant in UTC is read in Córdoba", split, at(t, 1, "10:30").UTC(), search.OpenStatus{Open: true, ClosesAt: "13:00"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := search.StatusAt(tt.slots, tt.now); got != tt.want {
				t.Errorf("StatusAt = %+v, want %+v", got, tt.want)
			}
		})
	}
}
