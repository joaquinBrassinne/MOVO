package commands_test

import (
	"context"
	"sort"
	"sync"

	"github.com/joaquinBrassinne/MOVO/internal/business"
	"github.com/joaquinBrassinne/MOVO/internal/business/domain"
)

// fakeServices is an in-memory ServiceRepository scoped by business id like the
// real one.
type fakeServices struct {
	mu        sync.Mutex
	nextID    int64
	byID      map[int64]domain.Service
	schedules map[int64][]domain.Slot
}

func newFakeServices() *fakeServices {
	return &fakeServices{byID: map[int64]domain.Service{}, schedules: map[int64][]domain.Slot{}}
}

func (f *fakeServices) Create(_ context.Context, s domain.Service) (domain.Service, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.nextID++
	s.ID = f.nextID
	f.byID[s.ID] = s
	return s, nil
}

func (f *fakeServices) List(_ context.Context, businessID int64) ([]domain.Service, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	var out []domain.Service
	for _, s := range f.byID {
		if s.BusinessID == businessID {
			out = append(out, s)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (f *fakeServices) owned(businessID, serviceID int64) (domain.Service, bool) {
	s, ok := f.byID[serviceID]
	return s, ok && s.BusinessID == businessID
}

func (f *fakeServices) Find(_ context.Context, businessID, serviceID int64) (domain.Service, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	s, ok := f.owned(businessID, serviceID)
	if !ok {
		return domain.Service{}, business.ErrServiceNotFound
	}
	return s, nil
}

func (f *fakeServices) Save(_ context.Context, s domain.Service) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if _, ok := f.owned(s.BusinessID, s.ID); !ok {
		return business.ErrServiceNotFound
	}
	f.byID[s.ID] = s
	return nil
}

func (f *fakeServices) Delete(_ context.Context, businessID, serviceID int64) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if _, ok := f.owned(businessID, serviceID); !ok {
		return business.ErrServiceNotFound
	}
	delete(f.byID, serviceID)
	delete(f.schedules, serviceID)
	return nil
}

func (f *fakeServices) Schedule(_ context.Context, businessID, serviceID int64) ([]domain.Slot, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if _, ok := f.owned(businessID, serviceID); !ok {
		return nil, business.ErrServiceNotFound
	}
	return append([]domain.Slot(nil), f.schedules[serviceID]...), nil
}

func (f *fakeServices) ReplaceSchedule(_ context.Context, businessID, serviceID int64, slots []domain.Slot) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if _, ok := f.owned(businessID, serviceID); !ok {
		return business.ErrServiceNotFound
	}
	f.schedules[serviceID] = append([]domain.Slot(nil), slots...)
	return nil
}
