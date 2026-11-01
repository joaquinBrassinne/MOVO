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

// PreviousRedemptionsLimit is how many past redemptions a preview lists. The count says
// how many there were; the list shows the newest ones, which is what the owner needs to
// notice a customer who comes back for the same thing again and again.
const PreviousRedemptionsLimit = 10

// PastRedemption is a coupon the same customer already redeemed at the business. Its
// benefit carries the terms the coupon had when it was redeemed, not the benefit's today.
type PastRedemption struct {
	Benefit    domain.Benefit
	BranchName string
	RedeemedAt time.Time
}

// CouponPreview is what an owner may see of a coupon before confirming its redemption:
// the benefit, the customer's name and the dates. It never carries the phone.
type CouponPreview struct {
	Coupon       domain.Coupon // Status is the stored one; use StatusAt for what to show
	Benefit      domain.Benefit
	CustomerName string
	RedeemedAt   *time.Time // nil until redeemed
	// CustomerRedeemed is how many coupons this customer already redeemed at the business.
	CustomerRedeemed int
	// PreviousRedemptions lists, newest first and up to PreviousRedemptionsLimit, the other
	// coupons this customer redeemed at the business. The coupon being previewed is not in it.
	PreviousRedemptions []PastRedemption
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
	// Save persists the details of an existing benefit and never the active flag, so an
	// edit cannot undo a concurrent on or off; ErrBenefitNotFound as in Find.
	Save(ctx context.Context, b domain.Benefit) error
	// SetActive flips the owner's switch alone and returns the benefit as stored;
	// ErrBenefitNotFound as in Find.
	SetActive(ctx context.Context, businessID, benefitID int64, active bool) (domain.Benefit, error)
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

// QuotaView is the monthly coupon quota of a business as the panel shows it: the limit,
// how many coupons were generated this month and when the count starts again.
type QuotaView struct {
	domain.Quota
	ResetsOn time.Time
}

// CouponQuotas reads and sets the monthly cap on the coupons a business generates. The
// month is the calendar month in Argentina, given as [monthStart, monthEnd).
type CouponQuotas interface {
	// Quota returns the limit of the business and what it generated in the month, or
	// ErrBusinessNotFound.
	Quota(ctx context.Context, businessID int64, monthStart, monthEnd time.Time) (domain.Quota, error)
	// SetQuotaLimit sets the limit (nil removes it) and returns the quota. It locks the
	// business row, like generating a coupon does, so the check against what was already
	// generated cannot be overtaken by a coupon that is being generated. It returns
	// domain.ErrQuotaLimitInvalid or a *domain.QuotaBelowGeneratedError when the limit is
	// not allowed.
	SetQuotaLimit(ctx context.Context, businessID int64, limit *int, monthStart, monthEnd time.Time) (domain.Quota, error)
}

// Usage is what one benefit did in a calendar month: coupons generated in it and coupons
// redeemed in it, whenever they were generated.
type Usage struct {
	Generated int
	Redeemed  int
}

// BenefitUsage reads what each benefit of a business did in a month, for the panel.
type BenefitUsage interface {
	// MonthlyUsage returns the usage by benefit id for [monthStart, monthEnd). A benefit
	// with no coupons is absent.
	MonthlyUsage(ctx context.Context, businessID int64, monthStart, monthEnd time.Time) (map[int64]Usage, error)
}

// BenefitWithUsage is a benefit and what it did this month.
type BenefitWithUsage struct {
	domain.Benefit
	Usage
}

// Overview is the handful of numbers on the home of the panel (SDD 4.9b): coupons that
// can still be redeemed, coupons redeemed today, and the views of the profile and the
// clicks to WhatsApp of the last 7 days (today and the six before). Days are Argentina's.
type Overview struct {
	ActiveCoupons  int
	RedeemedToday  int
	ProfileViews   int
	WhatsAppClicks int
}

// OverviewReader reads the numbers of the home of the panel. It only counts.
type OverviewReader interface {
	Overview(ctx context.Context, businessID int64, now time.Time) (Overview, error)
}

// ErrMetricsPeriodInvalid means the requested number of days is not one the panel offers.
var ErrMetricsPeriodInvalid = errors.New("benefits: metrics period is not valid")

// MetricsPeriod is the window of the metrics: the last Days days, today included, on
// Argentina's calendar. [From, To) are the instants for the coupon tables; FromDay and ToDay
// are the same window as inclusive dates (YYYY-MM-DD) for metric_daily.
type MetricsPeriod struct {
	Days    int
	From    time.Time
	To      time.Time
	FromDay string
	ToDay   string
}

// EssentialMetrics are the numbers every plan sees (SDD 4.3): the views of the profile, and
// the coupons generated and redeemed in the period. They count events, never people.
type EssentialMetrics struct {
	ProfileViews     int64
	CouponsGenerated int64
	CouponsRedeemed  int64
}

// MinimumSample is how many coupons a business must have generated in a period before the
// Pro metrics show percentages (SDD 4.3). Below it a rate says nothing.
const MinimumSample = 20

// ConversionCounts are the raw counts behind the conversion rates of one period. The
// coupon counts follow a cohort: of the coupons generated in the period, how many have been
// redeemed so far.
type ConversionCounts struct {
	ProfileViews     int64
	CouponsGenerated int64
	CouponsRedeemed  int64
}

// BranchConversion is what one branch got in the period: the taps on its WhatsApp button and
// the coupons redeemed there (redeemed_branch_id).
type BranchConversion struct {
	BranchID       int64
	Name           string
	WhatsAppClicks int64
	Redemptions    int64
}

// ActivityConversion is what one activity got in the period: the times it was looked at and
// the taps on WhatsApp from it.
type ActivityConversion struct {
	ServiceID      int64
	Name           string
	Views          int64
	WhatsAppClicks int64
}

// Conversion is everything the conversion metric needs: the counts of the period and of the
// one before it, and the breakdown of the period by branch and by activity.
type Conversion struct {
	Current    ConversionCounts
	Previous   ConversionCounts
	ByBranch   []BranchConversion
	ByActivity []ActivityConversion
}

// ConversionReader reads the conversion counts of a business. It only counts.
type ConversionReader interface {
	Conversion(ctx context.Context, businessID int64, current, previous MetricsPeriod) (Conversion, error)
}

// BenefitCounts is what became of the coupons of one benefit generated in a period. Expired
// counts the coupons marked expired and the pending ones whose time is already over, so it does
// not wait for any background task; the rest of the generated ones are still pending.
type BenefitCounts struct {
	BenefitID int64
	Generated int64
	Redeemed  int64
	Expired   int64
}

// Performance is the counts per benefit of a period and of the one before it.
type Performance struct {
	Current  []BenefitCounts
	Previous []BenefitCounts
}

// PerformanceReader reads the counts per benefit. It only counts.
type PerformanceReader interface {
	// BenefitPerformance counts the coupons by benefit for each period; now decides which
	// pending coupons count as expired.
	BenefitPerformance(ctx context.Context, businessID int64, current, previous MetricsPeriod, now time.Time) (Performance, error)
}

// HeatCell is how many coupons were redeemed on one weekday (ISO: Monday 1 to Sunday 7) in
// one time slot (0 to 3, see queries.HeatSlots), on Argentina's clock.
type HeatCell struct {
	Weekday     int
	Slot        int
	Redemptions int64
}

// Heatmap is the redemptions of a period by weekday and slot, with the totals the metric
// needs. Cells lists only the ones with redemptions.
type Heatmap struct {
	// CouponsGenerated is the sample: the coupons generated in the period.
	CouponsGenerated    int64
	Redemptions         int64
	PreviousRedemptions int64
	Cells               []HeatCell
}

// HeatmapReader reads the redemption heat map of a business. It only counts.
type HeatmapReader interface {
	Heatmap(ctx context.Context, businessID int64, current, previous MetricsPeriod) (Heatmap, error)
}

// ActivityDemand is how much one activity (a service) was asked for in a period: the times
// it was looked at and the times someone tapped WhatsApp from it. Counts only.
type ActivityDemand struct {
	ServiceID int64
	Name      string
	Views     int64
	Clicks    int64
}

// DemandReader ranks the activities of a business by demand.
type DemandReader interface {
	// TopActivities returns at most limit activities, the most asked for first (views plus
	// clicks, then name). Services deleted by the owner are left out.
	TopActivities(ctx context.Context, businessID int64, period MetricsPeriod, limit int) ([]ActivityDemand, error)
}

// ErrPlanUpgradeRequired means the business's plan does not include the Pro metrics.
var ErrPlanUpgradeRequired = errors.New("benefits: the plan does not include the advanced metrics")

// MetricsPermission tells whether a business may read the Pro metrics ("inteligencia
// comercial", SDD 4.3). The composition root answers it from the business's entitlements,
// so a pilot gets what its pilot plan gives and nothing here ever asks which plan it is.
type MetricsPermission interface {
	AdvancedMetrics(ctx context.Context, businessID int64) (bool, error)
}

// MetricsReader reads the metrics of a business over a period. It only counts.
type MetricsReader interface {
	Essential(ctx context.Context, businessID int64, period MetricsPeriod) (EssentialMetrics, error)
}
