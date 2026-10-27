// Package benefits is the module that lets a business offer benefits and lets a
// customer turn them into coupons. This file declares the ports the application
// layer consumes; adapters implement them.
package benefits

import (
	"context"
	"errors"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/benefits/domain"
)

// Errors the adapters report so the application layer can react without knowing the
// storage engine.
var (
	ErrBenefitNotFound = errors.New("benefits: benefit not found")
	// ErrBusinessNotFound means the owner has not created a business yet.
	ErrBusinessNotFound = errors.New("benefits: owner has no business")
	// ErrBusinessNotEditable means the business was deleted and can no longer change.
	ErrBusinessNotEditable = errors.New("benefits: business can no longer change")
)

// Errors of issuing a coupon.
var (
	// ErrQuotaReached means the benefit already generated its monthly limit of coupons.
	ErrQuotaReached = errors.New("benefits: monthly limit of coupons reached")
	// ErrPendingCoupon means the customer already holds a pending coupon at the business.
	ErrPendingCoupon = errors.New("benefits: customer already has a pending coupon at this business")
	// ErrCodeTaken means the random code collided with an existing one; the caller
	// draws another and tries again.
	ErrCodeTaken = errors.New("benefits: coupon code is already taken")
)

// IssueRequest is everything needed to generate one coupon. The code, the token hash
// and the clock come from the caller so the adapter stays a plain transaction.
type IssueRequest struct {
	BenefitID int64
	Contact   domain.Contact
	Code      string
	TokenHash []byte
	Now       time.Time
	ExpiresAt time.Time
	// MonthStart and MonthEnd bound the calendar month the monthly limit counts in.
	MonthStart, MonthEnd time.Time
}

// Issued is the coupon that was generated and the benefit it claims.
type Issued struct {
	Coupon  domain.Coupon
	Benefit domain.Benefit
}

// CouponIssuer generates coupons. Issue is one transaction: it finds the benefit, which
// must be active and belong to a visible business, files the customer, enforces the
// one-pending-coupon rule and the monthly limit, and stores the coupon, so two
// customers racing for the last slot cannot both get it.
type CouponIssuer interface {
	// Issue returns ErrBenefitNotFound when the benefit is inactive, does not exist or
	// its business is not visible; ErrPendingCoupon; ErrQuotaReached; or ErrCodeTaken.
	Issue(ctx context.Context, req IssueRequest) (Issued, error)
}

// GenerationLimiter caps how often one origin may ask for coupons (SDD 5.2): with no
// verification of the customer, the only brakes are this limit and the monthly quota.
// The in-memory adapter is per process; the port lets Redis replace it when the API
// runs several instances.
type GenerationLimiter interface {
	// Allow counts one attempt of key, successful or not, and reports whether it may
	// go on. When it may not, retryAfter is how long until the key can try again.
	Allow(key string) (retryAfter time.Duration, ok bool)
}

// RateLimitedError is returned when an origin asked for coupons too often.
type RateLimitedError struct {
	RetryAfter time.Duration
}

func (e *RateLimitedError) Error() string {
	return "benefits: too many coupon requests, retry in " + e.RetryAfter.Round(time.Second).String()
}

// OwnerBusinesses resolves the business an owner may change. The composition root
// builds it from the business module, so benefits never reads businesses and the
// business id always comes from the session, never from a request.
type OwnerBusinesses interface {
	// EditableBusinessOf returns the id of the owner's business, ErrBusinessNotFound
	// when there is none, or ErrBusinessNotEditable when it was deleted.
	EditableBusinessOf(ctx context.Context, ownerID int64) (int64, error)
}

// BenefitRepository stores benefits. Every method is scoped by business id, so a
// benefit id of another business behaves as if it did not exist.
type BenefitRepository interface {
	// Create stores a new benefit and returns it with ID set.
	Create(ctx context.Context, b domain.Benefit) (domain.Benefit, error)
	// List returns every benefit of the business, oldest first, inactive ones included.
	List(ctx context.Context, businessID int64) ([]domain.Benefit, error)
	// Find returns ErrBenefitNotFound when the business has no such benefit.
	Find(ctx context.Context, businessID, benefitID int64) (domain.Benefit, error)
	// Save persists the details and the active flag; ErrBenefitNotFound as in Find.
	Save(ctx context.Context, b domain.Benefit) error
}
