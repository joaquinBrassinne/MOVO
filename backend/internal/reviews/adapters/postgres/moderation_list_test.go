package postgres_test

import (
	"slices"
	"testing"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/platform/dbtest"
	"github.com/joaquinBrassinne/MOVO/internal/reviews"
	"github.com/joaquinBrassinne/MOVO/internal/reviews/adapters/postgres"
	"github.com/joaquinBrassinne/MOVO/internal/reviews/domain"
)

func moderationIDs(list []reviews.ModerationReview) []int64 {
	out := make([]int64, len(list))
	for i, r := range list {
		out[i] = r.ID
	}
	return out
}

func TestListOfBusinessShowsWhatTheOwnerCanReportNewestFirst(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	ctx := t.Context()
	s := newSeed(ctx, t, pool, true)
	repo := postgres.NewReviews(pool)

	oldest := addReview(ctx, t, pool, s, 1, "visible", created)
	reported := addReview(ctx, t, pool, s, 2, "reported", created.Add(time.Hour))
	_ = addReview(ctx, t, pool, s, 3, "hidden", created.Add(2*time.Hour))
	newest := addReview(ctx, t, pool, s, 4, "visible", created.Add(3*time.Hour))

	got, err := repo.ListOfBusiness(ctx, s.business, nil, 10)
	if err != nil {
		t.Fatalf("ListOfBusiness: %v", err)
	}
	if want := []int64{newest, reported, oldest}; !slices.Equal(moderationIDs(got), want) {
		t.Fatalf("reviews = %v, want %v: newest first and without the hidden one", moderationIDs(got), want)
	}
	first := got[0]
	if first.BranchID != s.branch || first.BranchName != "Centro" || first.BusinessID != s.business || first.BusinessName != "Studio" ||
		first.Status != domain.StatusVisible || first.Comment != "Comment 4" || first.DisplayName != "Cliente 4" || first.Rating != 4 {
		t.Errorf("review = %+v, want where it was left, its text and its state", first)
	}
	if got[1].Status != domain.StatusReported {
		t.Errorf("status = %q, want reported: the owner sees what is already reported", got[1].Status)
	}

	other, err := repo.ListOfBusiness(ctx, s.business+1000, nil, 10)
	if err != nil || len(other) != 0 {
		t.Errorf("another business = %v, %v; want nothing: an owner never reads other reviews", other, err)
	}
}

func TestListOfBusinessPagesByKeyset(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	ctx := t.Context()
	s := newSeed(ctx, t, pool, true)
	repo := postgres.NewReviews(pool)
	var want []int64
	for i := range 5 {
		want = append([]int64{addReview(ctx, t, pool, s, i+1, "visible", created.Add(time.Duration(i)*time.Minute))}, want...)
	}

	var got []int64
	var after *domain.Cursor
	for range 5 {
		page, err := repo.ListOfBusiness(ctx, s.business, after, 2)
		if err != nil {
			t.Fatalf("ListOfBusiness: %v", err)
		}
		got = append(got, moderationIDs(page)...)
		if len(page) < 2 {
			break
		}
		last := page[len(page)-1]
		after = &domain.Cursor{CreatedAt: last.CreatedAt, ID: last.ID}
	}
	if !slices.Equal(got, want) {
		t.Errorf("paged reviews = %v, want %v without repeating or skipping one", got, want)
	}
}

func TestListByStatusIsTheAdminQueue(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	ctx := t.Context()
	s := newSeed(ctx, t, pool, true)
	repo := postgres.NewReviews(pool)

	_ = addReview(ctx, t, pool, s, 1, "visible", created)
	reportedOld := addReview(ctx, t, pool, s, 2, "reported", created.Add(time.Hour))
	reportedNew := addReview(ctx, t, pool, s, 3, "reported", created.Add(2*time.Hour))
	_ = addReview(ctx, t, pool, s, 4, "hidden", created.Add(3*time.Hour))
	if n, err := repo.PurgeHidden(ctx, created.Add(24*time.Hour)); err != nil || n != 1 {
		t.Fatalf("purge = %d, %v; want the one hidden review erased", n, err)
	}

	reported, err := repo.ListByStatus(ctx, domain.StatusReported, nil, 10)
	if err != nil || !slices.Equal(moderationIDs(reported), []int64{reportedNew, reportedOld}) {
		t.Errorf("reported = %v, %v; want %v, newest first", moderationIDs(reported), err, []int64{reportedNew, reportedOld})
	}
	hiddenList, err := repo.ListByStatus(ctx, domain.StatusHidden, nil, 10)
	if err != nil || len(hiddenList) != 0 {
		t.Errorf("hidden = %v, %v; want none: the only one was purged and has no text to restore", moderationIDs(hiddenList), err)
	}

	fresh := addReview(ctx, t, pool, s, 5, "hidden", created.Add(4*time.Hour))
	hiddenList, err = repo.ListByStatus(ctx, domain.StatusHidden, nil, 10)
	if err != nil || !slices.Equal(moderationIDs(hiddenList), []int64{fresh}) {
		t.Errorf("hidden = %v, %v; want only %d, the one that can still be restored", moderationIDs(hiddenList), err, fresh)
	}
}
