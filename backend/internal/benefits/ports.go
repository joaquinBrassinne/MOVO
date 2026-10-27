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

// Errors of redeeming a coupon.
var (
	// ErrCouponNotFound covers a code that does not exist, is malformed or belongs to
	// another business: the three answer the same, so a business cannot probe the
	// codes of the others.
	ErrCouponNotFound = errors.New("benefits: coupon not found")
	// ErrCouponRedeemed means the coupon of this business was already redeemed.
	ErrCouponRedeemed = errors.New("benefits: coupon was already redeemed")
	// ErrCouponExpired means the coupon of this business is past its 48 hours.
	ErrCouponExpired = errors.New("benefits: coupon has expired")
	// ErrBranchRequired means the business has several active branches (or none) and the
	// redemption did not say where it happened.
	ErrBranchRequired = errors.New("benefits: the branch of the redemption must be given")
	// ErrBranchNotFound means the branch is not an active branch of the business.
	ErrBranchNotFound = errors.New("benefits: branch not found")
)

// RedeemRequest is one redemption. BusinessID and UserID come from the session, never
// from the request body.
type RedeemRequest struct {
	BusinessID int64
	UserID     int64
	Code       string
	// BranchID is where the customer redeemed; 0 means the only active branch.
	BranchID int64
	Now      time.Time
}

// Redeemed is the coupon that was redeemed, with what the business needs to hand over
// the benefit. It never carries the customer's phone.
type Redeemed struct {
	Coupon       domain.Coupon
	Benefit      domain.Benefit
	CustomerName string
	BranchID     int64
	RedeemedAt   time.Time
}

// CouponRedeemer redeems coupons. Redeem is a single conditional UPDATE, so of any
// number of simultaneous redemptions of one code exactly one succeeds.
type CouponRedeemer interface {
	// Redeem returns ErrCouponNotFound, ErrCouponRedeemed, ErrCouponExpired,
	// ErrBranchRequired or ErrBranchNotFound.
	Redeem(ctx context.Context, req RedeemRequest) (Redeemed, error)
}

// CouponPreview is what an owner may see of a coupon before confirming its redemption:
// the benefit, the customer's name and the dates. It never carries the phone.
type CouponPreview struct {
	Coupon       domain.Coupon // Status is the stored one; use StatusAt for what to show
	Benefit      domain.Benefit
	CustomerName string
	RedeemedAt   *time.Time // nil until redeemed
}

// HeldCoupon is a coupon as its customer sees it: the preview plus the business that
// offers it. It never carries the phone.
type HeldCoupon struct {
	CouponPreview
	BusinessName string
}

// CouponReader reads coupons. It never changes anything.
type CouponReader interface {
	// Preview returns the coupon of the business with that code, or ErrCouponNotFound
	// when there is none, including a code that belongs to another business.
	Preview(ctx context.Context, businessID int64, code string) (CouponPreview, error)
	// ByToken returns the one coupon whose access token has that hash, or
	// ErrCouponNotFound. A token opens its own coupon only, never the other coupons of
	// the same customer: the phone is not verified, so it proves nothing about who holds it.
	ByToken(ctx context.Context, tokenHash []byte) (HeldCoupon, error)
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

// CouponExpirer marks overdue coupons in bulk. Reads already show an overdue coupon as
// expired, so this only keeps the stored status honest for reports.
type CouponExpirer interface {
	// ExpireOverdue marks every pending coupon whose time is up at now as expired and
	// returns how many. It never touches a redeemed coupon and is safe to repeat.
	ExpireOverdue(ctx context.Context, now time.Time) (int64, error)
}

// ActivityKind is what happened to a coupon of the business.
type ActivityKind string

// The events the panel's "Actividad reciente" lists. Reviews join them in a later phase.
const (
	ActivityGenerated ActivityKind = "coupon_generated"
	ActivityRedeemed  ActivityKind = "coupon_redeemed"
)

// ActivityEvent is one thing that happened. It names the customer as the business
// already knows them from the coupon, and never carries the phone.
type ActivityEvent struct {
	Kind         ActivityKind
	At           time.Time
	CustomerName string
	Benefit      domain.Benefit
}

// Activity is the newest events and how many of all of them happened after the mark the
// panel last saw.
type Activity struct {
	Events   []ActivityEvent
	NewCount int
}

// ActivityReader reads what happened at a business. It never changes anything.
type ActivityReader interface {
	// RecentActivity returns up to limit events, newest first. NewCount counts every
	// event after since, whatever the limit; with a nil since it is 0.
	RecentActivity(ctx context.Context, businessID int64, since *time.Time, limit int) (Activity, error)
}

// BenefitModerator lets the super admin pause any benefit, whichever business owns it.
type BenefitModerator interface {
	// SetAdminPaused sets the pause and records actorID in the audit log, atomically,
	// and returns the benefit. Repeating the current state changes nothing. It returns
	// ErrBenefitNotFound when there is no such benefit.
	SetAdminPaused(ctx context.Context, actorID, benefitID int64, paused bool) (domain.Benefit, error)
}
