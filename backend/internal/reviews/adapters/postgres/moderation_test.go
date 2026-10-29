package postgres_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/joaquinBrassinne/MOVO/internal/platform/dbtest"
	"github.com/joaquinBrassinne/MOVO/internal/reviews"
	"github.com/joaquinBrassinne/MOVO/internal/reviews/adapters/postgres"
	"github.com/joaquinBrassinne/MOVO/internal/reviews/domain"
)

// createdReview stores, through the repository, a review with the given stars of a new
// redeemed coupon (n makes it distinct) and returns its id, so the rating of the branch
// is kept the way the application keeps it.
func createdReview(ctx context.Context, t *testing.T, pool *pgxpool.Pool, s seed, n, stars int) int64 {
	t.Helper()

	customer, coupon := addRedeemedCoupon(ctx, t, pool, s, n)
	if err := postgres.NewReviews(pool).Create(ctx, reviewOf(s, customer, coupon, stars)); err != nil {
		t.Fatalf("Create review %d: %v", n, err)
	}
	var id int64
	if err := pool.QueryRow(ctx, `SELECT id FROM reviews WHERE benefit_code_id = $1`, coupon).Scan(&id); err != nil {
		t.Fatalf("read review id: %v", err)
	}
	return id
}

func statusOf(ctx context.Context, t *testing.T, pool *pgxpool.Pool, id int64) string {
	t.Helper()
	var status string
	if err := pool.QueryRow(ctx, `SELECT status FROM reviews WHERE id = $1`, id).Scan(&status); err != nil {
		t.Fatalf("read status: %v", err)
	}
	return status
}

// auditOf lists the actions recorded for a review, oldest first.
func auditOf(ctx context.Context, t *testing.T, pool *pgxpool.Pool, id int64) []string {
	t.Helper()
	rows, err := pool.Query(ctx, `SELECT action FROM audit_log WHERE entity = 'review' AND entity_id = $1 ORDER BY id`, id)
	if err != nil {
		t.Fatalf("read audit: %v", err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var action string
		if err := rows.Scan(&action); err != nil {
			t.Fatalf("scan audit: %v", err)
		}
		out = append(out, action)
	}
	return out
}

func TestReportMarksAVisibleReviewAndRecordsWhoDidIt(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	ctx := t.Context()
	s := newSeed(ctx, t, pool, true)
	id := createdReview(ctx, t, pool, s, 1, 5)
	repo := postgres.NewReviews(pool)

	got, err := repo.Report(ctx, s.owner, s.business, id)
	if err != nil || got.ID != id || got.Status != domain.StatusReported {
		t.Fatalf("Report = %+v, %v; want review %d reported", got, err, id)
	}
	if status := statusOf(ctx, t, pool, id); status != "reported" {
		t.Errorf("stored status = %q, want reported", status)
	}
	if fmt.Sprint(auditOf(ctx, t, pool, id)) != "[review.report]" {
		t.Errorf("audit = %v, want one review.report", auditOf(ctx, t, pool, id))
	}
	var actor int64
	var business float64
	if err := pool.QueryRow(ctx, `SELECT actor_id, (metadata->>'business_id')::float8 FROM audit_log WHERE entity = 'review' AND entity_id = $1`, id).Scan(&actor, &business); err != nil || actor != s.owner || int64(business) != s.business {
		t.Errorf("audit actor %d business %v, %v; want the owner %d and business %d", actor, business, err, s.owner, s.business)
	}
	// A reported review still counts: only the super admin hides one.
	if avg, count := rating(ctx, t, pool, s.branch); avg != "5.00" || count != 1 {
		t.Errorf("rating after the report = %s (%d), want it untouched at 5.00 (1)", avg, count)
	}
}

func TestReportingAgainOrAHiddenReviewChangesAndRecordsNothing(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	ctx := t.Context()
	s := newSeed(ctx, t, pool, true)
	reported := createdReview(ctx, t, pool, s, 1, 5)
	hidden := createdReview(ctx, t, pool, s, 2, 4)
	repo := postgres.NewReviews(pool)

	if _, err := repo.Report(ctx, s.owner, s.business, reported); err != nil {
		t.Fatalf("first report: %v", err)
	}
	if got, err := repo.Report(ctx, s.owner, s.business, reported); err != nil || got.Status != domain.StatusReported {
		t.Errorf("second report = %+v, %v; want it still reported", got, err)
	}
	if n := len(auditOf(ctx, t, pool, reported)); n != 1 {
		t.Errorf("audit entries after reporting twice = %d, want 1", n)
	}

	if _, err := repo.SetStatus(ctx, s.owner, hidden, domain.StatusHidden); err != nil {
		t.Fatalf("hide: %v", err)
	}
	if got, err := repo.Report(ctx, s.owner, s.business, hidden); err != nil || got.Status != domain.StatusHidden {
		t.Errorf("report of a hidden review = %+v, %v; want it left hidden", got, err)
	}
	if fmt.Sprint(auditOf(ctx, t, pool, hidden)) != "[review.hide]" {
		t.Errorf("audit of the hidden review = %v, want only review.hide", auditOf(ctx, t, pool, hidden))
	}
}

// TestReportOfAnotherBusinessReviewLooksLikeOneThatDoesNotExist pins that an owner learns
// nothing about the reviews of a business that is not theirs.
func TestReportOfAnotherBusinessReviewLooksLikeOneThatDoesNotExist(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	ctx := t.Context()
	s := newSeed(ctx, t, pool, true)
	id := createdReview(ctx, t, pool, s, 1, 5)
	repo := postgres.NewReviews(pool)

	if _, err := repo.Report(ctx, s.owner, s.business+1000, id); !errors.Is(err, reviews.ErrReviewNotFound) {
		t.Errorf("review of another business: err = %v, want ErrReviewNotFound", err)
	}
	if _, err := repo.Report(ctx, s.owner, s.business, id+1000); !errors.Is(err, reviews.ErrReviewNotFound) {
		t.Errorf("review that does not exist: err = %v, want ErrReviewNotFound", err)
	}
	if status := statusOf(ctx, t, pool, id); status != "visible" || len(auditOf(ctx, t, pool, id)) != 0 {
		t.Errorf("after refused reports: status %q, audit %v; want visible and none", status, auditOf(ctx, t, pool, id))
	}
}

func TestHidingAReviewRecountsTheRatingAndRestoringBringsItBack(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	ctx := t.Context()
	s := newSeed(ctx, t, pool, true)
	createdReview(ctx, t, pool, s, 1, 5)
	low := createdReview(ctx, t, pool, s, 2, 2)
	repo := postgres.NewReviews(pool)
	if avg, count := rating(ctx, t, pool, s.branch); avg != "3.50" || count != 2 {
		t.Fatalf("before hiding: %s (%d), want 3.50 (2)", avg, count)
	}

	if got, err := repo.SetStatus(ctx, s.owner, low, domain.StatusHidden); err != nil || got.Status != domain.StatusHidden {
		t.Fatalf("hide = %+v, %v", got, err)
	}
	if avg, count := rating(ctx, t, pool, s.branch); avg != "5.00" || count != 1 {
		t.Errorf("after hiding the 2: %s (%d), want 5.00 (1)", avg, count)
	}
	list, err := repo.ListPublic(ctx, s.branch, nil, 10)
	if err != nil || len(list) != 1 {
		t.Errorf("public list after hiding = %v, %v; want one review", ids(list), err)
	}

	if got, err := repo.SetStatus(ctx, s.owner, low, domain.StatusVisible); err != nil || got.Status != domain.StatusVisible {
		t.Fatalf("restore = %+v, %v", got, err)
	}
	if avg, count := rating(ctx, t, pool, s.branch); avg != "3.50" || count != 2 {
		t.Errorf("after restoring: %s (%d), want 3.50 (2)", avg, count)
	}
	if fmt.Sprint(auditOf(ctx, t, pool, low)) != "[review.hide review.restore]" {
		t.Errorf("audit = %v, want review.hide then review.restore", auditOf(ctx, t, pool, low))
	}
}

func TestRestoringAReportedReviewDismissesTheReport(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	ctx := t.Context()
	s := newSeed(ctx, t, pool, true)
	id := createdReview(ctx, t, pool, s, 1, 5)
	repo := postgres.NewReviews(pool)
	if _, err := repo.Report(ctx, s.owner, s.business, id); err != nil {
		t.Fatalf("report: %v", err)
	}

	if got, err := repo.SetStatus(ctx, s.owner, id, domain.StatusVisible); err != nil || got.Status != domain.StatusVisible {
		t.Fatalf("dismiss = %+v, %v", got, err)
	}
	if status := statusOf(ctx, t, pool, id); status != "visible" {
		t.Errorf("status = %q, want visible", status)
	}
	if fmt.Sprint(auditOf(ctx, t, pool, id)) != "[review.report review.restore]" {
		t.Errorf("audit = %v, want review.report then review.restore", auditOf(ctx, t, pool, id))
	}
}

func TestSettingTheCurrentStatusAgainChangesAndRecordsNothing(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	ctx := t.Context()
	s := newSeed(ctx, t, pool, true)
	id := createdReview(ctx, t, pool, s, 1, 5)
	repo := postgres.NewReviews(pool)

	if _, err := repo.SetStatus(ctx, s.owner, id, domain.StatusVisible); err != nil {
		t.Fatalf("visible on a visible review: %v", err)
	}
	for range 2 {
		if _, err := repo.SetStatus(ctx, s.owner, id, domain.StatusHidden); err != nil {
			t.Fatalf("hide: %v", err)
		}
	}
	if fmt.Sprint(auditOf(ctx, t, pool, id)) != "[review.hide]" {
		t.Errorf("audit = %v, want a single review.hide", auditOf(ctx, t, pool, id))
	}
	if _, err := repo.SetStatus(ctx, s.owner, id+1000, domain.StatusHidden); !errors.Is(err, reviews.ErrReviewNotFound) {
		t.Errorf("unknown review: err = %v, want ErrReviewNotFound", err)
	}
}

// TestHidingTwoReviewsOfOneBranchTogetherLeavesTheRightCount pins that the branch row is
// locked: otherwise each transaction recounts without the other's change and one hidden
// review would still count.
func TestHidingManyReviewsOfOneBranchTogetherLeavesTheRightCount(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	ctx := t.Context()
	s := newSeed(ctx, t, pool, true)
	// Eight, not two: with so few the transactions rarely overlap and the test would
	// pass without the lock.
	const racers = 8
	var all []int64
	for n := 1; n <= racers; n++ {
		all = append(all, createdReview(ctx, t, pool, s, n, 1+n%5))
	}
	repo := postgres.NewReviews(pool)

	start := make(chan struct{})
	errs := make(chan error, racers)
	var wg sync.WaitGroup
	for _, id := range all {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, err := repo.SetStatus(ctx, s.owner, id, domain.StatusHidden)
			errs <- err
		}()
	}
	close(start)
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Errorf("SetStatus: %v", err)
		}
	}

	if avg, count := rating(ctx, t, pool, s.branch); avg != "0.00" || count != 0 {
		t.Errorf("rating = %s (%d), want 0.00 (0): every review hidden", avg, count)
	}
}
