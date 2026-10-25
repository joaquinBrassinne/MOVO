package commands_test

import (
	"context"
	"sort"
	"sync"

	"github.com/joaquinBrassinne/MOVO/internal/business"
	"github.com/joaquinBrassinne/MOVO/internal/business/domain"
)

// fakeBranches is an in-memory BranchRepository scoped by business id like the
// real one.
type fakeBranches struct {
	mu     sync.Mutex
	nextID int64
	byID   map[int64]domain.Branch
	hours  map[int64][]domain.Slot
}

func newFakeBranches() *fakeBranches {
	return &fakeBranches{byID: map[int64]domain.Branch{}, hours: map[int64][]domain.Slot{}}
}

func (f *fakeBranches) Create(_ context.Context, b domain.Branch) (domain.Branch, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.nextID++
	b.ID = f.nextID
	f.byID[b.ID] = b
	return b, nil
}

func (f *fakeBranches) List(_ context.Context, businessID int64) ([]domain.Branch, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	var out []domain.Branch
	for _, b := range f.byID {
		if b.BusinessID == businessID {
			out = append(out, b)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (f *fakeBranches) Find(_ context.Context, businessID, branchID int64) (domain.Branch, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	b, ok := f.byID[branchID]
	if !ok || b.BusinessID != businessID {
		return domain.Branch{}, business.ErrBranchNotFound
	}
	return b, nil
}

func (f *fakeBranches) Save(_ context.Context, b domain.Branch) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	current, ok := f.byID[b.ID]
	if !ok || current.BusinessID != b.BusinessID {
		return business.ErrBranchNotFound
	}
	f.byID[b.ID] = b
	return nil
}

func (f *fakeBranches) Hours(_ context.Context, businessID, branchID int64) ([]domain.Slot, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	b, ok := f.byID[branchID]
	if !ok || b.BusinessID != businessID {
		return nil, business.ErrBranchNotFound
	}
	return append([]domain.Slot(nil), f.hours[branchID]...), nil
}

func (f *fakeBranches) ReplaceHours(_ context.Context, businessID, branchID int64, slots []domain.Slot) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	b, ok := f.byID[branchID]
	if !ok || b.BusinessID != businessID {
		return business.ErrBranchNotFound
	}
	f.hours[branchID] = append([]domain.Slot(nil), slots...)
	return nil
}
