package search

import (
	"context"
	"errors"
	"time"
)

// Metric is a kind of daily counter. They count events, never visitors (SDD 4.3).
type Metric string

// The counters recorded from phase 3 on, so history exists when the panel does.
const (
	ProfileView   Metric = "profile_view"
	ServiceView   Metric = "service_view"
	WhatsAppClick Metric = "whatsapp_click"
)

// ErrInvalidEvent is returned when an event does not fit the shape of its metric.
var ErrInvalidEvent = errors.New("search: invalid metric event")

// Event is one thing that happened. It names where, and nothing about who: there is
// no field that could hold an address, an agent or an identifier. A zero BranchID or
// ServiceID means none.
type Event struct {
	Metric     Metric
	BusinessID int64
	BranchID   int64
	ServiceID  int64
}

// Validate reports ErrInvalidEvent unless the event has the shape its metric needs:
// a profile view belongs to the business alone, a click needs its branch, and a
// service view needs both its branch and its service.
func (e Event) Validate() error {
	if e.BusinessID <= 0 || e.BranchID < 0 || e.ServiceID < 0 {
		return ErrInvalidEvent
	}
	switch e.Metric {
	case ProfileView:
		if e.BranchID != 0 || e.ServiceID != 0 {
			return ErrInvalidEvent
		}
	case WhatsAppClick:
		if e.BranchID == 0 {
			return ErrInvalidEvent
		}
	case ServiceView:
		if e.BranchID == 0 || e.ServiceID == 0 {
			return ErrInvalidEvent
		}
	default:
		return ErrInvalidEvent
	}
	return nil
}

// MetricCounter adds one to the counter of an event on a day.
type MetricCounter interface {
	Increment(ctx context.Context, e Event, day time.Time) error
}
