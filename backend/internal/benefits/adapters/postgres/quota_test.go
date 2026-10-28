package postgres_test

import (
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/joaquinBrassinne/MOVO/internal/benefits"
	"github.com/joaquinBrassinne/MOVO/internal/benefits/adapters/postgres"
	"github.com/joaquinBrassinne/MOVO/internal/benefits/domain"
	"github.com/joaquinBrassinne/MOVO/internal/platform/dbtest"
)

func intp(n int) *int { return &n }

func TestTwoCustomersRacingForTheLastCouponOfTheBusinessGetExactlyOne(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	ctx := t.Context()
	issuer := postgres.NewCoupons(pool)

	// Several rounds, each with its own business and quota of 1, so a race that is lost
	// by luck once is still caught by another round.
	for round := range 10 {
		s := newShop(ctx, t, pool, fmt.Sprintf("studio%d", round), "active", 1)

		var (
			wg   sync.WaitGroup
			errs = make([]error, 2)
		)
		for i, phone := range []string{"5491100000001", "5491100000002"} {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, errs[i] = issuer.Issue(ctx, request(s, phone, raceCode(round*2+i)))
			}()
		}
		wg.Wait()

		issued, refused := 0, 0
		for _, err := range errs {
			switch {
			case err == nil:
				issued++
			case errors.Is(err, benefits.ErrQuotaReached):
				refused++
			default:
				t.Fatalf("round %d: unexpected error %v", round, err)
			}
		}
		if issued != 1 || refused != 1 {
			t.Fatalf("round %d: issued %d, refused %d; want exactly one of each", round, issued, refused)
		}
	}
}

func TestTheQuotaIsSharedByEveryBenefitOfTheBusinessAndNullMeansNoLimit(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	ctx := t.Context()
	issuer := postgres.NewCoupons(pool)
	s := newShop(ctx, t, pool, "studio", "active", 2)

	var second int64
	if err := pool.QueryRow(ctx, `INSERT INTO benefits (business_id, type, amount_ars) VALUES ($1, 'amount', 500) RETURNING id`, s.business).Scan(&second); err != nil {
		t.Fatalf("insert second benefit: %v", err)
	}
	other := s
	other.benefit = second

	if _, err := issuer.Issue(ctx, request(s, "5491100000001", "AAAAAAAA")); err != nil {
		t.Fatalf("first: %v", err)
	}
	if _, err := issuer.Issue(ctx, request(other, "5491100000002", "BBBBBBBB")); err != nil {
		t.Fatalf("second, from another benefit: %v", err)
	}
	if _, err := issuer.Issue(ctx, request(s, "5491100000003", "CCCCCCCC")); !errors.Is(err, benefits.ErrQuotaReached) {
		t.Errorf("third = %v, want the shared quota reached", err)
	}

	// Removing the limit lets coupons through again.
	start, end := domain.QuotaMonth(now)
	if _, err := issuer.SetQuotaLimit(ctx, s.business, nil, start, end); err != nil {
		t.Fatalf("remove limit: %v", err)
	}
	if _, err := issuer.Issue(ctx, request(s, "5491100000003", "CCCCCCCC")); err != nil {
		t.Errorf("without a limit: %v", err)
	}
}

func TestSetQuotaLimitRefusesALimitBelowWhatWasGenerated(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	ctx := t.Context()
	store := postgres.NewCoupons(pool)
	s := newShop(ctx, t, pool, "studio", "active", 0)
	start, end := domain.QuotaMonth(now)

	for i, phone := range []string{"5491100000001", "5491100000002", "5491100000003"} {
		if _, err := store.Issue(ctx, request(s, phone, raceCode(i))); err != nil {
			t.Fatalf("issue %d: %v", i, err)
		}
	}
	got, err := store.Quota(ctx, s.business, start, end)
	if err != nil || got.Limit != nil || got.Generated != 3 {
		t.Fatalf("Quota = %+v, %v; want no limit and 3 generated", got, err)
	}

	_, err = store.SetQuotaLimit(ctx, s.business, intp(2), start, end)
	var below *domain.QuotaBelowGeneratedError
	if !errors.As(err, &below) || below.Generated != 3 {
		t.Fatalf("limit 2 with 3 generated: %v, want a QuotaBelowGeneratedError carrying 3", err)
	}
	if after, _ := store.Quota(ctx, s.business, start, end); after.Limit != nil {
		t.Errorf("a refused limit must not be stored, got %v", *after.Limit)
	}
	for _, bad := range []int{0, -4} {
		if _, err := store.SetQuotaLimit(ctx, s.business, intp(bad), start, end); !errors.Is(err, domain.ErrQuotaLimitInvalid) {
			t.Errorf("limit %d: %v, want ErrQuotaLimitInvalid", bad, err)
		}
	}
	if ok, err := store.SetQuotaLimit(ctx, s.business, intp(3), start, end); err != nil || ok.Limit == nil || *ok.Limit != 3 || ok.Generated != 3 {
		t.Errorf("limit equal to the generated = %+v, %v; want it allowed", ok, err)
	}
	if _, err := store.Quota(ctx, 999999, start, end); !errors.Is(err, benefits.ErrBusinessNotFound) {
		t.Errorf("unknown business: %v, want ErrBusinessNotFound", err)
	}
}

func TestSettingTheQuotaWaitsForACouponBeingGeneratedAndCannotGoBelowIt(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	ctx := t.Context()
	store := postgres.NewCoupons(pool)
	s := newShop(ctx, t, pool, "studio", "active", 0)
	start, end := domain.QuotaMonth(now)

	// Many customers generate while the owner sets a limit of 5: whatever the order, the
	// limit never ends below what was generated, and what was generated never passes it.
	const customers = 12
	var wg sync.WaitGroup
	for i := range customers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = store.Issue(ctx, request(s, fmt.Sprintf("54911000000%02d", i), raceCode(i)))
		}()
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, _ = store.SetQuotaLimit(ctx, s.business, intp(5), start, end)
	}()
	wg.Wait()

	got, err := store.Quota(ctx, s.business, start, end)
	if err != nil {
		t.Fatalf("Quota: %v", err)
	}
	if got.Limit != nil && got.Generated > *got.Limit {
		t.Errorf("generated %d with a limit of %d: the limit was overshot", got.Generated, *got.Limit)
	}
}
