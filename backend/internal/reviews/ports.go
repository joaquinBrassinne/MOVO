// Package reviews is the module that lets a customer review a business after redeeming a
// coupon there (SDD 5.5). This file declares the ports the application layer consumes;
// adapters implement them.
package reviews

import (
	"context"
	"errors"
	"time"

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
	// ErrBranchNotFound covers a branch that does not exist and one the public cannot
	// see, so a listing never reveals which is which.
	ErrBranchNotFound = errors.New("reviews: branch not found")
)

// ListedReview is a review as the public reads it. ID only builds the cursor of the next
// page and never leaves the application layer. There is no customer: the name to show
// was fixed when the review was written.
type ListedReview struct {
	ID          int64
	Rating      int
	Comment     string // empty when none
	DisplayName string
	CreatedAt   time.Time
}

// ReviewReader reads the reviews the public sees.
type ReviewReader interface {
	// ListPublic returns up to limit reviews of the branch, newest first, starting after
	// the cursor when there is one. A reported review is listed: only the super admin
	// hides one. It returns ErrBranchNotFound when the branch is not publicly visible.
	ListPublic(ctx context.Context, branchID int64, after *domain.Cursor, limit int) ([]ListedReview, error)
}

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
