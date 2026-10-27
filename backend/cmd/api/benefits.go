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

// newBenefitsRoutes composes the benefits module: adapters, use cases and HTTP.
func newBenefitsRoutes(log *slog.Logger, pool *pgxpool.Pool, verifier httpx.TokenVerifier, couponAttempts int) func(chi.Router) {
	owners := ownerBusinesses{businesses: businesspostgres.NewBusinesses(pool)}
	repo := benefitspostgres.NewBenefits(pool)
	coupons := benefitspostgres.NewCoupons(pool)
	limiter := ratelimit.New(ratelimit.Config{MaxAttempts: couponAttempts, Window: couponWindow, MaxKeys: couponMaxKeys}, nil)
	return benefitshttp.New(benefitshttp.Deps{
		Verifier:  verifier,
		Log:       log,
		List:      queries.NewListMyBenefits(owners, repo, coupons, nil),
		Add:       commands.NewAddBenefit(owners, repo),
		Edit:      commands.NewEditBenefit(owners, repo),
		SetActive: commands.NewSetBenefitActive(owners, repo),
		Pause:     commands.NewPauseBenefit(repo),
		Overview:  queries.NewGetOverview(owners, coupons, nil),
		Quota:     queries.NewGetCouponQuota(owners, coupons, nil),
		SetQuota:  commands.NewSetCouponQuota(owners, coupons, nil),
		Generate:  commands.NewGenerateCode(coupons, limiter, nil, nil),
		Redeem:    commands.NewRedeemCode(owners, coupons, nil),
		Preview:   queries.NewPreviewCode(owners, coupons, nil),
		View:      queries.NewViewMyCoupon(coupons, nil),
		Activity:  queries.NewRecentActivity(owners, coupons),
	}).Mount
}
