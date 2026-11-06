package commands_test

import (
	"context"
	"errors"
	"testing"

	"github.com/joaquinBrassinne/MOVO/internal/benefits"
	"github.com/joaquinBrassinne/MOVO/internal/benefits/app/commands"
)

// fakeOwners resolves every owner to a fixed business, or fails with err.
type fakeOwners struct {
	business int64
	err      error
}

func (f fakeOwners) EditableBusinessOf(context.Context, int64) (int64, error) {
	return f.business, f.err
}

// fakeRedeemer records the requests and answers with err.
type fakeRedeemer struct {
	err   error
	calls []benefits.RedeemRequest
}

func (f *fakeRedeemer) Redeem(_ context.Context, req benefits.RedeemRequest) (benefits.Redeemed, error) {
	f.calls = append(f.calls, req)
	return benefits.Redeemed{}, f.err
}

func TestRedeemCodeTakesTheBusinessAndTheUserFromTheSession(t *testing.T) {
	t.Parallel()

	redeemer := &fakeRedeemer{}
	_, err := commands.NewRedeemCode(fakeOwners{business: 42}, redeemer, fixedNow).Handle(t.Context(),
		commands.RedeemCodeInput{OwnerID: 7, Code: " abcd-efgh ", BranchID: 3})
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}

	got := redeemer.calls[0]
	if got.BusinessID != 42 || got.UserID != 7 || got.Code != "ABCDEFGH" || got.BranchID != 3 || !got.Now.Equal(fixedNow()) {
		t.Errorf("request = %+v, want business 42 and user 7 from the session, the normalized code, branch 3 and the clock", got)
	}
}

func TestRedeemCodeRefusesAMalformedCodeAsNotFoundWithoutRedeeming(t *testing.T) {
	t.Parallel()

	redeemer := &fakeRedeemer{}
	_, err := commands.NewRedeemCode(fakeOwners{business: 1}, redeemer, fixedNow).Handle(t.Context(),
		commands.RedeemCodeInput{OwnerID: 7, Code: "not a code"})
	if !errors.Is(err, benefits.ErrCouponNotFound) || len(redeemer.calls) != 0 {
		t.Errorf("error = %v with %d redeem calls, want ErrCouponNotFound and none", err, len(redeemer.calls))
	}
}

func TestRedeemCodeNeedsABusinessThatCanChange(t *testing.T) {
	t.Parallel()

	for _, want := range []error{benefits.ErrBusinessNotFound, benefits.ErrBusinessNotEditable} {
		redeemer := &fakeRedeemer{}
		_, err := commands.NewRedeemCode(fakeOwners{err: want}, redeemer, fixedNow).Handle(t.Context(),
			commands.RedeemCodeInput{OwnerID: 7, Code: "ABCDEFGH"})
		if !errors.Is(err, want) || len(redeemer.calls) != 0 {
			t.Errorf("error = %v with %d redeem calls, want %v and none", err, len(redeemer.calls), want)
		}
	}
}

func TestRedeemCodePassesTheReasonsThrough(t *testing.T) {
	t.Parallel()

	for _, want := range []error{benefits.ErrCouponNotFound, benefits.ErrCouponRedeemed, benefits.ErrCouponExpired, benefits.ErrBranchRequired, benefits.ErrBranchNotFound} {
		_, err := commands.NewRedeemCode(fakeOwners{business: 1}, &fakeRedeemer{err: want}, fixedNow).Handle(t.Context(),
			commands.RedeemCodeInput{OwnerID: 7, Code: "ABCDEFGH"})
		if !errors.Is(err, want) {
			t.Errorf("error = %v, want %v", err, want)
		}
	}
}
