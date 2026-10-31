package domain

import (
	"errors"
	"strings"
	"time"
	"unicode/utf8"
)

// GraceDays is how many days a business stays visible after paid_until. The same
// number lives in the visible_branches view (migration 00023); a migration test pins
// that both agree, because a SQL view cannot read configuration.
const GraceDays = 7

// ExpiringSoonDays is the window of the "por vencer" list (SDD 5.4).
const ExpiringSoonDays = 7

// ErrPaidUntilInvalid means a validity date is the zero value.
var ErrPaidUntilInvalid = errors.New("billing: paid until is not a valid date")

// argentina is the zone every validity date is counted in. Argentina has no
// daylight saving time, so a fixed offset is exact and needs no tzdata.
var argentina = time.FixedZone("ART", -3*60*60)

// Day is a calendar date in Argentina time. The zero Day is not valid.
type Day struct{ t time.Time }

// DayOf is the Argentine calendar date of an instant.
func DayOf(instant time.Time) Day {
	y, m, d := instant.In(argentina).Date()
	return Day{time.Date(y, m, d, 0, 0, 0, 0, time.UTC)}
}

// DayFromDate reads a SQL date: the calendar date is taken as stored, with no zone
// conversion (a date has no time zone).
func DayFromDate(date time.Time) Day {
	y, m, d := date.Date()
	return Day{time.Date(y, m, d, 0, 0, 0, 0, time.UTC)}
}

// NewDay builds a date from its parts, or ErrPaidUntilInvalid when it does not exist.
func NewDay(year int, month time.Month, day int) (Day, error) {
	t := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
	if t.Year() != year || t.Month() != month || t.Day() != day {
		return Day{}, ErrPaidUntilInvalid
	}
	return Day{t}, nil
}

// IsZero reports whether the day is unset.
func (d Day) IsZero() bool { return d.t.IsZero() }

// Time is the date at midnight UTC, the shape a SQL date round-trips through.
func (d Day) Time() time.Time { return d.t }

// String is the ISO date, e.g. 2026-11-30.
func (d Day) String() string { return d.t.Format(time.DateOnly) }

// AddDays moves the date by n days (n may be negative).
func (d Day) AddDays(n int) Day { return Day{d.t.AddDate(0, 0, n)} }

// AddMonth moves the date one calendar month, clamping to the last day of the
// shorter month: January 31 becomes February 28 (or 29), never March 3.
func (d Day) AddMonth() Day {
	y, m, day := d.t.Date()
	next := time.Date(y, m+1, 1, 0, 0, 0, 0, time.UTC)
	last := time.Date(next.Year(), next.Month()+1, 0, 0, 0, 0, 0, time.UTC).Day()
	return Day{time.Date(next.Year(), next.Month(), min(day, last), 0, 0, 0, 0, time.UTC)}
}

// Before reports whether d is earlier than other.
func (d Day) Before(other Day) bool { return d.t.Before(other.t) }

// After reports whether d is later than other.
func (d Day) After(other Day) bool { return d.t.After(other.t) }

// DaysUntil is how many days from d to other (negative when other is earlier).
func (d Day) DaysUntil(other Day) int { return int(other.t.Sub(d.t).Hours() / 24) }

// Status is where a business stands with its payments, derived from the dates.
type Status string

// The derived states. None means a business with no subscription that is not a pilot.
const (
	StatusPilot   Status = "pilot"
	StatusActive  Status = "active"
	StatusGrace   Status = "grace"
	StatusExpired Status = "expired"
	StatusNone    Status = "none"
)

// Subscription is the current payment state of one business.
type Subscription struct {
	BusinessID int64
	Plan       Plan
	PaidUntil  Day
	// MPSubscriptionRef is the optional reference shown in the Mercado Pago panel.
	MPSubscriptionRef string
	Notes             string
}

// NewSubscription validates a subscription the super admin is about to create.
func NewSubscription(businessID int64, plan Plan, paidUntil Day) (Subscription, error) {
	if _, err := ParsePlan(string(plan)); err != nil {
		return Subscription{}, err
	}
	if paidUntil.IsZero() {
		return Subscription{}, ErrPaidUntilInvalid
	}
	return Subscription{BusinessID: businessID, Plan: plan, PaidUntil: paidUntil}, nil
}

// Limits of the free-text fields, the same as the CHECKs of the subscriptions table.
const (
	maxRefLen   = 100
	maxNotesLen = 500
)

// Errors for the free-text fields of a subscription.
var (
	ErrRefInvalid   = errors.New("billing: Mercado Pago reference must be at most 100 characters")
	ErrNotesInvalid = errors.New("billing: notes must be at most 500 characters")
)

// WithDetails sets the optional Mercado Pago reference and the admin's notes. Both
// are trimmed; empty means absent.
func (s Subscription) WithDetails(ref, notes string) (Subscription, error) {
	ref, notes = strings.TrimSpace(ref), strings.TrimSpace(notes)
	if utf8.RuneCountInString(ref) > maxRefLen {
		return Subscription{}, ErrRefInvalid
	}
	if utf8.RuneCountInString(notes) > maxNotesLen {
		return Subscription{}, ErrNotesInvalid
	}
	s.MPSubscriptionRef, s.Notes = ref, notes
	return s, nil
}

// GraceEnd is the last day the business is still visible.
func (s Subscription) GraceEnd() Day { return s.PaidUntil.AddDays(GraceDays) }

// StatusAt derives the state on a given day: active up to and including
// paid_until, in grace for the next GraceDays days, expired after that.
func (s Subscription) StatusAt(today Day) Status {
	switch {
	case !today.After(s.PaidUntil):
		return StatusActive
	case !today.After(s.GraceEnd()):
		return StatusGrace
	default:
		return StatusExpired
	}
}

// ExpiringSoon reports whether the subscription is active and ends within
// ExpiringSoonDays (today included). A subscription already in grace is not
// "por vencer": it is late, and the panel shows it as such.
func (s Subscription) ExpiringSoon(today Day) bool {
	return s.StatusAt(today) == StatusActive && today.DaysUntil(s.PaidUntil) <= ExpiringSoonDays
}

// RegisterPayment extends the validity by one month. A subscription that is still
// current is extended from its paid_until, so paying early loses nothing; one that
// already lapsed is extended from today, so the owner does not get days for free.
func (s Subscription) RegisterPayment(today Day) Subscription {
	base := s.PaidUntil
	if base.Before(today) {
		base = today
	}
	s.PaidUntil = base.AddMonth()
	return s
}

// ChangePlan moves the subscription to another plan without touching its dates.
func (s Subscription) ChangePlan(plan Plan) (Subscription, error) {
	if _, err := ParsePlan(string(plan)); err != nil {
		return Subscription{}, err
	}
	s.Plan = plan
	return s, nil
}

// StatusOf derives the state of a business: a pilot is a pilot whatever its
// payments, a business with no subscription has none, otherwise the dates decide.
func StatusOf(pilot bool, sub *Subscription, today Day) Status {
	switch {
	case pilot:
		return StatusPilot
	case sub == nil:
		return StatusNone
	default:
		return sub.StatusAt(today)
	}
}

// Visible reports whether a business may appear in public search, once its own
// status is already active (SDD 4.1): a pilot, or a subscription still active or
// in grace. It mirrors the visible_branches view.
func Visible(pilot bool, sub *Subscription, today Day) bool {
	switch StatusOf(pilot, sub, today) {
	case StatusPilot, StatusActive, StatusGrace:
		return true
	default:
		return false
	}
}
