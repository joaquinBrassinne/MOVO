package postgres_test

import (
	"testing"

	"github.com/joaquinBrassinne/MOVO/internal/platform/dbtest"
	"github.com/joaquinBrassinne/MOVO/internal/search"
	"github.com/joaquinBrassinne/MOVO/internal/search/adapters/postgres"
)

func coversByBranch(branches []search.Branch) map[int64]string {
	out := map[int64]string{}
	for _, b := range branches {
		out[b.BranchID] = b.CoverKey
	}
	return out
}

func TestCardsCarryTheKeyOfTheCoverPhotoOfTheBusiness(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	featuredAsPilot(t, pool)
	s := &seeded{pool: pool}
	repo := postgres.NewBranches(pool)

	withPhotos := s.add(t, "yoga", "active", 100, true)
	business := businessOf(t, pool, withPhotos)
	for position, key := range []string{"businesses/1/second.jpg", "businesses/1/cover.jpg"} {
		// The cover is the photo in the first position, whatever order they were inserted in.
		if _, err := pool.Exec(t.Context(),
			`INSERT INTO business_images (business_id, object_key, position) VALUES ($1, $2, $3)`,
			business, key, 1-position); err != nil {
			t.Fatalf("insert image: %v", err)
		}
	}
	withoutPhotos := s.add(t, "yoga", "active", 200, true)

	query := search.Query{Lat: originLat, Lng: originLng, RadiusM: 20_000, Limit: 50}
	nearby, err := repo.Nearby(t.Context(), query)
	if err != nil {
		t.Fatalf("Nearby: %v", err)
	}
	for name, got := range map[string]map[int64]string{
		"nearby":   coversByBranch(nearby),
		"featured": coversByBranch(featured(t, repo, search.Query{}, 1)),
	} {
		if len(got) != 2 || got[withPhotos] != "businesses/1/cover.jpg" || got[withoutPhotos] != "" {
			t.Errorf("%s covers = %v, want the first photo for the first branch and none for the second, one row each", name, got)
		}
	}
}
