package commands

import (
	"context"
	"fmt"

	"github.com/joaquinBrassinne/MOVO/internal/reviews"
	"github.com/joaquinBrassinne/MOVO/internal/reviews/domain"
)

// ReportReview lets a business report a review of its own to the super admin (SDD 5.5).
// Reporting does not hide anything: only the super admin hides a review, so an owner
// cannot silence a bad review by reporting it.
type ReportReview struct {
	owners    reviews.OwnerBusinesses
	moderator reviews.ReviewModerator
}

// NewReportReview wires the ReportReview use case.
func NewReportReview(owners reviews.OwnerBusinesses, moderator reviews.ReviewModerator) *ReportReview {
	return &ReportReview{owners: owners, moderator: moderator}
}

// Handle reports the review on behalf of the owner. The business comes from the owner's
// session, never from the request, so a review of another business answers
// reviews.ErrReviewNotFound like one that does not exist.
func (c *ReportReview) Handle(ctx context.Context, ownerID, reviewID int64) (reviews.ModeratedReview, error) {
	businessID, err := c.owners.EditableBusinessOf(ctx, ownerID)
	if err != nil {
		return reviews.ModeratedReview{}, fmt.Errorf("find business of owner: %w", err)
	}
	got, err := c.moderator.Report(ctx, ownerID, businessID, reviewID)
	if err != nil {
		return reviews.ModeratedReview{}, fmt.Errorf("report review: %w", err)
	}
	return got, nil
}

// SetReviewStatus lets the super admin hide a review or make it visible again, which is
// also how a report is dismissed (SDD 5.5). The role is checked by the HTTP group, like
// the rest of /admin.
type SetReviewStatus struct {
	moderator reviews.ReviewModerator
}

// NewSetReviewStatus wires the SetReviewStatus use case.
func NewSetReviewStatus(moderator reviews.ReviewModerator) *SetReviewStatus {
	return &SetReviewStatus{moderator: moderator}
}

// Handle sets the status on behalf of actorID. Only hidden and visible are accepted.
func (c *SetReviewStatus) Handle(ctx context.Context, actorID, reviewID int64, status domain.Status) (reviews.ModeratedReview, error) {
	if status != domain.StatusHidden && status != domain.StatusVisible {
		return reviews.ModeratedReview{}, domain.ErrStatusInvalid
	}
	got, err := c.moderator.SetStatus(ctx, actorID, reviewID, status)
	if err != nil {
		return reviews.ModeratedReview{}, fmt.Errorf("set review status: %w", err)
	}
	return got, nil
}
