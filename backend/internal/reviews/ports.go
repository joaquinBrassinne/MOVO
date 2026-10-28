// Package reviews is the module that lets a customer review a business after redeeming a
// coupon there (SDD 5.5). This file declares the ports the application layer consumes;
// adapters implement them.
package reviews

import (
	"context"
	"errors"

	"github.com/joaquinBrassinne/MOVO/internal/reviews/domain"
)

// Errors the adapters report so the application layer can react without knowing the
// storage engine.
var (
	// ErrCouponNotFound covers a token that does not open any coupon, whatever it looks
	// like.
	ErrCouponNotFound = errors.New("reviews: coupon not found")
	// ErrAlreadyReviewed means that coupon already has its review.
	ErrAlreadyReviewed = errors.New("reviews: the coupon was already reviewed")
)

// RedemptionReader finds the coupon an access token opens.
type RedemptionReader interface {
	// ByTokenHash returns the redemption of the coupon whose access token has that hash,
	// or ErrCouponNotFound. A coupon that is not redeemed comes back with a zero
	// RedeemedAt; deciding what that means is the domain's job.
	ByTokenHash(ctx context.Context, tokenHash []byte) (domain.Redemption, error)
}

// ReviewStore keeps reviews.
type ReviewStore interface {
	// Create stores the review, or returns ErrAlreadyReviewed when its coupon has one.
	Create(ctx context.Context, review domain.Review) error
}
