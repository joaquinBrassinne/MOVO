package search

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
	_ "time/tzdata" // the zone must resolve on hosts without a system tz database
)

// ZoneName is the zone the opening hours are written in: a branch's "09:00" is a
// wall clock in Córdoba, whatever the server's own zone is.
const ZoneName = "America/Argentina/Cordoba"

// zone is Argentina's wall clock. It has had no daylight saving since 2009, so the
// fixed offset is only the fallback if the zone data cannot be loaded.
var zone = func() *time.Location {
	if loc, err := time.LoadLocation(ZoneName); err == nil {
		return loc
	}
	return time.FixedZone("ART", -3*60*60)
}()

// OpenStatus answers "is it open right now?" for one branch. When Open, ClosesAt is
// when the current stretch ends (touching slots count as one stretch). When closed,
// OpensAt and OpensWeekday say when it opens next; HasNext is false when the branch
// has no slots at all.
type OpenStatus struct {
	Open         bool
	ClosesAt     string
	HasNext      bool
	OpensAt      string
	OpensWeekday int
}

// StatusAt computes the status of a branch with the given weekly slots at the
// instant now, read in the Córdoba zone. A slot is half open, [start, end): at
// exactly the closing time the branch is already closed.
func StatusAt(slots []Slot, now time.Time) OpenStatus {
	local := now.In(zone)
	today := int(local.Weekday())
	minute := local.Hour()*60 + local.Minute()

	days := stretchesByDay(slots)
	for _, s := range days[today] {
		if s.start <= minute && minute < s.end {
			return OpenStatus{Open: true, ClosesAt: clock(s.end)}
		}
	}

	// Later today first, then each following day; offset 7 is the same weekday next week.
	for offset := 0; offset <= 7; offset++ {
		day := (today + offset) % 7
		for _, s := range days[day] {
			if offset > 0 || s.start > minute {
				return OpenStatus{HasNext: true, OpensAt: clock(s.start), OpensWeekday: day}
			}
		}
	}
	return OpenStatus{}
}

type stretch struct{ start, end int }

// stretchesByDay turns the slots into minutes since midnight, ordered, with slots
// that touch or overlap merged so "closes at" is the real end of the stretch.
func stretchesByDay(slots []Slot) map[int][]stretch {
	byDay := map[int][]stretch{}
	for _, s := range slots {
		start, okStart := minutes(s.Start)
		end, okEnd := minutes(s.End)
		if okStart && okEnd && end > start {
			byDay[s.Weekday] = append(byDay[s.Weekday], stretch{start, end})
		}
	}
	for day, list := range byDay {
		sort.Slice(list, func(i, j int) bool { return list[i].start < list[j].start })
		merged := list[:1]
		for _, s := range list[1:] {
			if last := &merged[len(merged)-1]; s.start <= last.end {
				last.end = max(last.end, s.end)
			} else {
				merged = append(merged, s)
			}
		}
		byDay[day] = merged
	}
	return byDay
}

func minutes(hhmm string) (int, bool) {
	h, m, found := strings.Cut(hhmm, ":")
	hours, errH := strconv.Atoi(h)
	mins, errM := strconv.Atoi(m)
	if !found || errH != nil || errM != nil {
		return 0, false
	}
	return hours*60 + mins, true
}

func clock(minute int) string { return fmt.Sprintf("%02d:%02d", minute/60, minute%60) }
