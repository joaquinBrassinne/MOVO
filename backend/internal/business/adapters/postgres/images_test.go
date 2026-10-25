package postgres_test

import (
	"errors"
	"slices"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/joaquinBrassinne/MOVO/internal/business"
	"github.com/joaquinBrassinne/MOVO/internal/business/adapters/postgres"
	"github.com/joaquinBrassinne/MOVO/internal/business/domain"
)

func setupImages(t *testing.T) (*pgxpool.Pool, *postgres.Images, int64, int64) {
	t.Helper()

	pool, _, mine, theirs := setupBranches(t)
	return pool, postgres.NewImages(pool), mine, theirs
}

func keys(images []domain.Image) []string {
	out := make([]string, 0, len(images))
	for _, img := range images {
		out = append(out, img.ObjectKey)
	}
	return out
}

func TestImagesAddAppendsInOrderAndListsThem(t *testing.T) {
	t.Parallel()

	_, repo, mine, theirs := setupImages(t)

	for i, key := range []string{"businesses/1/a.jpg", "businesses/1/b.jpg", "businesses/1/c.jpg"} {
		got, err := repo.Add(t.Context(), mine, key, 6)
		if err != nil || got.Position != i || got.ID == 0 || got.BusinessID != mine {
			t.Fatalf("Add %q = %+v, %v; want position %d", key, got, err, i)
		}
	}
	if _, err := repo.Add(t.Context(), theirs, "businesses/2/z.jpg", 6); err != nil {
		t.Fatalf("Add foreign: %v", err)
	}

	list, err := repo.List(t.Context(), mine)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if want := []string{"businesses/1/a.jpg", "businesses/1/b.jpg", "businesses/1/c.jpg"}; !slices.Equal(keys(list), want) {
		t.Errorf("list = %v, want %v", keys(list), want)
	}
	if empty, _ := repo.List(t.Context(), 999999); empty == nil || len(empty) != 0 {
		t.Errorf("list of a business without photos = %#v, want a non-nil empty list", empty)
	}
}

func TestImagesAddEnforcesTheLimitAndUniqueKeys(t *testing.T) {
	t.Parallel()

	_, repo, mine, theirs := setupImages(t)

	for _, key := range []string{"k/1.jpg", "k/2.jpg"} {
		if _, err := repo.Add(t.Context(), mine, key, 2); err != nil {
			t.Fatalf("Add %s: %v", key, err)
		}
	}
	if _, err := repo.Add(t.Context(), mine, "k/3.jpg", 2); !errors.Is(err, business.ErrImageLimitReached) {
		t.Errorf("over the limit error = %v, want ErrImageLimitReached", err)
	}
	if _, err := repo.Add(t.Context(), theirs, "k/1.jpg", 2); !errors.Is(err, business.ErrImageExists) {
		t.Errorf("a key registered by anyone error = %v, want ErrImageExists", err)
	}
	if list, _ := repo.List(t.Context(), mine); len(list) != 2 {
		t.Errorf("photos = %d, a refused one must not be stored", len(list))
	}
}

// TestImagesAddNeverExceedsTheLimitUnderConcurrency pins that two uploads racing
// for the last place cannot both win.
func TestImagesAddNeverExceedsTheLimitUnderConcurrency(t *testing.T) {
	t.Parallel()

	_, repo, mine, _ := setupImages(t)

	const attempts, limit = 12, 4
	var wg sync.WaitGroup
	results := make(chan error, attempts)
	for i := range attempts {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := repo.Add(t.Context(), mine, "race/"+itoa(int64(i))+".jpg", limit)
			results <- err
		}()
	}
	wg.Wait()
	close(results)

	var accepted int
	for err := range results {
		switch {
		case err == nil:
			accepted++
		case !errors.Is(err, business.ErrImageLimitReached):
			t.Errorf("unexpected error: %v", err)
		}
	}
	list, _ := repo.List(t.Context(), mine)
	if accepted != limit || len(list) != limit {
		t.Fatalf("accepted %d, stored %d; want exactly %d", accepted, len(list), limit)
	}
	for i, img := range list {
		if img.Position != i {
			t.Errorf("position %d = %d, positions must be contiguous", i, img.Position)
		}
	}
}

func TestImagesRemoveClosesTheGapAndIsScopedByBusiness(t *testing.T) {
	t.Parallel()

	_, repo, mine, theirs := setupImages(t)
	var ids []int64
	for _, key := range []string{"r/a.jpg", "r/b.jpg", "r/c.jpg"} {
		img, err := repo.Add(t.Context(), mine, key, 6)
		if err != nil {
			t.Fatalf("Add: %v", err)
		}
		ids = append(ids, img.ID)
	}

	if _, err := repo.Remove(t.Context(), theirs, ids[0]); !errors.Is(err, business.ErrImageNotFound) {
		t.Fatalf("removal through another business error = %v, want ErrImageNotFound", err)
	}

	removed, err := repo.Remove(t.Context(), mine, ids[0])
	if err != nil || removed.ObjectKey != "r/a.jpg" {
		t.Fatalf("Remove = %+v, %v; want the removed photo back so its object can be deleted", removed, err)
	}
	list, _ := repo.List(t.Context(), mine)
	if len(list) != 2 || list[0].ObjectKey != "r/b.jpg" || list[0].Position != 0 || list[1].Position != 1 {
		t.Errorf("list = %+v, want b as the cover and no gap", list)
	}
	if _, err := repo.Remove(t.Context(), mine, ids[0]); !errors.Is(err, business.ErrImageNotFound) {
		t.Errorf("second removal error = %v, want ErrImageNotFound", err)
	}
}

func TestImagesReorderSwapsPositionsAtomically(t *testing.T) {
	t.Parallel()

	_, repo, mine, theirs := setupImages(t)
	var ids []int64
	for _, key := range []string{"o/a.jpg", "o/b.jpg", "o/c.jpg"} {
		img, err := repo.Add(t.Context(), mine, key, 6)
		if err != nil {
			t.Fatalf("Add: %v", err)
		}
		ids = append(ids, img.ID)
	}
	foreign, _ := repo.Add(t.Context(), theirs, "o/z.jpg", 6)

	if err := repo.Reorder(t.Context(), mine, []int64{ids[2], ids[0], ids[1]}); err != nil {
		t.Fatalf("Reorder: %v", err)
	}
	list, _ := repo.List(t.Context(), mine)
	if want := []string{"o/c.jpg", "o/a.jpg", "o/b.jpg"}; !slices.Equal(keys(list), want) {
		t.Fatalf("list = %v, want %v", keys(list), want)
	}

	for name, order := range map[string][]int64{
		"missing one":      {ids[0], ids[1]},
		"duplicate":        {ids[0], ids[0], ids[1]},
		"unknown":          {ids[0], ids[1], 999999},
		"photo of another": {ids[0], ids[1], foreign.ID},
		"empty":            {},
		"one too many":     {ids[0], ids[1], ids[2], foreign.ID},
	} {
		if err := repo.Reorder(t.Context(), mine, order); !errors.Is(err, business.ErrImageOrderInvalid) {
			t.Errorf("%s error = %v, want ErrImageOrderInvalid", name, err)
		}
	}
	if after, _ := repo.List(t.Context(), mine); !slices.Equal(keys(after), keys(list)) {
		t.Errorf("a rejected order changed the list to %v", keys(after))
	}
}
