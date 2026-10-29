package postgres_test

import (
	"errors"
	"testing"

	"github.com/joaquinBrassinne/MOVO/internal/platform/dbtest"
	"github.com/joaquinBrassinne/MOVO/internal/reviews"
	"github.com/joaquinBrassinne/MOVO/internal/reviews/adapters/postgres"
	"github.com/joaquinBrassinne/MOVO/internal/reviews/domain"
)

func TestFormByTokenHashShowsWhereTheCouponWasRedeemedAndIfItWasReviewed(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	ctx := t.Context()
	s := newSeed(ctx, t, pool, true)
	repo := postgres.NewReviews(pool)

	got, err := repo.FormByTokenHash(ctx, domain.HashAccessToken(token))
	if err != nil {
		t.Fatalf("FormByTokenHash: %v", err)
	}
	if got.BusinessName != "Studio" || got.BranchName != "Centro" || got.City != "Cordoba" || got.Code != "ABCDEFGH" ||
		got.CustomerName != "Ana García" || got.RedeemedAt.IsZero() || got.Reviewed {
		t.Errorf("form = %+v, want the redeeming branch, the code, the customer and not reviewed", got)
	}

	if err := repo.Create(ctx, reviewOf(s, s.customer, s.coupon, 5)); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if got, err := repo.FormByTokenHash(ctx, domain.HashAccessToken(token)); err != nil || !got.Reviewed {
		t.Errorf("after reviewing = %+v, %v; want it reviewed", got, err)
	}
}

func TestFormByTokenHashOfACouponNotRedeemedHasNoBranchNorTime(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	ctx := t.Context()
	_ = newSeed(ctx, t, pool, false)

	got, err := postgres.NewReviews(pool).FormByTokenHash(ctx, domain.HashAccessToken(token))
	if err != nil || got.BusinessName != "Studio" || got.BranchName != "" || !got.RedeemedAt.IsZero() {
		t.Errorf("form = %+v, %v; want the business and neither branch nor time", got, err)
	}
}

func TestFormByTokenHashOfAnUnknownTokenIsNotFound(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	ctx := t.Context()
	_ = newSeed(ctx, t, pool, true)

	if _, err := postgres.NewReviews(pool).FormByTokenHash(ctx, domain.HashAccessToken("nope")); !errors.Is(err, reviews.ErrCouponNotFound) {
		t.Errorf("err = %v, want ErrCouponNotFound", err)
	}
}
