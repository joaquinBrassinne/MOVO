// Package domain holds the rules of the reviews verified by redemption (SDD 5.5).
package domain

import (
	"errors"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// ReviewWindow is how long after redeeming a coupon its customer can review (SDD 5.5).
const ReviewWindow = 30 * 24 * time.Hour

// AnonymousName is shown when the customer chose not to show their name.
const AnonymousName = "Cliente de MOVO"

const maxCommentLen = 500

// Errors a review is refused with.
var (
	ErrNotRedeemed   = errors.New("reviews: the coupon was not redeemed")
	ErrWindowClosed  = errors.New("reviews: the 30 days to review this coupon are over")
	ErrRatingInvalid = errors.New("reviews: the rating must be from 1 to 5")
	// ErrStatusInvalid means the super admin asked for a state only hiding or restoring
	// can reach: a review is reported by its business, never set by the admin.
	ErrStatusInvalid = errors.New("reviews: the status must be hidden or visible")
)

// Status is where a review is in moderation.
type Status string

// Review statuses (SDD 4).
const (
	StatusVisible  Status = "visible"
	StatusReported Status = "reported"
	StatusHidden   Status = "hidden"
)

// Redemption is what the reviews module needs to know about a coupon: where and by whom
// it was redeemed. A zero RedeemedAt means the coupon is not redeemed.
type Redemption struct {
	CouponID     int64
	BranchID     int64 // the branch that redeemed it
	BusinessID   int64
	CustomerID   int64
	CustomerName string
	RedeemedAt   time.Time
}

// Input is what the customer fills in. There is no name: it comes from the coupon.
type Input struct {
	Rating   int
	Comment  string // optional
	ShowName bool
}

// Review is a rating a customer left after redeeming a coupon.
type Review struct {
	CouponID    int64
	BranchID    int64
	BusinessID  int64
	CustomerID  int64
	Rating      int
	Comment     string // empty when none
	DisplayName string
	ShowName    bool
	Status      Status
	CreatedAt   time.Time
}

// NewReview checks the rules of SDD 5.5 and returns the review to store: the coupon must
// be redeemed, the review must come within ReviewWindow of the redemption, the rating is
// 1 to 5 and the comment is optional plain text. The branch is the one that redeemed the
// coupon. That a coupon has only one review is the store's job (a unique key).
func NewReview(r Redemption, in Input, now time.Time) (Review, error) {
	if r.RedeemedAt.IsZero() {
		return Review{}, ErrNotRedeemed
	}
	if now.After(r.RedeemedAt.Add(ReviewWindow)) {
		return Review{}, ErrWindowClosed
	}
	if in.Rating < 1 || in.Rating > 5 {
		return Review{}, ErrRatingInvalid
	}
	comment, err := plainText("comment", in.Comment, maxCommentLen)
	if err != nil {
		return Review{}, err
	}

	name := AnonymousName
	if in.ShowName {
		name = displayName(r.CustomerName)
	}
	return Review{
		CouponID:    r.CouponID,
		BranchID:    r.BranchID,
		BusinessID:  r.BusinessID,
		CustomerID:  r.CustomerID,
		Rating:      in.Rating,
		Comment:     comment,
		DisplayName: name,
		ShowName:    in.ShowName,
		Status:      StatusVisible,
		CreatedAt:   now,
	}, nil
}

// displayName is the first word of the customer's name plus the initial of the last one
// ("Ana G."). The customer typed their name freely, so a name that carries a phone number
// or a link is never shown.
func displayName(customerName string) string {
	words := strings.Fields(customerName)
	if len(words) == 0 {
		return AnonymousName
	}
	if _, err := plainText("name", customerName, utf8.RuneCountInString(customerName)); err != nil {
		return AnonymousName
	}
	if len(words) == 1 {
		return words[0]
	}
	initial, _ := utf8.DecodeRuneInString(words[len(words)-1])
	return words[0] + " " + string(unicode.ToUpper(initial)) + "."
}
