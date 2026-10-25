package queries

import (
	"context"
	"fmt"

	"github.com/joaquinBrassinne/MOVO/internal/business"
	"github.com/joaquinBrassinne/MOVO/internal/business/domain"
)

// ServiceView is a service with its weekly schedule.
type ServiceView struct {
	Service  domain.Service
	Schedule []domain.Slot
}

// ListMyServices reads the services of the owner's business, active or not.
type ListMyServices struct {
	businesses business.BusinessRepository
	services   business.ServiceRepository
}

// NewListMyServices wires the ListMyServices query.
func NewListMyServices(businesses business.BusinessRepository, services business.ServiceRepository) *ListMyServices {
	return &ListMyServices{businesses: businesses, services: services}
}

// Handle returns business.ErrBusinessNotFound when the owner has no business. The
// list is never nil so JSON renders [].
func (q *ListMyServices) Handle(ctx context.Context, ownerID int64) ([]ServiceView, error) {
	b, err := q.businesses.FindByOwner(ctx, ownerID)
	if err != nil {
		return nil, fmt.Errorf("find business: %w", err)
	}
	services, err := q.services.List(ctx, b.ID)
	if err != nil {
		return nil, fmt.Errorf("list services: %w", err)
	}

	views := make([]ServiceView, 0, len(services))
	for _, s := range services {
		schedule, err := q.services.Schedule(ctx, b.ID, s.ID)
		if err != nil {
			return nil, fmt.Errorf("read schedule of service %d: %w", s.ID, err)
		}
		views = append(views, ServiceView{Service: s, Schedule: schedule})
	}
	return views, nil
}
