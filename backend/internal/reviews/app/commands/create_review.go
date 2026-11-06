// Package commands holds the use cases that change reviews.
package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/reviews"
	"github.com/joaquinBrassinne/MOVO/internal/reviews/domain"
)

// CreateReviewInput is what a customer sends. The token is the whole credential: the
// branch, the business and the customer come from the coupon it opens, never from the
// request.
type CreateReviewInput struct {
	Token    string
	Rating   int
	Comment  string
	ShowName bool
}

// CreateReview lets the customer of a redeemed coupon review it (SDD 5.5).
type CreateReview struct {
	redemptions reviews.RedemptionReader
	store       reviews.ReviewStore
	now         func() time.Time
}

// NewCreateReview wires the CreateReview use case. A nil clock uses time.Now.
func NewCreateReview(redemptions reviews.RedemptionReader, store reviews.ReviewStore, now func() time.Time) *CreateReview {
	if now == nil {
		now = time.Now
	}
	return &CreateReview{redemptions: redemptions, store: store, now: now}
}

// Handle finds the coupon of the token (only its hash reaches the store), applies the
// rules of the domain and stores the review. An unknown token answers
// reviews.ErrCouponNotFound whatever it looks like; a second review of the same coupon
// answers reviews.ErrAlreadyReviewed.
func (c *CreateReview) Handle(ctx context.Context, in CreateReviewInput) (domain.Review, error) {
	redemption, err := c.redemptions.ByTokenHash(ctx, domain.HashAccessToken(in.Token))
	if err != nil {
		return domain.Review{}, fmt.Errorf("read redemption: %w", err)
	}

	review, err := domain.NewReview(redemption, domain.Input{Rating: in.Rating, Comment: in.Comment, ShowName: in.ShowName}, c.now())
	if err != nil {
		return domain.Review{}, err
	}
	if err := c.store.Create(ctx, review); err != nil {
		return domain.Review{}, fmt.Errorf("create review: %w", err)
	}
	return review, nil
}
