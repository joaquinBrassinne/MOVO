package commands

import (
	"context"
	"errors"
	"log/slog"

	"github.com/joaquinBrassinne/MOVO/internal/search"
)

// RecordServiceView counts a view of a service on the public page of a branch (SDD 6.5).
// Like the profile view, the page asks for it from the browser once the service is on screen.
type RecordServiceView struct {
	contacts search.ContactFinder
	record   *RecordMetric
	log      *slog.Logger
}

// NewRecordServiceView builds the command. A nil logger discards logs.
func NewRecordServiceView(contacts search.ContactFinder, record *RecordMetric, log *slog.Logger) *RecordServiceView {
	if log == nil {
		log = slog.New(slog.DiscardHandler)
	}
	return &RecordServiceView{contacts: contacts, record: record, log: log}
}

// Handle counts one view and answers nothing about it: a branch that is not public, a
// service that is inactive or belongs elsewhere, and a bot all look the same to the caller.
// A counter that fails is logged and swallowed; only a failure to look the branch up is an error.
func (r *RecordServiceView) Handle(ctx context.Context, branchID, serviceID int64, userAgent string) error {
	contact, err := r.contacts.ContactFor(ctx, branchID, serviceID)
	if errors.Is(err, search.ErrBranchNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	// The contact comes back without the service when it does not qualify.
	if contact.ServiceID == 0 {
		return nil
	}

	if _, err := r.record.Handle(ctx, search.Event{
		Metric:     search.ServiceView,
		BusinessID: contact.BusinessID,
		BranchID:   contact.BranchID,
		ServiceID:  contact.ServiceID,
	}, userAgent); err != nil {
		r.log.Warn("service view not counted", "branch_id", branchID, "service_id", serviceID, "error", err)
	}
	return nil
}
