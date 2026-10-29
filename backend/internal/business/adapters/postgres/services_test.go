package postgres_test

import (
	"errors"
	"testing"

	"github.com/joaquinBrassinne/MOVO/internal/business"
	"github.com/joaquinBrassinne/MOVO/internal/business/adapters/postgres"
	"github.com/joaquinBrassinne/MOVO/internal/business/domain"
)

func newService(t *testing.T, businessID, branchID int64, name string) domain.Service {
	t.Helper()

	s, err := domain.NewService(domain.ServiceInput{Name: name, Description: "desc", BranchID: branchID})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	s.BusinessID = businessID
	return s
}

func TestServicesCreateListFindAndSave(t *testing.T) {
	t.Parallel()

	pool, branches, mine, theirs := setupBranches(t)
	repo := postgres.NewServices(pool)
	branch, err := branches.Create(t.Context(), newBranch(t, mine, "Centro"))
	if err != nil {
		t.Fatalf("create branch: %v", err)
	}

	first, err := repo.Create(t.Context(), newService(t, mine, branch.ID, "Yoga"))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, err := repo.Create(t.Context(), newService(t, mine, 0, "Pilates")); err != nil {
		t.Fatalf("Create without branch: %v", err)
	}
	if _, err := repo.Create(t.Context(), newService(t, theirs, 0, "Foreign")); err != nil {
		t.Fatalf("Create foreign: %v", err)
	}

	list, err := repo.List(t.Context(), mine)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 2 || list[0].Name != "Yoga" || list[1].Name != "Pilates" || list[1].BranchID != 0 {
		t.Fatalf("list = %+v, want my two services oldest first", list)
	}

	got, err := repo.Find(t.Context(), mine, first.ID)
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	if got.BranchID != branch.ID || got.Description != "desc" || !got.Active {
		t.Errorf("service = %+v", got)
	}
	if _, err := repo.Find(t.Context(), theirs, first.ID); !errors.Is(err, business.ErrServiceNotFound) {
		t.Errorf("find through another business error = %v, want ErrServiceNotFound", err)
	}

	// Save writes the details and leaves the flag to SetActive, even when the value
	// it is given disagrees: an edit must not undo a concurrent on or off.
	changed := got.Deactivate()
	changed.Name, changed.BranchID = "Yoga Flow", 0
	if err := repo.Save(t.Context(), changed); err != nil {
		t.Fatalf("Save: %v", err)
	}
	after, _ := repo.Find(t.Context(), mine, first.ID)
	if after.Name != "Yoga Flow" || !after.Active || after.BranchID != 0 {
		t.Errorf("after save = %+v, want the new details and the flag untouched", after)
	}
	if off, err := repo.SetActive(t.Context(), mine, first.ID, false, nil); err != nil || off.Active {
		t.Errorf("SetActive(false) = %+v, %v, want the service off", off, err)
	}

	changed.BusinessID = theirs
	if err := repo.Save(t.Context(), changed); !errors.Is(err, business.ErrServiceNotFound) {
		t.Errorf("save through another business error = %v, want ErrServiceNotFound", err)
	}
}

func TestServicesRefuseABranchOfAnotherBusiness(t *testing.T) {
	t.Parallel()

	pool, branches, mine, theirs := setupBranches(t)
	repo := postgres.NewServices(pool)
	theirBranch, err := branches.Create(t.Context(), newBranch(t, theirs, "Foreign"))
	if err != nil {
		t.Fatalf("create branch: %v", err)
	}

	if _, err := repo.Create(t.Context(), newService(t, mine, theirBranch.ID, "Hijack")); !errors.Is(err, business.ErrBranchNotFound) {
		t.Fatalf("error = %v, want ErrBranchNotFound from the composite foreign key", err)
	}
}

func TestServicesScheduleIsReplacedAtomically(t *testing.T) {
	t.Parallel()

	pool, _, mine, theirs := setupBranches(t)
	repo := postgres.NewServices(pool)
	service, err := repo.Create(t.Context(), newService(t, mine, 0, "Yoga"))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	week := []domain.Slot{
		{Weekday: 3, Start: 9 * 60, End: 10 * 60},
		{Weekday: 1, Start: 18 * 60, End: 19 * 60},
		{Weekday: 1, Start: 9 * 60, End: 10 * 60},
	}
	if err := repo.ReplaceSchedule(t.Context(), mine, service.ID, week); err != nil {
		t.Fatalf("ReplaceSchedule: %v", err)
	}
	want := []domain.Slot{week[2], week[1], week[0]}
	got, err := repo.Schedule(t.Context(), mine, service.ID)
	if err != nil || len(got) != 3 {
		t.Fatalf("Schedule = %v, %v", got, err)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("schedule = %v, want %v ordered by weekday and start", got, want)
		}
	}

	bad := []domain.Slot{{Weekday: 2, Start: 600, End: 660}, {Weekday: 2, Start: 600, End: 660}}
	if err := repo.ReplaceSchedule(t.Context(), mine, service.ID, bad); err == nil {
		t.Fatal("ReplaceSchedule accepted a duplicated slot")
	}
	if kept, _ := repo.Schedule(t.Context(), mine, service.ID); len(kept) != 3 {
		t.Errorf("schedule after a failed replace = %v, want the previous 3", kept)
	}

	if err := repo.ReplaceSchedule(t.Context(), theirs, service.ID, week); !errors.Is(err, business.ErrServiceNotFound) {
		t.Errorf("replace through another business error = %v, want ErrServiceNotFound", err)
	}
	if _, err := repo.Schedule(t.Context(), theirs, service.ID); !errors.Is(err, business.ErrServiceNotFound) {
		t.Errorf("schedule through another business error = %v, want ErrServiceNotFound", err)
	}
}

func TestServicesDeleteRemovesTheScheduleAndIsScopedByBusiness(t *testing.T) {
	t.Parallel()

	pool, _, mine, theirs := setupBranches(t)
	repo := postgres.NewServices(pool)
	service, err := repo.Create(t.Context(), newService(t, mine, 0, "Yoga"))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := repo.ReplaceSchedule(t.Context(), mine, service.ID, []domain.Slot{{Weekday: 1, Start: 540, End: 600}}); err != nil {
		t.Fatalf("ReplaceSchedule: %v", err)
	}

	if err := repo.Delete(t.Context(), theirs, service.ID); !errors.Is(err, business.ErrServiceNotFound) {
		t.Fatalf("delete through another business error = %v, want ErrServiceNotFound", err)
	}
	if err := repo.Delete(t.Context(), mine, service.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if err := repo.Delete(t.Context(), mine, service.ID); !errors.Is(err, business.ErrServiceNotFound) {
		t.Errorf("second delete error = %v, want ErrServiceNotFound", err)
	}
	var n int
	if err := pool.QueryRow(t.Context(), `SELECT count(*) FROM service_schedules`).Scan(&n); err != nil || n != 0 {
		t.Errorf("schedules left after deleting the service = %d, %v; want 0", n, err)
	}
}

func TestServicesStoreAnOptionalPrice(t *testing.T) {
	t.Parallel()

	pool, _, mine, _ := setupBranches(t)
	repo := postgres.NewServices(pool)

	priced := newService(t, mine, 0, "Yoga")
	priced.Price = 5500
	created, err := repo.Create(t.Context(), priced)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if got, _ := repo.Find(t.Context(), mine, created.ID); got.Price != 5500 {
		t.Errorf("price = %d, want 5500", got.Price)
	}

	cleared := created
	cleared.Price = 0
	if err := repo.Save(t.Context(), cleared); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if got, _ := repo.Find(t.Context(), mine, created.ID); got.Price != 0 {
		t.Errorf("price after clearing = %d, want 0", got.Price)
	}
	var stored *int
	if err := pool.QueryRow(t.Context(), `SELECT price FROM services WHERE id = $1`, created.ID).Scan(&stored); err != nil || stored != nil {
		t.Errorf("stored price = %v, %v; want NULL and not zero", stored, err)
	}
}
