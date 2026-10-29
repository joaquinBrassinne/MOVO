package queries

import (
	"context"
	"errors"
	"fmt"

	"github.com/joaquinBrassinne/MOVO/internal/reviews"
	"github.com/joaquinBrassinne/MOVO/internal/reviews/domain"
)

// ErrQueueStatusInvalid means the admin asked for a queue that does not exist: only
// reported and hidden reviews are worked on.
var ErrQueueStatusInvalid = errors.New("reviews: the queue must be reported or hidden")

// ModerationInput selects a page, with the same cursor and limits as the public list.
type ModerationInput struct {
	Cursor string
	Limit  int
}

// ModerationPage is one page, newest first. NextCursor is empty on the last page.
type ModerationPage struct {
	Reviews    []reviews.ModerationReview
	NextCursor string
}

// paging validates the cursor and the page size. It returns the size to ask the store
// for: one more than the page, to know whether another follows without a count.
func paging(in ModerationInput) (after *domain.Cursor, limit int, err error) {
	limit = in.Limit
	if limit == 0 {
		limit = DefaultLimit
	}
	if limit < 0 || limit > MaxLimit {
		return nil, 0, ErrLimitInvalid
	}
	if in.Cursor != "" {
		cursor, err := domain.ParseCursor(in.Cursor)
		if err != nil {
			return nil, 0, err
		}
		after = &cursor
	}
	return after, limit, nil
}

func moderationPage(found []reviews.ModerationReview, limit int) ModerationPage {
	if len(found) <= limit {
		return ModerationPage{Reviews: found}
	}
	page := found[:limit]
	last := page[limit-1]
	return ModerationPage{Reviews: page, NextCursor: domain.Cursor{CreatedAt: last.CreatedAt, ID: last.ID}.Encode()}
}

// ListOwnReviews lists the reviews of the owner's business with the id that reporting
// needs. The business comes from the session.
type ListOwnReviews struct {
	owners reviews.OwnerBusinesses
	reader reviews.ModerationReader
}

// NewListOwnReviews wires the ListOwnReviews query.
func NewListOwnReviews(owners reviews.OwnerBusinesses, reader reviews.ModerationReader) *ListOwnReviews {
	return &ListOwnReviews{owners: owners, reader: reader}
}

// Handle returns the page of the owner's business.
func (q *ListOwnReviews) Handle(ctx context.Context, ownerID int64, in ModerationInput) (ModerationPage, error) {
	after, limit, err := paging(in)
	if err != nil {
		return ModerationPage{}, err
	}
	businessID, err := q.owners.EditableBusinessOf(ctx, ownerID)
	if err != nil {
		return ModerationPage{}, fmt.Errorf("find business of owner: %w", err)
	}
	found, err := q.reader.ListOfBusiness(ctx, businessID, after, limit+1)
	if err != nil {
		return ModerationPage{}, fmt.Errorf("list own reviews: %w", err)
	}
	return moderationPage(found, limit), nil
}

// ListReviewsToModerate is the super admin's queue: the reviews owners reported, and the
// hidden ones that can still be restored. The role is checked by the HTTP group.
type ListReviewsToModerate struct {
	reader reviews.ModerationReader
}

// NewListReviewsToModerate wires the ListReviewsToModerate query.
func NewListReviewsToModerate(reader reviews.ModerationReader) *ListReviewsToModerate {
	return &ListReviewsToModerate{reader: reader}
}

// Handle returns the page of the queue for status, reported or hidden.
func (q *ListReviewsToModerate) Handle(ctx context.Context, status domain.Status, in ModerationInput) (ModerationPage, error) {
	if status != domain.StatusReported && status != domain.StatusHidden {
		return ModerationPage{}, ErrQueueStatusInvalid
	}
	after, limit, err := paging(in)
	if err != nil {
		return ModerationPage{}, err
	}
	found, err := q.reader.ListByStatus(ctx, status, after, limit+1)
	if err != nil {
		return ModerationPage{}, fmt.Errorf("list reviews to moderate: %w", err)
	}
	return moderationPage(found, limit), nil
}
