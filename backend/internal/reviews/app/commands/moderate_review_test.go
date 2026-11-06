package commands_test

import (
	"context"
	"errors"
	"testing"

	"github.com/joaquinBrassinne/MOVO/internal/reviews"
	"github.com/joaquinBrassinne/MOVO/internal/reviews/app/commands"
	"github.com/joaquinBrassinne/MOVO/internal/reviews/domain"
)

type fakeOwners struct {
	business int64
	err      error
}

func (f fakeOwners) EditableBusinessOf(context.Context, int64) (int64, error) {
	return f.business, f.err
}

type fakeModerator struct {
	err        error
	gotActor   int64
	gotBiz     int64
	gotReview  int64
	gotStatus  domain.Status
	reportRuns int
	statusRuns int
}

func (f *fakeModerator) Report(_ context.Context, actorID, businessID, reviewID int64) (reviews.ModeratedReview, error) {
	f.reportRuns++
	f.gotActor, f.gotBiz, f.gotReview = actorID, businessID, reviewID
	return reviews.ModeratedReview{ID: reviewID, Status: domain.StatusReported}, f.err
}

func (f *fakeModerator) SetStatus(_ context.Context, actorID, reviewID int64, status domain.Status) (reviews.ModeratedReview, error) {
	f.statusRuns++
	f.gotActor, f.gotReview, f.gotStatus = actorID, reviewID, status
	return reviews.ModeratedReview{ID: reviewID, Status: status}, f.err
}

func TestReportReviewActsOnTheBusinessOfTheSessionNeverOnOneFromTheRequest(t *testing.T) {
	t.Parallel()

	mod := &fakeModerator{}
	got, err := commands.NewReportReview(fakeOwners{business: 42}, mod).Handle(t.Context(), 7, 99)
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if mod.gotActor != 7 || mod.gotBiz != 42 || mod.gotReview != 99 {
		t.Errorf("moderator got actor %d, business %d, review %d; want 7, 42 (the owner's), 99", mod.gotActor, mod.gotBiz, mod.gotReview)
	}
	if got.Status != domain.StatusReported {
		t.Errorf("status = %q, want reported", got.Status)
	}
}

func TestReportReviewStopsBeforeTouchingAnythingWhenTheOwnerCannotAct(t *testing.T) {
	t.Parallel()

	for _, want := range []error{reviews.ErrBusinessNotFound, reviews.ErrBusinessNotEditable} {
		mod := &fakeModerator{}
		if _, err := commands.NewReportReview(fakeOwners{err: want}, mod).Handle(t.Context(), 7, 99); !errors.Is(err, want) {
			t.Errorf("err = %v, want %v", err, want)
		}
		if mod.reportRuns != 0 {
			t.Errorf("the moderator ran %d times, want none", mod.reportRuns)
		}
	}
}

func TestReportReviewReportsAReviewThatIsNotTheirs(t *testing.T) {
	t.Parallel()

	mod := &fakeModerator{err: reviews.ErrReviewNotFound}
	if _, err := commands.NewReportReview(fakeOwners{business: 42}, mod).Handle(t.Context(), 7, 99); !errors.Is(err, reviews.ErrReviewNotFound) {
		t.Errorf("err = %v, want ErrReviewNotFound", err)
	}
}

func TestSetReviewStatusOnlyHidesOrRestores(t *testing.T) {
	t.Parallel()

	for _, ok := range []domain.Status{domain.StatusHidden, domain.StatusVisible} {
		mod := &fakeModerator{}
		got, err := commands.NewSetReviewStatus(mod).Handle(t.Context(), 1, 99, ok)
		if err != nil || got.Status != ok || mod.gotActor != 1 {
			t.Errorf("status %q: got %+v, %v, actor %d; want it applied on behalf of actor 1", ok, got, err, mod.gotActor)
		}
	}

	// A report is the owner's move: the admin cannot set it, nor invent a state.
	for _, bad := range []domain.Status{domain.StatusReported, "deleted", ""} {
		mod := &fakeModerator{}
		if _, err := commands.NewSetReviewStatus(mod).Handle(t.Context(), 1, 99, bad); !errors.Is(err, domain.ErrStatusInvalid) {
			t.Errorf("status %q: err = %v, want ErrStatusInvalid", bad, err)
		}
		if mod.statusRuns != 0 {
			t.Errorf("status %q: the moderator ran, want it untouched", bad)
		}
	}
}

func TestSetReviewStatusReportsAReviewThatDoesNotExist(t *testing.T) {
	t.Parallel()

	mod := &fakeModerator{err: reviews.ErrReviewNotFound}
	if _, err := commands.NewSetReviewStatus(mod).Handle(t.Context(), 1, 99, domain.StatusHidden); !errors.Is(err, reviews.ErrReviewNotFound) {
		t.Errorf("err = %v, want ErrReviewNotFound", err)
	}
}
