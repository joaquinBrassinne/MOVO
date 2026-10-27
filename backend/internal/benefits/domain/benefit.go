// Package domain holds the rules of benefits and coupons: which benefits a business
// may offer, how long a coupon lives and how a monthly quota is counted. It imports
// no infrastructure.
package domain

import (
	"errors"
	"strconv"
	"strings"
	"time"
)

// Validation errors for benefits. The free texts report a *TextError instead.
var (
	ErrBenefitTypeInvalid  = errors.New("benefits: benefit type is not valid")
	ErrPercentInvalid      = errors.New("benefits: percent must be between 1 and 100")
	ErrAmountInvalid       = errors.New("benefits: amount must be greater than zero")
	ErrMonthlyLimitInvalid = errors.New("benefits: monthly limit must be greater than zero")
)

// Type is the kind of benefit a business offers.
type Type string

// Benefit types (SDD 5.2). The owner defines the benefit freely; 10, 15 and 20 are only
// shortcuts of the interface.
const (
	TypePercent Type = "percent"
	TypeAmount  Type = "amount"
	TypeCustom  Type = "custom"
)

const (
	maxTitleLen      = 80
	maxConditionsLen = 120
)

// argentina is the zone the monthly quota is counted in, like the daily metrics.
var argentina = time.FixedZone("ART", -3*60*60)

// Benefit is what a business gives to a customer who generates a coupon. Only the
// field of its type is set.
type Benefit struct {
	ID         int64
	BusinessID int64
	Type       Type
	Percent    int    // 1 to 100; percent only
	AmountARS  int    // pesos off; amount only
	Title      string // what is given; custom only
	Conditions string // optional, any type
	// Active is the owner's switch. AdminPaused is the super admin's: the owner cannot
	// lift it, so a benefit is offered only when both allow it.
	Active       bool
	AdminPaused  bool
	MonthlyLimit int // coupons the business accepts to generate per calendar month
}

// BenefitInput is what an owner provides to create or edit a benefit.
type BenefitInput struct {
	Type         Type
	Percent      int
	AmountARS    int
	Title        string
	Conditions   string
	MonthlyLimit int
}

// NewBenefit validates the input and builds an active benefit. BusinessID and ID are
// assigned by the use case and the repository.
func NewBenefit(in BenefitInput) (Benefit, error) {
	return Benefit{Active: true}.Update(in)
}

// Update validates the input and returns the benefit with the new details, keeping its
// identity and both switches. Only the field of the chosen type is kept, so a percent
// benefit never carries a title and the other way around.
func (b Benefit) Update(in BenefitInput) (Benefit, error) {
	if in.MonthlyLimit <= 0 {
		return Benefit{}, ErrMonthlyLimitInvalid
	}
	conditions, err := plainText("conditions", in.Conditions, false, maxConditionsLen)
	if err != nil {
		return Benefit{}, err
	}

	b.Percent, b.AmountARS, b.Title = 0, 0, ""
	switch in.Type {
	case TypePercent:
		if in.Percent < 1 || in.Percent > 100 {
			return Benefit{}, ErrPercentInvalid
		}
		b.Percent = in.Percent
	case TypeAmount:
		if in.AmountARS <= 0 {
			return Benefit{}, ErrAmountInvalid
		}
		b.AmountARS = in.AmountARS
	case TypeCustom:
		title, err := plainText("title", in.Title, true, maxTitleLen)
		if err != nil {
			return Benefit{}, err
		}
		b.Title = title
	default:
		return Benefit{}, ErrBenefitTypeInvalid
	}

	b.Type, b.Conditions, b.MonthlyLimit = in.Type, conditions, in.MonthlyLimit
	return b, nil
}

// ShortLabel is the label of cards and pills: "-15%", "-$2.000" or "Beneficio". The
// coupon and the redemption screen show the title and the conditions in full.
func (b Benefit) ShortLabel() string {
	switch b.Type {
	case TypePercent:
		return "-" + strconv.Itoa(b.Percent) + "%"
	case TypeAmount:
		return "-$" + thousands(b.AmountARS)
	default:
		return "Beneficio"
	}
}

// thousands writes n with a dot between groups of three digits, as pesos are written.
func thousands(n int) string {
	digits := strconv.Itoa(n)
	var out strings.Builder
	for i, d := range digits {
		if i > 0 && (len(digits)-i)%3 == 0 {
			out.WriteByte('.')
		}
		out.WriteRune(d)
	}
	return out.String()
}

// Activate turns the owner's switch on.
func (b Benefit) Activate() Benefit {
	b.Active = true
	return b
}

// Deactivate turns the owner's switch off without deleting the benefit: its coupons
// are history.
func (b Benefit) Deactivate() Benefit {
	b.Active = false
	return b
}

// PauseByAdmin hides the benefit whatever the owner does. Only the super admin calls it.
func (b Benefit) PauseByAdmin() Benefit {
	b.AdminPaused = true
	return b
}

// ResumeByAdmin lifts the admin's pause. It does not turn on what the owner turned off.
func (b Benefit) ResumeByAdmin() Benefit {
	b.AdminPaused = false
	return b
}

// Offered reports whether customers can see and claim the benefit: the owner has it on
// and the super admin has not paused it.
func (b Benefit) Offered() bool {
	return b.Active && !b.AdminPaused
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
