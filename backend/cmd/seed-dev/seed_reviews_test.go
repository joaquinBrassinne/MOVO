package main

import (
	"testing"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/platform/dbtest"
	reviewspostgres "github.com/joaquinBrassinne/MOVO/internal/reviews/adapters/postgres"
	"github.com/joaquinBrassinne/MOVO/internal/reviews/domain"
)

// TestSeedGivesTheDemoBusinessReviewsInEveryModerationState runs the three scripts twice:
// the reviews must not be duplicated, the rating of the branch must count them like the
// reviews module does, and the demo tokens written in the script must open the states it
// promises.
func TestSeedGivesTheDemoBusinessReviewsInEveryModerationState(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	ctx := t.Context()
	for range 2 {
		for _, script := range []string{seedSQL, seedCouponsSQL, seedReviewsSQL} {
			if _, err := pool.Exec(ctx, script); err != nil {
				t.Fatalf("seed: %v", err)
			}
		}
	}

	var visible, reported, hidden, purged, branchCount int
	err := pool.QueryRow(ctx, `
SELECT count(*) FILTER (WHERE r.status = 'visible'), count(*) FILTER (WHERE r.status = 'reported'),
       count(*) FILTER (WHERE r.status = 'hidden'),  count(*) FILTER (WHERE r.purged_at IS NOT NULL),
       (SELECT rating_count FROM branches WHERE id = min(r.branch_id))
FROM reviews r JOIN businesses b ON b.id = r.business_id WHERE b.slug = 'alma-haus-training-club'`).
		Scan(&visible, &reported, &hidden, &purged, &branchCount)
	if err != nil {
		t.Fatalf("read the reviews: %v", err)
	}
	if visible != 20 || reported != 2 || hidden != 2 || purged != 1 || branchCount != 22 {
		t.Errorf("%d visible, %d reported, %d hidden (%d purged), rating_count %d; want 20, 2, 2 (1), 22: a reported review counts and a hidden one does not",
			visible, reported, hidden, purged, branchCount)
	}

	repo := reviewspostgres.NewReviews(pool)
	for token, want := range map[string]domain.FormState{
		"seed-review-open-00000000000000000000000000": domain.FormOpen,
		"seed-review-closed-000000000000000000000000": domain.FormWindowClosed,
		"seed-review-pending-00000000000000000000000": domain.FormNotRedeemed,
		"seed-reviewed-01000000000000000000000000000": domain.FormReviewed,
	} {
		data, err := repo.FormByTokenHash(ctx, domain.HashAccessToken(token))
		if err != nil {
			t.Errorf("token %s: %v", token, err)
			continue
		}
		if state := domain.StateOf(data.RedeemedAt, data.Reviewed, time.Now()); state != want {
			t.Errorf("token %s = %q, want %q", token, state, want)
		}
	}
}
