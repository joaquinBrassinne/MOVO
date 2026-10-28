package domain

import (
	"errors"
	"strconv"
	"time"
)

// ErrQuotaLimitInvalid means the limit is not a whole number greater than zero.
var ErrQuotaLimitInvalid = errors.New("benefits: quota limit must be greater than zero")

// QuotaBelowGeneratedError means the new limit is lower than the coupons the business
// already generated this month. It carries the number so the owner sees why.
type QuotaBelowGeneratedError struct {
	Generated int
}

func (e *QuotaBelowGeneratedError) Error() string {
	return "benefits: quota limit is below the " + strconv.Itoa(e.Generated) + " coupons already generated this month"
}

// Quota is the monthly cap on the coupons a business generates, and how many it has
// generated in the current calendar month (Argentina time). A nil Limit means no limit.
type Quota struct {
	Limit     *int
	Generated int
}

// HasRoom reports whether one more coupon can be generated this month.
func (q Quota) HasRoom() bool {
	return q.Limit == nil || q.Generated < *q.Limit
}

// ValidateQuotaLimit checks a limit an owner wants to set: absent (no limit) or greater
// than zero, and not below what the business already generated this month.
func ValidateQuotaLimit(limit *int, generated int) error {
	if limit == nil {
		return nil
	}
	if *limit <= 0 {
		return ErrQuotaLimitInvalid
	}
	if *limit < generated {
		return &QuotaBelowGeneratedError{Generated: generated}
	}
	return nil
}

// QuotaResetsOn is the first day of the month after the one that contains now, at
// midnight in Argentina: when the count starts again.
func QuotaResetsOn(now time.Time) time.Time {
	_, end := QuotaMonth(now)
	return end
}

// DateInArgentina writes t as a calendar date (YYYY-MM-DD) in Argentina time.
func DateInArgentina(t time.Time) string {
	return t.In(argentina).Format(time.DateOnly)
}

// DayInArgentina returns the half-open interval [start, end) of the calendar day, in
// Argentina time, that contains now.
func DayInArgentina(now time.Time) (start, end time.Time) {
	local := now.In(argentina)
	start = time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, argentina)
	return start, start.AddDate(0, 0, 1)
}
