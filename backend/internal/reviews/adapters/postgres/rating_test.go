package postgres_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/joaquinBrassinne/MOVO/internal/platform/dbtest"
	"github.com/joaquinBrassinne/MOVO/internal/reviews"
	"github.com/joaquinBrassinne/MOVO/internal/reviews/adapters/postgres"
	"github.com/joaquinBrassinne/MOVO/internal/reviews/domain"
)

// rating reads the average and the count the cards of the branch show.
func rating(ctx context.Context, t *testing.T, pool *pgxpool.Pool, branch int64) (avg string, count int) {
	t.Helper()
	if err := pool.QueryRow(ctx, `SELECT rating_avg::text, rating_count FROM branches WHERE id = $1`, branch).Scan(&avg, &count); err != nil {
		t.Fatalf("read the rating: %v", err)
	}
	return avg, count
}

// reviewOf is a review of the coupon of a customer, with the given rating.
func reviewOf(s seed, customer, coupon int64, stars int) domain.Review {
	r := review(s)
	r.CustomerID, r.CouponID, r.Rating = customer, coupon, stars
	return r
}

func TestCreateKeepsTheRatingOfTheBranchInStep(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	ctx := t.Context()
	s := newSeed(ctx, t, pool, true)
	repo := postgres.NewReviews(pool)

	if avg, count := rating(ctx, t, pool, s.branch); avg != "0.00" || count != 0 {
		t.Fatalf("before any review: %s (%d), want 0.00 (0)", avg, count)
	}
	if err := repo.Create(ctx, review(s)); err != nil {
		t.Fatalf("first review: %v", err)
	}
	if avg, count := rating(ctx, t, pool, s.branch); avg != "5.00" || count != 1 {
		t.Errorf("after a 5: %s (%d), want 5.00 (1)", avg, count)
	}

	customer, coupon := addRedeemedCoupon(ctx, t, pool, s, 1)
	if err := repo.Create(ctx, reviewOf(s, customer, coupon, 2)); err != nil {
		t.Fatalf("second review: %v", err)
	}
	if avg, count := rating(ctx, t, pool, s.branch); avg != "3.50" || count != 2 {
		t.Errorf("after a 5 and a 2: %s (%d), want 3.50 (2)", avg, count)
	}
}

// TestRatingCountsEveryReviewButTheHiddenOnes pins SDD 5.5: a reported review still
// counts, and only the one the super admin hid does not.
func TestRatingCountsEveryReviewButTheHiddenOnes(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	ctx := t.Context()
	s := newSeed(ctx, t, pool, true)
	addReview(ctx, t, pool, s, 1, "reported", created) // a 4
	addReview(ctx, t, pool, s, 2, "hidden", created)   // a 4 that must not count

	if err := postgres.NewReviews(pool).Create(ctx, review(s)); err != nil { // a 5
		t.Fatalf("Create: %v", err)
	}
	if avg, count := rating(ctx, t, pool, s.branch); avg != "4.50" || count != 2 {
		t.Errorf("rating = %s (%d), want 4.50 (2): the reported counts and the hidden does not", avg, count)
	}
}

func TestRefusedReviewDoesNotMoveTheRating(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	ctx := t.Context()
	s := newSeed(ctx, t, pool, true)
	repo := postgres.NewReviews(pool)
	if err := repo.Create(ctx, review(s)); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := repo.Create(ctx, reviewOf(s, s.customer, s.coupon, 1)); !errors.Is(err, reviews.ErrAlreadyReviewed) {
		t.Fatalf("second review of the coupon: err = %v, want ErrAlreadyReviewed", err)
	}
	if avg, count := rating(ctx, t, pool, s.branch); avg != "5.00" || count != 1 {
		t.Errorf("rating = %s (%d), want it untouched at 5.00 (1)", avg, count)
	}
}

// TestRatingSurvivesReviewsOfOneBranchArrivingTogether pins that no review is lost from
// the count when several coupons of a branch are reviewed at the same moment: each
// transaction would otherwise see only its own review.
func TestRatingSurvivesReviewsOfOneBranchArrivingTogether(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	ctx := t.Context()
	s := newSeed(ctx, t, pool, true)
	repo := postgres.NewReviews(pool)

	const racers = 8
	type coupon struct{ customer, id int64 }
	coupons := make([]coupon, racers)
	for i := range coupons {
		coupons[i].customer, coupons[i].id = addRedeemedCoupon(ctx, t, pool, s, i+1)
	}

	start := make(chan struct{})
	errs := make(chan error, racers)
	var wg sync.WaitGroup
	for i, c := range coupons {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			stars := 1 + i%5 // 1, 2, 3, 4, 5, 1, 2, 3: sums to 21
			errs <- repo.Create(ctx, reviewOf(s, c.customer, c.id, stars))
		}()
	}
	close(start)
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Errorf("Create: %v", err)
		}
	}

	// 21 / 8 = 2.625, which rounds to 2.63.
	if avg, count := rating(ctx, t, pool, s.branch); avg != "2.63" || count != racers {
		t.Errorf("rating = %s (%d), want 2.63 (%d): every review counted", avg, count, racers)
	}
}
