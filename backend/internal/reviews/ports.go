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
	// ErrReviewNotFound covers a review that does not exist and, for an owner, one that
	// belongs to another business, so a report never reveals which is which.
	ErrReviewNotFound = errors.New("reviews: review not found")
	// ErrBusinessNotFound means the owner has not created a business yet.
	ErrBusinessNotFound = errors.New("reviews: business not found")
	// ErrBusinessNotEditable means the business was deleted, so its owner can no longer act.
	ErrBusinessNotEditable = errors.New("reviews: business can no longer change")
	// ErrReviewPurged means the review was hidden long enough for its text to be erased,
	// so it cannot be made visible again.
	ErrReviewPurged = errors.New("reviews: the review was purged and cannot be restored")
)

// OwnerBusinesses answers which business an owner may act on. It is the consumer-side
// view of the business module, joined in cmd/api so reviews never reads it.
type OwnerBusinesses interface {
	// EditableBusinessOf returns the business of the owner, ErrBusinessNotFound when
	// there is none and ErrBusinessNotEditable when it was deleted.
	EditableBusinessOf(ctx context.Context, ownerID int64) (int64, error)
}

// ModeratedReview is the moderation state of a review after a change.
type ModeratedReview struct {
	ID     int64
	Status domain.Status
}

// ReviewModerator changes the moderation state of reviews, and records who did it.
type ReviewModerator interface {
	// Report marks a review of the business as reported. Only a visible review changes
	// state; repeating the report, or reporting a hidden one, changes nothing and records
	// nothing. A review of another business answers ErrReviewNotFound.
	Report(ctx context.Context, actorID, businessID, reviewID int64) (ModeratedReview, error)
	// SetStatus hides a review or makes it visible again, which is also how a report is
	// dismissed, and recounts the rating of its branch in the same transaction. Repeating
	// the current state changes nothing and records nothing.
	SetStatus(ctx context.Context, actorID, reviewID int64, status domain.Status) (ModeratedReview, error)
}

// ModerationReview is a review as the people who moderate it read it: it carries the id
// that reporting and hiding need, and where it was left.
type ModerationReview struct {
	ID           int64
	BranchID     int64
	BranchName   string
	BusinessID   int64
	BusinessName string
	Rating       int
	Comment      string // empty when none
	DisplayName  string
	Status       domain.Status
	CreatedAt    time.Time
}

// ModerationReader lists reviews for the owner who may report them and the super admin
// who hides them. Both are newest first and paged by the same cursor as the public list.
type ModerationReader interface {
	// ListOfBusiness returns the reviews of the business that are visible or reported;
	// a hidden one is not the owner's to act on any more.
	ListOfBusiness(ctx context.Context, businessID int64, after *domain.Cursor, limit int) ([]ModerationReview, error)
	// ListByStatus returns the reviews in that status. A purged review has no text left
	// and is not listed.
	ListByStatus(ctx context.Context, status domain.Status, after *domain.Cursor, limit int) ([]ModerationReview, error)
}

// HiddenReviewPurger erases the text of reviews that stayed hidden long enough.
type HiddenReviewPurger interface {
	// PurgeHidden erases the comment and the name of every hidden review hidden at or
	// before hiddenBefore and returns how many it erased. The row, its rating and its
	// coupon stay, so the coupon cannot be reviewed again. Running it twice erases nothing
	// the second time.
	PurgeHidden(ctx context.Context, hiddenBefore time.Time) (int64, error)
}

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

// RatingSummary is how a branch's reviews add up: how many there are and how many gave
// each number of stars. Stars[0] counts the 1-star reviews and Stars[4] the 5-star ones.
type RatingSummary struct {
	Count int
	Stars [5]int
}

// Average is the mean rating, 0 when there are no reviews.
func (s RatingSummary) Average() float64 {
	if s.Count == 0 {
		return 0
	}
	sum := 0
	for i, n := range s.Stars {
		sum += (i + 1) * n
	}
	return float64(sum) / float64(s.Count)
}

// ReviewReader reads the reviews the public sees.
type ReviewReader interface {
	// ListPublic returns up to limit reviews of the branch that pass the star filter,
	// newest first, starting after the cursor when there is one. A reported review is
	// listed: only the super admin hides one. It returns ErrBranchNotFound when the branch
	// is not publicly visible.
	ListPublic(ctx context.Context, branchID int64, stars domain.StarFilter, after *domain.Cursor, limit int) ([]ListedReview, error)
	// Summary counts the public reviews of the branch by stars, with the same visibility
	// as ListPublic.
	Summary(ctx context.Context, branchID int64) (RatingSummary, error)
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
