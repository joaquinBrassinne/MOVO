package commands_test

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/reviews"
	"github.com/joaquinBrassinne/MOVO/internal/reviews/app/commands"
	"github.com/joaquinBrassinne/MOVO/internal/reviews/domain"
)

var (
	redeemedAt = time.Date(2028, 3, 1, 10, 0, 0, 0, time.UTC)
	now        = redeemedAt.Add(24 * time.Hour)
)

type fakeReader struct {
	redemption domain.Redemption
	err        error
	gotHash    []byte
}

func (f *fakeReader) ByTokenHash(_ context.Context, hash []byte) (domain.Redemption, error) {
	f.gotHash = hash
	return f.redemption, f.err
}

type fakeStore struct {
	err     error
	created []domain.Review
}

func (f *fakeStore) Create(_ context.Context, review domain.Review) error {
	if f.err != nil {
		return f.err
	}
	f.created = append(f.created, review)
	return nil
}

func redeemed() domain.Redemption {
	return domain.Redemption{CouponID: 7, BranchID: 3, BusinessID: 2, CustomerID: 9, CustomerName: "Ana García", RedeemedAt: redeemedAt}
}

func newCommand(reader *fakeReader, store *fakeStore) *commands.CreateReview {
	return commands.NewCreateReview(reader, store, func() time.Time { return now })
}

func TestCreateReviewStoresTheReviewOfTheCouponOfTheToken(t *testing.T) {
	t.Parallel()

	reader, store := &fakeReader{redemption: redeemed()}, &fakeStore{}
	got, err := newCommand(reader, store).Handle(t.Context(), commands.CreateReviewInput{Token: "secret-token", Rating: 5, Comment: "Genial", ShowName: true})
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if len(store.created) != 1 || store.created[0] != got {
		t.Fatalf("stored = %+v, want exactly the returned review %+v", store.created, got)
	}
	if got.BranchID != 3 || got.CustomerID != 9 || got.DisplayName != "Ana G." || got.Rating != 5 || !got.CreatedAt.Equal(now) {
		t.Errorf("review = %+v, want the branch and customer of the redemption, the shown name and the clock", got)
	}
	if want := domain.HashAccessToken("secret-token"); !bytes.Equal(reader.gotHash, want) {
		t.Errorf("the reader got %x, want only the hash %x", reader.gotHash, want)
	}
}

func TestCreateReviewRefusesWithoutStoring(t *testing.T) {
	t.Parallel()

	pending := redeemed()
	pending.RedeemedAt = time.Time{}
	late := redeemed()
	late.RedeemedAt = now.Add(-31 * 24 * time.Hour)

	tests := []struct {
		name   string
		reader *fakeReader
		in     commands.CreateReviewInput
		want   error
	}{
		{"unknown token", &fakeReader{err: reviews.ErrCouponNotFound}, commands.CreateReviewInput{Token: "x", Rating: 5}, reviews.ErrCouponNotFound},
		{"coupon not redeemed", &fakeReader{redemption: pending}, commands.CreateReviewInput{Token: "x", Rating: 5}, domain.ErrNotRedeemed},
		{"window over", &fakeReader{redemption: late}, commands.CreateReviewInput{Token: "x", Rating: 5}, domain.ErrWindowClosed},
		{"rating out of range", &fakeReader{redemption: redeemed()}, commands.CreateReviewInput{Token: "x", Rating: 6}, domain.ErrRatingInvalid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			store := &fakeStore{}
			if _, err := newCommand(tt.reader, store).Handle(t.Context(), tt.in); !errors.Is(err, tt.want) {
				t.Errorf("err = %v, want %v", err, tt.want)
			}
			if len(store.created) != 0 {
				t.Errorf("stored %+v, want nothing", store.created)
			}
		})
	}
}

func TestCreateReviewReportsACouponThatWasAlreadyReviewed(t *testing.T) {
	t.Parallel()

	store := &fakeStore{err: reviews.ErrAlreadyReviewed}
	_, err := newCommand(&fakeReader{redemption: redeemed()}, store).Handle(t.Context(), commands.CreateReviewInput{Token: "x", Rating: 5})
	if !errors.Is(err, reviews.ErrAlreadyReviewed) {
		t.Errorf("err = %v, want ErrAlreadyReviewed", err)
	}
}
