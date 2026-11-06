package postgres_test

import (
	"errors"
	"testing"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/platform/dbtest"
	"github.com/joaquinBrassinne/MOVO/internal/reviews"
	"github.com/joaquinBrassinne/MOVO/internal/reviews/adapters/postgres"
	"github.com/joaquinBrassinne/MOVO/internal/reviews/domain"
)

type storedReview struct {
	comment     *string
	displayName string
	showName    bool
	purged      bool
}

func TestPurgeHiddenErasesTheTextOfOldHiddenReviewsOnly(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	ctx := t.Context()
	s := newSeed(ctx, t, pool, true)
	repo := postgres.NewReviews(pool)

	old := createdReview(ctx, t, pool, s, 1, 5)
	recent := createdReview(ctx, t, pool, s, 2, 1)
	visible := createdReview(ctx, t, pool, s, 3, 4)
	for _, id := range []int64{old, recent, visible} {
		if _, err := pool.Exec(ctx, `UPDATE reviews SET comment = 'texto ofensivo', display_name = 'Ana G.', show_name = true WHERE id = $1`, id); err != nil {
			t.Fatalf("set text: %v", err)
		}
	}
	for _, id := range []int64{old, recent} {
		if _, err := repo.SetStatus(ctx, s.owner, id, domain.StatusHidden); err != nil {
			t.Fatalf("hide %d: %v", id, err)
		}
	}
	// "old" was hidden four days ago, "recent" just now.
	if _, err := pool.Exec(ctx, `UPDATE reviews SET hidden_at = now() - interval '96 hours' WHERE id = $1`, old); err != nil {
		t.Fatalf("age review: %v", err)
	}

	n, err := repo.PurgeHidden(ctx, time.Now().Add(-domain.HiddenRetention))
	if err != nil || n != 1 {
		t.Fatalf("PurgeHidden = %d, %v; want exactly the old one", n, err)
	}

	read := func(id int64) storedReview {
		var r storedReview
		if err := pool.QueryRow(ctx, `SELECT comment, display_name, show_name, purged_at IS NOT NULL FROM reviews WHERE id = $1`, id).
			Scan(&r.comment, &r.displayName, &r.showName, &r.purged); err != nil {
			t.Fatalf("read review: %v", err)
		}
		return r
	}
	if got := read(old); got.comment != nil || got.displayName != domain.AnonymousName || got.showName || !got.purged {
		t.Errorf("purged review = %+v, want no comment, anonymous and purged", got)
	}
	for name, id := range map[string]int64{"recent hidden": recent, "visible": visible} {
		if got := read(id); got.comment == nil || *got.comment != "texto ofensivo" || got.displayName != "Ana G." || got.purged {
			t.Errorf("%s review = %+v, want it untouched", name, got)
		}
	}

	var rows int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM reviews WHERE id = $1 AND status = 'hidden'`, old).Scan(&rows); err != nil || rows != 1 {
		t.Errorf("the purged review has to stay hidden in its row (%d, %v): its coupon stays reviewed", rows, err)
	}

	again, err := repo.PurgeHidden(ctx, time.Now().Add(-domain.HiddenRetention))
	if err != nil || again != 0 {
		t.Errorf("second PurgeHidden = %d, %v; want nothing left to erase", again, err)
	}
}

func TestAPurgedReviewCannotBeRestoredNorItsCouponReviewedAgain(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	ctx := t.Context()
	s := newSeed(ctx, t, pool, true)
	repo := postgres.NewReviews(pool)

	id := createdReview(ctx, t, pool, s, 1, 5)
	if _, err := repo.SetStatus(ctx, s.owner, id, domain.StatusHidden); err != nil {
		t.Fatalf("hide: %v", err)
	}
	if _, err := repo.PurgeHidden(ctx, time.Now().Add(time.Minute)); err != nil {
		t.Fatalf("purge: %v", err)
	}

	if _, err := repo.SetStatus(ctx, s.owner, id, domain.StatusVisible); !errors.Is(err, reviews.ErrReviewPurged) {
		t.Errorf("restore a purged review = %v, want ErrReviewPurged", err)
	}
	if status := statusOf(ctx, t, pool, id); status != "hidden" {
		t.Errorf("status = %q, want it still hidden", status)
	}
	if got, err := repo.SetStatus(ctx, s.owner, id, domain.StatusHidden); err != nil || got.Status != domain.StatusHidden {
		t.Errorf("hiding a purged review again = %+v, %v; want a no-op", got, err)
	}

	var coupon int64
	if err := pool.QueryRow(ctx, `SELECT benefit_code_id FROM reviews WHERE id = $1`, id).Scan(&coupon); err != nil {
		t.Fatalf("read coupon: %v", err)
	}
	again := reviewOf(s, s.customer, coupon, 1)
	if err := repo.Create(ctx, again); !errors.Is(err, reviews.ErrAlreadyReviewed) {
		t.Errorf("review the same coupon after the purge = %v, want ErrAlreadyReviewed", err)
	}
}

func TestHidingStampsHiddenAtAndRestoringClearsIt(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	ctx := t.Context()
	s := newSeed(ctx, t, pool, true)
	repo := postgres.NewReviews(pool)
	id := createdReview(ctx, t, pool, s, 1, 3)

	hiddenAt := func() (set bool) {
		if err := pool.QueryRow(ctx, `SELECT hidden_at IS NOT NULL FROM reviews WHERE id = $1`, id).Scan(&set); err != nil {
			t.Fatalf("read hidden_at: %v", err)
		}
		return set
	}
	if hiddenAt() {
		t.Fatal("a new review has no hidden_at")
	}
	if _, err := repo.SetStatus(ctx, s.owner, id, domain.StatusHidden); err != nil || !hiddenAt() {
		t.Fatalf("hide: %v, hidden_at set = %v; want it set", err, hiddenAt())
	}
	if _, err := repo.SetStatus(ctx, s.owner, id, domain.StatusVisible); err != nil || hiddenAt() {
		t.Fatalf("restore: %v, hidden_at set = %v; want it cleared", err, hiddenAt())
	}
}
