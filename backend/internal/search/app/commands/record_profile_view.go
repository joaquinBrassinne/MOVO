package commands

import (
	"context"
	"errors"
	"log/slog"

	"github.com/joaquinBrassinne/MOVO/internal/search"
)

// RecordProfileView counts a view of the public page of a business. The page asks for
// it from the browser once it has loaded, so the server render that crawlers and
// previewers trigger is never counted (SDD 4.3).
type RecordProfileView struct {
	businesses search.BusinessResolver
	record     *RecordMetric
	log        *slog.Logger
}

// NewRecordProfileView builds the command. A nil logger discards logs.
func NewRecordProfileView(businesses search.BusinessResolver, record *RecordMetric, log *slog.Logger) *RecordProfileView {
	if log == nil {
		log = slog.New(slog.DiscardHandler)
	}
	return &RecordProfileView{businesses: businesses, record: record, log: log}
}

// Handle counts one view and answers nothing about it: a bot, a business that does
// not exist and one that is not public all look the same to the caller, so the call
// cannot be used to find out which slugs exist. A counter that fails is logged and
// swallowed; only a failure to look the business up is an error.
func (r *RecordProfileView) Handle(ctx context.Context, slug, userAgent string) error {
	id, err := r.businesses.PublicBusinessID(ctx, slug)
	if errors.Is(err, search.ErrBusinessNotFound) {
		return nil
	}
	if err != nil {
		return err
	}

	if _, err := r.record.Handle(ctx, search.Event{Metric: search.ProfileView, BusinessID: id}, userAgent); err != nil {
		r.log.Warn("profile view not counted", "business_id", id, "error", err)
	}
	return nil
}
