package queries

import (
	"context"
	"fmt"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/reviews"
	"github.com/joaquinBrassinne/MOVO/internal/reviews/domain"
)

// ReviewForm is what "Dejá tu reseña" needs before the customer writes anything: where
// they were, the name their review would show and whether they can still review.
type ReviewForm struct {
	State        domain.FormState
	BusinessName string
	BranchName   string
	City         string
	Code         string
	DisplayName  string
	RedeemedAt   time.Time // zero when the coupon was not redeemed
	ReviewUntil  time.Time // zero when the coupon was not redeemed
}

// GetReviewForm opens the review form of the coupon an access token opens (SDD 5.5).
type GetReviewForm struct {
	reader reviews.FormReader
	now    func() time.Time
}

// NewGetReviewForm wires the GetReviewForm query. A nil clock uses time.Now.
func NewGetReviewForm(reader reviews.FormReader, now func() time.Time) *GetReviewForm {
	if now == nil {
		now = time.Now
	}
	return &GetReviewForm{reader: reader, now: now}
}

// Handle finds the coupon of the token (only its hash reaches the store) and says what the
// form should show. An unknown token answers reviews.ErrCouponNotFound.
func (q *GetReviewForm) Handle(ctx context.Context, token string) (ReviewForm, error) {
	data, err := q.reader.FormByTokenHash(ctx, domain.HashAccessToken(token))
	if err != nil {
		return ReviewForm{}, fmt.Errorf("read review form: %w", err)
	}
	out := ReviewForm{
		State:        domain.StateOf(data.RedeemedAt, data.Reviewed, q.now()),
		BusinessName: data.BusinessName,
		BranchName:   data.BranchName,
		City:         data.City,
		Code:         data.Code,
		DisplayName:  domain.DisplayName(data.CustomerName),
		RedeemedAt:   data.RedeemedAt,
	}
	if !data.RedeemedAt.IsZero() {
		out.ReviewUntil = data.RedeemedAt.Add(domain.ReviewWindow)
	}
	return out, nil
}
