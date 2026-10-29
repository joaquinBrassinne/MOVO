package postgres_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/joaquinBrassinne/MOVO/internal/platform/dbtest"
	"github.com/joaquinBrassinne/MOVO/internal/reviews"
	"github.com/joaquinBrassinne/MOVO/internal/reviews/adapters/postgres"
	"github.com/joaquinBrassinne/MOVO/internal/reviews/domain"
)

const token = "a-customer-access-token"

// seed is a business with a branch, a customer and a coupon opened by token.
type seed struct {
	owner, business, branch, customer, coupon, benefit int64
}

// newSeed inserts the rows a review hangs from. The coupon is redeemed at the branch when
// redeemed is true, and stays pending otherwise.
func newSeed(ctx context.Context, t *testing.T, pool *pgxpool.Pool, redeemed bool) seed {
	t.Helper()

	var s seed
	if err := pool.QueryRow(ctx, `INSERT INTO users (email, password_hash, role) VALUES ('owner@example.com', 'hash', 'business_admin') RETURNING id`).Scan(&s.owner); err != nil {
		t.Fatalf("insert owner: %v", err)
	}
	if err := pool.QueryRow(ctx, `
INSERT INTO businesses (owner_user_id, category_id, name, slug, whatsapp_number, status)
VALUES ($1, (SELECT id FROM categories WHERE slug = 'yoga'), 'Studio', 'studio', '5491112345678', 'active') RETURNING id`, s.owner).Scan(&s.business); err != nil {
		t.Fatalf("insert business: %v", err)
	}
	if err := pool.QueryRow(ctx, `
INSERT INTO branches (business_id, name, address, city, location)
VALUES ($1, 'Centro', 'Calle 1', 'Cordoba', ST_SetSRID(ST_MakePoint(-64.18, -31.42), 4326)::geography) RETURNING id`, s.business).Scan(&s.branch); err != nil {
		t.Fatalf("insert branch: %v", err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO customers (name, phone) VALUES ('Ana García', '5491187654321') RETURNING id`).Scan(&s.customer); err != nil {
		t.Fatalf("insert customer: %v", err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO benefits (business_id, type, percent) VALUES ($1, 'percent', 10) RETURNING id`, s.business).Scan(&s.benefit); err != nil {
		t.Fatalf("insert benefit: %v", err)
	}
	// The hash is computed here with the database's own SHA-256, as the benefits module
	// stores it, so the lookup by domain.HashAccessToken proves both agree.
	if err := pool.QueryRow(ctx, `
INSERT INTO benefit_codes (code, benefit_id, business_id, customer_id, access_token_hash, expires_at)
VALUES ('ABCDEFGH', $1, $2, $3, sha256(convert_to($4::text, 'UTF8')), now() + interval '48 hours') RETURNING id`,
		s.benefit, s.business, s.customer, token).Scan(&s.coupon); err != nil {
		t.Fatalf("insert coupon: %v", err)
	}
	if redeemed {
		if _, err := pool.Exec(ctx,
			`UPDATE benefit_codes SET status = 'redeemed', redeemed_at = now(), redeemed_by = $1, redeemed_branch_id = $2 WHERE id = $3`,
			s.owner, s.branch, s.coupon); err != nil {
			t.Fatalf("redeem coupon: %v", err)
		}
	}
	return s
}

func review(s seed) domain.Review {
	return domain.Review{
		CouponID: s.coupon, BranchID: s.branch, BusinessID: s.business, CustomerID: s.customer,
		Rating: 5, Comment: "Genial", DisplayName: "Ana G.", ShowName: true,
		Status: domain.StatusVisible, CreatedAt: time.Date(2028, 3, 2, 10, 0, 0, 0, time.UTC),
	}
}

func TestByTokenHashReadsWhereAndByWhomTheCouponWasRedeemed(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	ctx := t.Context()
	s := newSeed(ctx, t, pool, true)

	got, err := postgres.NewReviews(pool).ByTokenHash(ctx, domain.HashAccessToken(token))
	if err != nil {
		t.Fatalf("ByTokenHash: %v", err)
	}
	if got.CouponID != s.coupon || got.BranchID != s.branch || got.BusinessID != s.business || got.CustomerID != s.customer ||
		got.CustomerName != "Ana García" || got.RedeemedAt.IsZero() {
		t.Errorf("redemption = %+v, want the coupon %d redeemed at branch %d", got, s.coupon, s.branch)
	}
}

func TestByTokenHashTellsAPendingCouponFromAnUnknownToken(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	ctx := t.Context()
	newSeed(ctx, t, pool, false)
	repo := postgres.NewReviews(pool)

	pending, err := repo.ByTokenHash(ctx, domain.HashAccessToken(token))
	if err != nil || !pending.RedeemedAt.IsZero() {
		t.Errorf("pending coupon = %+v, %v; want a zero RedeemedAt and no error", pending, err)
	}
	if _, err := repo.ByTokenHash(ctx, domain.HashAccessToken("nobody")); !errors.Is(err, reviews.ErrCouponNotFound) {
		t.Errorf("unknown token: err = %v, want ErrCouponNotFound", err)
	}
}

func TestCreateStoresTheReviewAndRefusesASecondOne(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	ctx := t.Context()
	s := newSeed(ctx, t, pool, true)
	repo := postgres.NewReviews(pool)

	if err := repo.Create(ctx, review(s)); err != nil {
		t.Fatalf("Create: %v", err)
	}
	var rating int
	var comment *string
	var name, status string
	if err := pool.QueryRow(ctx, `SELECT rating, comment, display_name, status FROM reviews WHERE benefit_code_id = $1`, s.coupon).Scan(&rating, &comment, &name, &status); err != nil {
		t.Fatalf("read back: %v", err)
	}
	if rating != 5 || comment == nil || *comment != "Genial" || name != "Ana G." || status != "visible" {
		t.Errorf("stored = %d %v %q %q, want 5 Genial Ana G. visible", rating, comment, name, status)
	}

	if err := repo.Create(ctx, review(s)); !errors.Is(err, reviews.ErrAlreadyReviewed) {
		t.Errorf("second review: err = %v, want ErrAlreadyReviewed", err)
	}
}

func TestCreateStoresNoCommentAsNull(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	ctx := t.Context()
	s := newSeed(ctx, t, pool, true)
	r := review(s)
	r.Comment = ""
	if err := postgres.NewReviews(pool).Create(ctx, r); err != nil {
		t.Fatalf("Create: %v", err)
	}
	var comment *string
	if err := pool.QueryRow(ctx, `SELECT comment FROM reviews WHERE benefit_code_id = $1`, s.coupon).Scan(&comment); err != nil || comment != nil {
		t.Errorf("comment = %v, %v; want NULL", comment, err)
	}
}

// TestRacingReviewsOfOneCouponStoreExactlyOne pins that the unique key, and not a check
// before the insert, settles two requests for the same coupon.
func TestRacingReviewsOfOneCouponStoreExactlyOne(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	ctx := t.Context()
	s := newSeed(ctx, t, pool, true)
	repo := postgres.NewReviews(pool)

	const racers = 8
	results := make(chan error, racers)
	var wg sync.WaitGroup
	for range racers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results <- repo.Create(ctx, review(s))
		}()
	}
	wg.Wait()
	close(results)

	var created, refused int
	for err := range results {
		switch {
		case err == nil:
			created++
		case errors.Is(err, reviews.ErrAlreadyReviewed):
			refused++
		default:
			t.Errorf("unexpected error: %v", err)
		}
	}
	if created != 1 || refused != racers-1 {
		t.Errorf("created %d, refused %d; want 1 and %d", created, refused, racers-1)
	}
}
