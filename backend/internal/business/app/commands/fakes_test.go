package commands_test

import (
	"context"
	"sync"

	"github.com/joaquinBrassinne/MOVO/internal/business"
	"github.com/joaquinBrassinne/MOVO/internal/business/domain"
)

// fakeBusinesses is an in-memory BusinessRepository that enforces the same
// uniqueness rules as the database.
type fakeBusinesses struct {
	mu         sync.Mutex
	nextID     int64
	byID       map[int64]domain.Business
	categories map[int64]bool
	saves      int
}

func newFakeBusinesses(categories ...int64) *fakeBusinesses {
	f := &fakeBusinesses{byID: map[int64]domain.Business{}, categories: map[int64]bool{}}
	for _, id := range categories {
		f.categories[id] = true
	}
	return f
}

func (f *fakeBusinesses) Create(_ context.Context, b domain.Business) (domain.Business, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if !f.categories[b.CategoryID] {
		return domain.Business{}, business.ErrCategoryNotFound
	}
	for _, other := range f.byID {
		if other.OwnerID == b.OwnerID {
			return domain.Business{}, business.ErrOwnerHasBusiness
		}
		if other.Slug == b.Slug {
			return domain.Business{}, business.ErrSlugTaken
		}
	}
	f.nextID++
	b.ID = f.nextID
	f.byID[b.ID] = b
	return b, nil
}

func (f *fakeBusinesses) FindByOwner(_ context.Context, ownerID int64) (domain.Business, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	for _, b := range f.byID {
		if b.OwnerID == ownerID {
			return b, nil
		}
	}
	return domain.Business{}, business.ErrBusinessNotFound
}

func (f *fakeBusinesses) Save(_ context.Context, b domain.Business) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if _, ok := f.byID[b.ID]; !ok {
		return business.ErrBusinessNotFound
	}
	if !f.categories[b.CategoryID] {
		return business.ErrCategoryNotFound
	}
	f.saves++
	f.byID[b.ID] = b
	return nil
}

// limits is a business.Entitlements with fixed limits; 0 means unlimited.
type limits struct{ branches, services int }

func (l limits) CanAddBranch(current int) bool { return l.branches == 0 || current < l.branches }

func (l limits) CanActivateService(current int) bool { return l.services == 0 || current < l.services }

// fixedEntitlements is a business.EntitlementsProvider that gives every business
// the same limits.
type fixedEntitlements struct{ limits limits }

func (f fixedEntitlements) For(context.Context, int64) (business.Entitlements, error) {
	return f.limits, nil
}

func unlimited() fixedEntitlements { return fixedEntitlements{} }
