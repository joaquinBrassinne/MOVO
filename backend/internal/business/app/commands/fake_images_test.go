package commands_test

import (
	"context"
	"sync"

	"github.com/joaquinBrassinne/MOVO/internal/business"
	"github.com/joaquinBrassinne/MOVO/internal/business/domain"
)

// fakeImages is an in-memory ImageRepository with the same rules as the database:
// contiguous positions, unique keys and a per-business cap.
type fakeImages struct {
	mu     sync.Mutex
	nextID int64
	list   []domain.Image // every business, in insertion order
}

func (f *fakeImages) forBusiness(businessID int64) []domain.Image {
	var out []domain.Image
	for _, img := range f.list {
		if img.BusinessID == businessID {
			out = append(out, img)
		}
	}
	return out
}

func (f *fakeImages) Add(_ context.Context, businessID int64, key string, max int) (domain.Image, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	for _, img := range f.list {
		if img.ObjectKey == key {
			return domain.Image{}, business.ErrImageExists
		}
	}
	if len(f.forBusiness(businessID)) >= max {
		return domain.Image{}, business.ErrImageLimitReached
	}
	f.nextID++
	img := domain.Image{ID: f.nextID, BusinessID: businessID, ObjectKey: key, Position: len(f.forBusiness(businessID))}
	f.list = append(f.list, img)
	return img, nil
}

func (f *fakeImages) List(_ context.Context, businessID int64) ([]domain.Image, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.forBusiness(businessID), nil
}

func (f *fakeImages) Remove(_ context.Context, businessID, imageID int64) (domain.Image, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	for i, img := range f.list {
		if img.ID == imageID && img.BusinessID == businessID {
			f.list = append(f.list[:i:i], f.list[i+1:]...)
			for j := range f.list {
				if f.list[j].BusinessID == businessID && f.list[j].Position > img.Position {
					f.list[j].Position--
				}
			}
			return img, nil
		}
	}
	return domain.Image{}, business.ErrImageNotFound
}

func (f *fakeImages) Reorder(_ context.Context, businessID int64, ids []int64) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	mine := f.forBusiness(businessID)
	if len(ids) != len(mine) {
		return business.ErrImageOrderInvalid
	}
	position := map[int64]int{}
	for i, id := range ids {
		position[id] = i
	}
	if len(position) != len(ids) {
		return business.ErrImageOrderInvalid
	}
	for _, img := range mine {
		if _, ok := position[img.ID]; !ok {
			return business.ErrImageOrderInvalid
		}
	}
	for j := range f.list {
		if f.list[j].BusinessID == businessID {
			f.list[j].Position = position[f.list[j].ID]
		}
	}
	return nil
}
