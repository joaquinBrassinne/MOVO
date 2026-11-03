package main

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/joaquinBrassinne/MOVO/internal/benefits"
	benefitshttp "github.com/joaquinBrassinne/MOVO/internal/benefits/adapters/http"
	benefitspostgres "github.com/joaquinBrassinne/MOVO/internal/benefits/adapters/postgres"
	"github.com/joaquinBrassinne/MOVO/internal/benefits/adapters/ratelimit"
	"github.com/joaquinBrassinne/MOVO/internal/benefits/app/commands"
	"github.com/joaquinBrassinne/MOVO/internal/benefits/app/queries"
	"github.com/joaquinBrassinne/MOVO/internal/business"
	businesspostgres "github.com/joaquinBrassinne/MOVO/internal/business/adapters/postgres"
	"github.com/joaquinBrassinne/MOVO/internal/platform/cache"
	"github.com/joaquinBrassinne/MOVO/internal/platform/httpx"
)

// ownerBusinesses answers which business an owner may change. It joins the two
// modules here so benefits never reads businesses.
type ownerBusinesses struct {
	businesses *businesspostgres.Businesses
}

func (o ownerBusinesses) EditableBusinessOf(ctx context.Context, ownerID int64) (int64, error) {
	b, err := o.businesses.FindByOwner(ctx, ownerID)
	if errors.Is(err, business.ErrBusinessNotFound) {
		return 0, benefits.ErrBusinessNotFound
	}
	if err != nil {
		return 0, err
	}
	if !b.Editable() {
		return 0, benefits.ErrBusinessNotEditable
	}
	return b.ID, nil
}

// Coupon request budget per origin: a customer needs one coupon, so a handful per hour
// covers a household or a gym's shared Wi-Fi, and a script guessing ids or numbers is
// stopped quickly. How many is COUPON_RATE_LIMIT_ATTEMPTS (config.Coupons); the window and
// the table size are fixed here. The monthly quota of the business remains the cap on the
// damage.
const (
	couponWindow  = time.Hour
	couponMaxKeys = 100_000
)

// metricsPlanPort is what the Pro metrics ask of the plan of a business: whether it includes
// them, and the figures the estimated return multiplies.
type metricsPlanPort interface {
	benefits.MetricsPermission
	benefits.BusinessEconomics
}

// newBenefitsRoutes composes the benefits module: adapters, use cases and HTTP. With a Redis
// store the coupon limit is counted in Redis, shared by every instance; nil keeps it in memory.
func newBenefitsRoutes(log *slog.Logger, pool *pgxpool.Pool, verifier httpx.TokenVerifier, couponAttempts int, plan metricsPlanPort, redisStore *cache.Redis) func(chi.Router) {
	owners := ownerBusinesses{businesses: businesspostgres.NewBusinesses(pool)}
	// Every Pro metric goes through this one gate.
	gate := queries.NewProGate(owners, plan)
	repo := benefitspostgres.NewBenefits(pool)
	coupons := benefitspostgres.NewCoupons(pool)
	couponLimit := ratelimit.Config{MaxAttempts: couponAttempts, Window: couponWindow, MaxKeys: couponMaxKeys}
	var limiter benefits.GenerationLimiter
	if redisStore == nil {
		limiter = ratelimit.New(couponLimit, nil)
	} else {
		limiter = ratelimit.NewRedis(redisStore, "coupon", couponLimit, log)
	}
	return benefitshttp.New(benefitshttp.Deps{
		Verifier:         verifier,
		Log:              log,
		List:             queries.NewListMyBenefits(owners, repo, coupons, nil),
		Add:              commands.NewAddBenefit(owners, repo),
		Edit:             commands.NewEditBenefit(owners, repo),
		SetActive:        commands.NewSetBenefitActive(owners, repo),
		Pause:            commands.NewPauseBenefit(repo),
		Overview:         queries.NewGetOverview(owners, coupons, nil),
		Metrics:          queries.NewGetMetrics(owners, coupons, nil),
		Activities:       queries.NewGetTopActivities(owners, coupons, nil),
		Conversion:       queries.NewGetConversion(gate, coupons, nil),
		Heatmap:          queries.NewGetHeatmap(gate, coupons, nil),
		Performance:      queries.NewGetBenefitPerformance(gate, repo, coupons, nil),
		TimeToRedeem:     queries.NewGetTimeToRedeem(gate, coupons, nil),
		Returning:        queries.NewGetRepeatCustomers(gate, coupons, nil),
		QuotaProjection:  queries.NewGetQuotaProjection(gate, coupons, nil),
		EstimatedReturn:  queries.NewGetEstimatedReturn(gate, plan, coupons, nil),
		BranchComparison: queries.NewGetBranchComparison(gate, coupons, nil),
		ReviewsOverTime:  queries.NewGetReviewsOverTime(gate, coupons, nil),
		Records:          queries.NewGetRecords(gate, repo, coupons, nil),
		Quota:            queries.NewGetCouponQuota(owners, coupons, nil),
		SetQuota:         commands.NewSetCouponQuota(owners, coupons, nil),
		Generate:         commands.NewGenerateCode(coupons, limiter, nil, nil),
		Redeem:           commands.NewRedeemCode(owners, coupons, nil),
		Preview:          queries.NewPreviewCode(owners, coupons, nil),
		View:             queries.NewViewMyCoupon(coupons, nil),
		Activity:         queries.NewRecentActivity(owners, coupons),
	}).Mount
}
