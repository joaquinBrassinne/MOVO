// Package domain holds the rules of benefits and coupons: which benefits a business
// may offer, how long a coupon lives and how a monthly quota is counted. It imports
// no infrastructure.
package domain

import (
	"errors"
	"strings"
	"time"
	"unicode/utf8"
)

// Validation errors for benefits.
var (
	ErrBenefitTypeInvalid     = errors.New("benefits: benefit type is not valid")
	ErrPercentInvalid         = errors.New("benefits: percent must be 10, 15 or 20")
	ErrGiftDescriptionInvalid = errors.New("benefits: gift description is not valid")
	ErrMonthlyLimitInvalid    = errors.New("benefits: monthly limit must be greater than zero")
)

// Type is the kind of benefit a business offers.
type Type string

// Benefit types (SDD 4).
const (
	TypePercent     Type = "percent"
	TypeWelcomeGift Type = "welcome_gift"
)

const maxGiftDescriptionLen = 200

// argentina is the zone the monthly quota is counted in, like the daily metrics.
var argentina = time.FixedZone("ART", -3*60*60)

// Benefit is what a business gives to a customer who generates a coupon.
type Benefit struct {
	ID              int64
	BusinessID      int64
	Type            Type
	Percent         int    // 10, 15 or 20; 0 for a welcome gift
	GiftDescription string // empty for a percent benefit
	Active          bool
	MonthlyLimit    int // coupons the business accepts to generate per calendar month
}

// BenefitInput is what an owner provides to create or edit a benefit.
type BenefitInput struct {
	Type            Type
	Percent         int
	GiftDescription string
	MonthlyLimit    int
}

// NewBenefit validates the input and builds an active benefit. BusinessID and ID are
// assigned by the use case and the repository.
func NewBenefit(in BenefitInput) (Benefit, error) {
	return Benefit{Active: true}.Update(in)
}

// Update validates the input and returns the benefit with the new details, keeping its
// identity and whether it is active. Only the field of the chosen type is kept, so a
// percent benefit never carries a gift and the other way around.
func (b Benefit) Update(in BenefitInput) (Benefit, error) {
	if in.MonthlyLimit <= 0 {
		return Benefit{}, ErrMonthlyLimitInvalid
	}

	switch in.Type {
	case TypePercent:
		if in.Percent != 10 && in.Percent != 15 && in.Percent != 20 {
			return Benefit{}, ErrPercentInvalid
		}
		b.Percent, b.GiftDescription = in.Percent, ""
	case TypeWelcomeGift:
		gift := strings.TrimSpace(in.GiftDescription)
		if gift == "" || utf8.RuneCountInString(gift) > maxGiftDescriptionLen {
			return Benefit{}, ErrGiftDescriptionInvalid
		}
		b.Percent, b.GiftDescription = 0, gift
	default:
		return Benefit{}, ErrBenefitTypeInvalid
	}

	b.Type, b.MonthlyLimit = in.Type, in.MonthlyLimit
	return b, nil
}

// Activate turns the benefit on.
func (b Benefit) Activate() Benefit {
	b.Active = true
	return b
}

// Deactivate turns the benefit off without deleting it: its coupons are history.
func (b Benefit) Deactivate() Benefit {
	b.Active = false
	return b
}

// HasRoom reports whether the business can still generate a coupon this month, given
// how many it already generated. The limit counts coupons generated, not redeemed.
func (b Benefit) HasRoom(generatedThisMonth int) bool {
	return generatedThisMonth < b.MonthlyLimit
}

// QuotaMonth returns the half-open interval [start, end) of the calendar month, in
// Argentina time, that contains now. Coupons generated inside it count against the
// monthly limit.
func QuotaMonth(now time.Time) (start, end time.Time) {
	local := now.In(argentina)
	start = time.Date(local.Year(), local.Month(), 1, 0, 0, 0, 0, argentina)
	return start, start.AddDate(0, 1, 0)
}
