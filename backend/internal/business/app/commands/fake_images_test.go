package commands_test

import (
	"context"
	"errors"
	"slices"
	"sync"

	"github.com/joaquinBrassinne/MOVO/internal/business"
	"github.com/joaquinBrassinne/MOVO/internal/business/domain"
	"github.com/joaquinBrassinne/MOVO/internal/platform/storage"
)

// fakeImages is an in-memory ImageRepository with the same rules as the database:
// contiguous positions, unique keys and a per-business cap.
type fakeImages struct {
	mu     sync.Mutex
	nextID int64
	list   []domain.Image // every business, in insertion order
	// cleanup, when set, receives the key of every removed photo, as the real repository does.
	cleanup *fakeCleanup
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
			if f.cleanup != nil {
				f.cleanup.queue(img.ObjectKey)
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

// fakeCleanup is the queue of objects waiting to be deleted, in memory.
type fakeCleanup struct {
	mu       sync.Mutex
	attempts map[string]int
	reasons  map[string]string
}

func (c *fakeCleanup) queue(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.attempts == nil {
		c.attempts, c.reasons = map[string]int{}, map[string]string{}
	}
	if _, queued := c.attempts[key]; !queued {
		c.attempts[key] = 0
	}
}

func (c *fakeCleanup) Pending(_ context.Context, limit int) ([]string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	keys := make([]string, 0, len(c.attempts))
	for key := range c.attempts {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys[:min(limit, len(keys))], nil
}

func (c *fakeCleanup) Done(_ context.Context, key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.attempts, key)
	delete(c.reasons, key)
	return nil
}

func (c *fakeCleanup) Failed(_ context.Context, key, reason string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.attempts[key]++
	c.reasons[key] = reason
	return nil
}

// flakyStorage is a storage whose deletes fail while down is set.
type flakyStorage struct {
	*storage.Memory
	down bool
}

func (s *flakyStorage) Delete(ctx context.Context, key string) error {
	if s.down {
		return errors.New("storage unavailable")
	}
	return s.Memory.Delete(ctx, key)
}
