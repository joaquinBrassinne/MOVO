package postgres_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/joaquinBrassinne/MOVO/internal/platform/dbtest"
	"github.com/joaquinBrassinne/MOVO/internal/reviews"
	"github.com/joaquinBrassinne/MOVO/internal/reviews/adapters/postgres"
	"github.com/joaquinBrassinne/MOVO/internal/reviews/domain"
)

var created = time.Date(2028, 3, 2, 10, 0, 0, 0, time.UTC)

// addReview inserts, for a new customer, a coupon redeemed at the branch and its review
// with the given status and creation time, returning the review id. n makes the code and
// the phone distinct.
func addReview(ctx context.Context, t *testing.T, pool *pgxpool.Pool, s seed, n int, status string, at time.Time) int64 {
	t.Helper()

	customer, coupon := addRedeemedCoupon(ctx, t, pool, s, n)
	var review int64
	if err := pool.QueryRow(ctx, `
INSERT INTO reviews (branch_id, business_id, benefit_code_id, customer_id, rating, comment, display_name, status, hidden_at, created_at)
VALUES ($1, $2, $3, $4, 4, $5, $6, $7::text, CASE WHEN $7::text = 'hidden' THEN $8::timestamptz END, $8::timestamptz) RETURNING id`,
		s.branch, s.business, coupon, customer, fmt.Sprintf("Comment %d", n), fmt.Sprintf("Cliente %d", n), status, at).Scan(&review); err != nil {
		t.Fatalf("insert review: %v", err)
	}
	return review
}

// addRedeemedCoupon inserts a new customer and a coupon of theirs redeemed at the branch,
// without a review, returning both ids. n makes the code and the phone distinct.
func addRedeemedCoupon(ctx context.Context, t *testing.T, pool *pgxpool.Pool, s seed, n int) (customer, coupon int64) {
	t.Helper()

	const letters = "ABCDEFGHJKLMNPQRSTUVWXYZ" // the alphabet of the coupon codes
	code := "AAAAAA" + string(letters[n/len(letters)]) + string(letters[n%len(letters)])
	if err := pool.QueryRow(ctx, `INSERT INTO customers (name, phone) VALUES ($1, $2) RETURNING id`,
		fmt.Sprintf("Cliente %d", n), fmt.Sprintf("54911%08d", n)).Scan(&customer); err != nil {
		t.Fatalf("insert customer: %v", err)
	}
	if err := pool.QueryRow(ctx, `
INSERT INTO benefit_codes (code, benefit_id, business_id, customer_id, access_token_hash, status, expires_at, redeemed_at, redeemed_by, redeemed_branch_id)
VALUES ($1, $2, $3, $4, sha256(convert_to($1::text, 'UTF8')), 'redeemed', now() + interval '48 hours', now(), $5, $6) RETURNING id`,
		code, s.benefit, s.business, customer, s.owner, s.branch).Scan(&coupon); err != nil {
		t.Fatalf("insert coupon: %v", err)
	}
	return customer, coupon
}

func ids(list []reviews.ListedReview) []int64 {
	out := make([]int64, len(list))
	for i, r := range list {
		out[i] = r.ID
	}
	return out
}

func TestListPublicReturnsTheBranchReviewsNewestFirst(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	ctx := t.Context()
	s := newSeed(ctx, t, pool, true)
	oldest := addReview(ctx, t, pool, s, 1, "visible", created)
	newest := addReview(ctx, t, pool, s, 2, "visible", created.Add(2*time.Hour))
	middle := addReview(ctx, t, pool, s, 3, "visible", created.Add(time.Hour))

	got, err := postgres.NewReviews(pool).ListPublic(ctx, s.branch, domain.StarFilter{}, nil, 10)
	if err != nil {
		t.Fatalf("ListPublic: %v", err)
	}
	if want := []int64{newest, middle, oldest}; fmt.Sprint(ids(got)) != fmt.Sprint(want) {
		t.Fatalf("order = %v, want %v", ids(got), want)
	}
	if first := got[0]; first.Rating != 4 || first.Comment != "Comment 2" || first.DisplayName != "Cliente 2" || !first.CreatedAt.Equal(created.Add(2*time.Hour)) {
		t.Errorf("review = %+v, want the stored rating, comment, name and time", first)
	}
}

// TestListPublicKeepsAReportedReviewAndDropsAHiddenOne pins SDD 5.5: the owner reports
// and the review stays public; only the super admin hides it.
func TestListPublicKeepsAReportedReviewAndDropsAHiddenOne(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	ctx := t.Context()
	s := newSeed(ctx, t, pool, true)
	visible := addReview(ctx, t, pool, s, 1, "visible", created)
	reported := addReview(ctx, t, pool, s, 2, "reported", created.Add(time.Hour))
	addReview(ctx, t, pool, s, 3, "hidden", created.Add(2*time.Hour))

	got, err := postgres.NewReviews(pool).ListPublic(ctx, s.branch, domain.StarFilter{}, nil, 10)
	if err != nil {
		t.Fatalf("ListPublic: %v", err)
	}
	if want := []int64{reported, visible}; fmt.Sprint(ids(got)) != fmt.Sprint(want) {
		t.Errorf("listed = %v, want the reported and the visible one, not the hidden %v", ids(got), want)
	}
}

func TestListPublicNeverMixesInTheReviewsOfAnotherBranch(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	ctx := t.Context()
	s := newSeed(ctx, t, pool, true)
	mine := addReview(ctx, t, pool, s, 1, "visible", created)

	var other int64
	if err := pool.QueryRow(ctx, `
INSERT INTO branches (business_id, name, address, city, location)
VALUES ($1, 'Norte', 'Calle 2', 'Cordoba', ST_SetSRID(ST_MakePoint(-64.19, -31.40), 4326)::geography) RETURNING id`, s.business).Scan(&other); err != nil {
		t.Fatalf("insert other branch: %v", err)
	}
	other2 := s
	other2.branch = other
	theirs := addReview(ctx, t, pool, other2, 2, "visible", created)

	repo := postgres.NewReviews(pool)
	for branch, want := range map[int64]int64{s.branch: mine, other: theirs} {
		got, err := repo.ListPublic(ctx, branch, domain.StarFilter{}, nil, 10)
		if err != nil || len(got) != 1 || got[0].ID != want {
			t.Errorf("branch %d lists %v, %v; want only %d", branch, ids(got), err, want)
		}
	}
}

// TestListPublicPagesWithoutRepeatingOrSkippingAnyReview pins the keyset order, with
// reviews created at the very same instant, where only the id breaks the tie.
func TestListPublicPagesWithoutRepeatingOrSkippingAnyReview(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	ctx := t.Context()
	s := newSeed(ctx, t, pool, true)
	var all []int64
	for n := 1; n <= 7; n++ {
		// Three of them share an instant.
		at := created.Add(time.Duration(n/3) * time.Hour)
		all = append(all, addReview(ctx, t, pool, s, n, "visible", at))
	}
	repo := postgres.NewReviews(pool)

	var seen []int64
	var after *domain.Cursor
	for pages := 0; ; pages++ {
		if pages > 7 {
			t.Fatal("the pages never end")
		}
		page, err := repo.ListPublic(ctx, s.branch, domain.StarFilter{}, after, 3)
		if err != nil {
			t.Fatalf("ListPublic: %v", err)
		}
		seen = append(seen, ids(page)...)
		if len(page) < 3 {
			break
		}
		last := page[len(page)-1]
		after = &domain.Cursor{CreatedAt: last.CreatedAt, ID: last.ID}
	}

	full, err := repo.ListPublic(ctx, s.branch, domain.StarFilter{}, nil, 100)
	if err != nil {
		t.Fatalf("ListPublic all: %v", err)
	}
	if len(seen) != len(all) || fmt.Sprint(seen) != fmt.Sprint(ids(full)) {
		t.Errorf("paged = %v, one go = %v; want the same %d reviews in the same order", seen, ids(full), len(all))
	}
}

func TestListPublicTreatsABranchTheCityCannotSeeAsNotFound(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	ctx := t.Context()
	s := newSeed(ctx, t, pool, true)
	addReview(ctx, t, pool, s, 1, "visible", created)
	repo := postgres.NewReviews(pool)

	if _, err := repo.ListPublic(ctx, s.branch+1000, domain.StarFilter{}, nil, 10); !errors.Is(err, reviews.ErrBranchNotFound) {
		t.Errorf("unknown branch: err = %v, want ErrBranchNotFound", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE branches SET active = false WHERE id = $1`, s.branch); err != nil {
		t.Fatalf("retire branch: %v", err)
	}
	if _, err := repo.ListPublic(ctx, s.branch, domain.StarFilter{}, nil, 10); !errors.Is(err, reviews.ErrBranchNotFound) {
		t.Errorf("retired branch: err = %v, want ErrBranchNotFound, as if it did not exist", err)
	}
}

func TestListPublicOfABranchWithoutReviewsIsEmptyNotAnError(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	ctx := t.Context()
	s := newSeed(ctx, t, pool, true)

	got, err := postgres.NewReviews(pool).ListPublic(ctx, s.branch, domain.StarFilter{}, nil, 10)
	if err != nil || len(got) != 0 {
		t.Errorf("ListPublic = %v, %v; want no reviews and no error", got, err)
	}
}
