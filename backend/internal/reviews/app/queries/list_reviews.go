// Package queries holds the use cases that read reviews.
package queries

import (
	"context"
	"errors"
	"fmt"

	"github.com/joaquinBrassinne/MOVO/internal/reviews"
	"github.com/joaquinBrassinne/MOVO/internal/reviews/domain"
)

// Page sizes of the public listing.
const (
	DefaultLimit = 20
	MaxLimit     = 50
)

// ErrLimitInvalid means the page size is negative or above MaxLimit.
var ErrLimitInvalid = errors.New("reviews: the page size is not valid")

// ListReviewsInput selects a page. An empty Cursor is the first page and a zero Limit
// takes DefaultLimit.
type ListReviewsInput struct {
	BranchID int64
	Cursor   string
	Limit    int
}

// ReviewsPage is one page of reviews, newest first. NextCursor is empty on the last page.
type ReviewsPage struct {
	Reviews    []reviews.ListedReview
	NextCursor string
}

// ListReviews serves the public reviews of a branch (SDD 5.5).
type ListReviews struct {
	reader reviews.ReviewReader
}

// NewListReviews wires the ListReviews query.
func NewListReviews(reader reviews.ReviewReader) *ListReviews {
	return &ListReviews{reader: reader}
}

// Handle returns the page. It asks the store for one review more than the page to know
// whether another page follows, without a count. It answers domain.ErrCursorInvalid
// before reading anything, and reviews.ErrBranchNotFound for a branch the public cannot
// see.
func (q *ListReviews) Handle(ctx context.Context, in ListReviewsInput) (ReviewsPage, error) {
	limit := in.Limit
	if limit == 0 {
		limit = DefaultLimit
	}
	if limit < 0 || limit > MaxLimit {
		return ReviewsPage{}, ErrLimitInvalid
	}
	var after *domain.Cursor
	if in.Cursor != "" {
		cursor, err := domain.ParseCursor(in.Cursor)
		if err != nil {
			return ReviewsPage{}, err
		}
		after = &cursor
	}

	found, err := q.reader.ListPublic(ctx, in.BranchID, after, limit+1)
	if err != nil {
		return ReviewsPage{}, fmt.Errorf("list reviews: %w", err)
	}
	if len(found) <= limit {
		return ReviewsPage{Reviews: found}, nil
	}
	page := found[:limit]
	last := page[limit-1]
	return ReviewsPage{Reviews: page, NextCursor: domain.Cursor{CreatedAt: last.CreatedAt, ID: last.ID}.Encode()}, nil
}
