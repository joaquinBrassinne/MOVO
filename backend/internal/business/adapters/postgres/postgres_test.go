package postgres_test

import (
	"errors"
	"strconv"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/joaquinBrassinne/MOVO/internal/business"
	"github.com/joaquinBrassinne/MOVO/internal/business/adapters/postgres"
	"github.com/joaquinBrassinne/MOVO/internal/business/domain"
	"github.com/joaquinBrassinne/MOVO/internal/platform/dbtest"
)

func newOwner(t *testing.T, pool *pgxpool.Pool, email string) int64 {
	t.Helper()

	var id int64
	err := pool.QueryRow(t.Context(),
		`INSERT INTO users (email, password_hash, role) VALUES ($1, 'hash', 'business_admin') RETURNING id`,
		email).Scan(&id)
	if err != nil {
		t.Fatalf("create owner: %v", err)
	}
	return id
}

func categoryID(t *testing.T, pool *pgxpool.Pool, slug string) int64 {
	t.Helper()

	var id int64
	if err := pool.QueryRow(t.Context(), `SELECT id FROM categories WHERE slug = $1`, slug).Scan(&id); err != nil {
		t.Fatalf("category %s: %v", slug, err)
	}
	return id
}

func draft(t *testing.T, owner, category int64, name string) domain.Business {
	t.Helper()

	b, err := domain.NewBusiness(domain.NewBusinessInput{
		OwnerID: owner, CategoryID: category, Name: name, Description: "desc", WhatsApp: "5491112345678",
	})
	if err != nil {
		t.Fatalf("NewBusiness: %v", err)
	}
	return b
}

func TestBusinessesCreateAndFindByOwner(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	repo := postgres.NewBusinesses(pool)
	owner := newOwner(t, pool, "owner@example.com")
	yoga := categoryID(t, pool, "yoga")

	created, err := repo.Create(t.Context(), draft(t, owner, yoga, "Yoga Center"))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.ID == 0 || created.CreatedAt.IsZero() {
		t.Fatalf("Create did not assign ID and CreatedAt: %+v", created)
	}

	got, err := repo.FindByOwner(t.Context(), owner)
	if err != nil {
		t.Fatalf("FindByOwner: %v", err)
	}
	if got.ID != created.ID || got.Slug != "yoga-center" || got.Status != domain.StatusDraft ||
		got.WhatsApp != "5491112345678" || got.Verified() || got.MPPayerEmail != "" || got.MPReturnParams != nil {
		t.Errorf("business = %+v, want the stored draft", got)
	}

	if _, err := repo.FindByOwner(t.Context(), newOwner(t, pool, "other@example.com")); !errors.Is(err, business.ErrBusinessNotFound) {
		t.Errorf("owner without business error = %v, want ErrBusinessNotFound", err)
	}
}

func TestBusinessesCreateMapsConstraintViolations(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	repo := postgres.NewBusinesses(pool)
	owner := newOwner(t, pool, "owner@example.com")
	other := newOwner(t, pool, "other@example.com")
	yoga := categoryID(t, pool, "yoga")

	if _, err := repo.Create(t.Context(), draft(t, owner, yoga, "Studio")); err != nil {
		t.Fatalf("first create: %v", err)
	}

	tests := []struct {
		name string
		b    domain.Business
		want error
	}{
		{"second business of the owner", draft(t, owner, yoga, "Another"), business.ErrOwnerHasBusiness},
		{"slug taken by someone else", draft(t, other, yoga, "Studio"), business.ErrSlugTaken},
		{"unknown category", draft(t, other, 9999, "Free Name"), business.ErrCategoryNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := repo.Create(t.Context(), tt.b); !errors.Is(err, tt.want) {
				t.Fatalf("error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestBusinessesSavePersistsProfileAndStatusButNotSlugOrOwner(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	repo := postgres.NewBusinesses(pool)
	owner := newOwner(t, pool, "owner@example.com")
	created, err := repo.Create(t.Context(), draft(t, owner, categoryID(t, pool, "yoga"), "Studio"))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	changed, err := created.UpdateProfile(domain.ProfileInput{
		CategoryID: categoryID(t, pool, "pilates"), Name: "Renamed", Description: "new", WhatsApp: "5491198765432",
	})
	if err != nil {
		t.Fatalf("UpdateProfile: %v", err)
	}
	changed, _ = changed.Submit()
	changed, _ = changed.DeclarePayer("ana@example.com", "Ana Pérez")
	changed = changed.RecordReturnParams(map[string]string{"plan": "pro"})
	changed.Slug = "tampered"
	changed.OwnerID = 12345

	if err := repo.Save(t.Context(), changed); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := repo.FindByOwner(t.Context(), owner)
	if err != nil {
		t.Fatalf("FindByOwner: %v", err)
	}
	if got.Name != "Renamed" || got.Description != "new" || got.WhatsApp != "5491198765432" ||
		got.Status != domain.StatusPendingReview || got.MPPayerEmail != "ana@example.com" || got.MPPayerName != "Ana Pérez" || got.PlanHint() != "pro" || got.CategoryID != changed.CategoryID {
		t.Errorf("business = %+v, want the saved fields", got)
	}
	if got.Slug != "studio" || got.OwnerID != owner {
		t.Errorf("slug/owner = %q/%d, they must never change", got.Slug, got.OwnerID)
	}

	missing := changed
	missing.ID = 424242
	if err := repo.Save(t.Context(), missing); !errors.Is(err, business.ErrBusinessNotFound) {
		t.Errorf("save of a missing business error = %v, want ErrBusinessNotFound", err)
	}
	changed.CategoryID = 9999
	if err := repo.Save(t.Context(), changed); !errors.Is(err, business.ErrCategoryNotFound) {
		t.Errorf("save with unknown category error = %v, want ErrCategoryNotFound", err)
	}
}

func setupBranches(t *testing.T) (*pgxpool.Pool, *postgres.Branches, int64, int64) {
	t.Helper()

	pool := dbtest.New(t)
	businesses := postgres.NewBusinesses(pool)
	yoga := categoryID(t, pool, "yoga")
	mine, err := businesses.Create(t.Context(), draft(t, newOwner(t, pool, "owner@example.com"), yoga, "Mine"))
	if err != nil {
		t.Fatalf("create business: %v", err)
	}
	theirs, err := businesses.Create(t.Context(), draft(t, newOwner(t, pool, "other@example.com"), yoga, "Theirs"))
	if err != nil {
		t.Fatalf("create business: %v", err)
	}
	return pool, postgres.NewBranches(pool), mine.ID, theirs.ID
}

func newBranch(t *testing.T, businessID int64, name string) domain.Branch {
	t.Helper()

	b, err := domain.NewBranch(domain.NewBranchInput{
		Name: name, Address: "Córdoba 1234", City: "Rosario", Latitude: -32.95, Longitude: -60.65,
		BookingURL: "https://example.com/reservas",
	})
	if err != nil {
		t.Fatalf("NewBranch: %v", err)
	}
	b.BusinessID = businessID
	return b
}

func TestBranchesCreateListFindAndSave(t *testing.T) {
	t.Parallel()

	_, repo, mine, theirs := setupBranches(t)

	first, err := repo.Create(t.Context(), newBranch(t, mine, "Centro"))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, err := repo.Create(t.Context(), newBranch(t, mine, "Norte")); err != nil {
		t.Fatalf("Create second: %v", err)
	}
	if _, err := repo.Create(t.Context(), newBranch(t, theirs, "Foreign")); err != nil {
		t.Fatalf("Create foreign: %v", err)
	}

	list, err := repo.List(t.Context(), mine)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 2 || list[0].Name != "Centro" || list[1].Name != "Norte" {
		t.Fatalf("list = %+v, want my two branches oldest first", list)
	}

	got, err := repo.Find(t.Context(), mine, first.ID)
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	if got.Latitude != -32.95 || got.Longitude != -60.65 || got.BookingURL != "https://example.com/reservas" || !got.Active {
		t.Errorf("branch = %+v, want the stored pin, booking url and active flag", got)
	}

	if _, err := repo.Find(t.Context(), theirs, first.ID); !errors.Is(err, business.ErrBranchNotFound) {
		t.Errorf("find through another business error = %v, want ErrBranchNotFound", err)
	}

	moved, err := got.Update(domain.NewBranchInput{Name: "Centro 2", Address: "Calle 9", City: "Funes", Latitude: -32.9, Longitude: -60.8})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if err := repo.Save(t.Context(), moved); err != nil {
		t.Fatalf("Save: %v", err)
	}
	after, _ := repo.Find(t.Context(), mine, first.ID)
	if after.Name != "Centro 2" || after.City != "Funes" || after.Latitude != -32.9 || after.BookingURL != "" {
		t.Errorf("after save = %+v", after)
	}

	moved.BusinessID = theirs
	if err := repo.Save(t.Context(), moved); !errors.Is(err, business.ErrBranchNotFound) {
		t.Errorf("save through another business error = %v, want ErrBranchNotFound", err)
	}
}

func TestBranchesHoursAreReplacedAtomically(t *testing.T) {
	t.Parallel()

	pool, repo, mine, theirs := setupBranches(t)
	branch, err := repo.Create(t.Context(), newBranch(t, mine, "Centro"))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	week := []domain.Slot{
		{Weekday: 3, Start: 9 * 60, End: 12 * 60},
		{Weekday: 1, Start: 16 * 60, End: 20 * 60},
		{Weekday: 1, Start: 9*60 + 30, End: 13 * 60},
	}
	if err := repo.ReplaceHours(t.Context(), mine, branch.ID, week); err != nil {
		t.Fatalf("ReplaceHours: %v", err)
	}
	got, err := repo.Hours(t.Context(), mine, branch.ID)
	if err != nil {
		t.Fatalf("Hours: %v", err)
	}
	want := []domain.Slot{
		{Weekday: 1, Start: 9*60 + 30, End: 13 * 60},
		{Weekday: 1, Start: 16 * 60, End: 20 * 60},
		{Weekday: 3, Start: 9 * 60, End: 12 * 60},
	}
	if len(got) != len(want) {
		t.Fatalf("hours = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("hours = %v, want %v ordered by weekday and start", got, want)
		}
	}

	// A batch the database rejects (the same slot twice) must roll back, keeping
	// the previous schedule instead of leaving the branch with none.
	bad := []domain.Slot{{Weekday: 2, Start: 600, End: 700}, {Weekday: 2, Start: 600, End: 700}}
	if err := repo.ReplaceHours(t.Context(), mine, branch.ID, bad); !errors.Is(err, domain.ErrSlotsOverlap) {
		t.Fatalf("ReplaceHours of a duplicated slot = %v, want ErrSlotsOverlap", err)
	}
	if kept, _ := repo.Hours(t.Context(), mine, branch.ID); len(kept) != 3 {
		t.Errorf("hours after a failed replace = %v, want the previous 3", kept)
	}

	if err := repo.ReplaceHours(t.Context(), theirs, branch.ID, week); !errors.Is(err, business.ErrBranchNotFound) {
		t.Errorf("replace through another business error = %v, want ErrBranchNotFound", err)
	}
	if _, err := repo.Hours(t.Context(), theirs, branch.ID); !errors.Is(err, business.ErrBranchNotFound) {
		t.Errorf("hours through another business error = %v, want ErrBranchNotFound", err)
	}

	if err := repo.ReplaceHours(t.Context(), mine, branch.ID, nil); err != nil {
		t.Fatalf("clear hours: %v", err)
	}
	var n int
	if err := pool.QueryRow(t.Context(), `SELECT count(*) FROM business_hours`).Scan(&n); err != nil || n != 0 {
		t.Errorf("rows after clearing = %d, %v; want 0", n, err)
	}
}

func TestBusinessesIsPilot(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	repo := postgres.NewBusinesses(pool)
	created, err := repo.Create(t.Context(), draft(t, newOwner(t, pool, "owner@example.com"), categoryID(t, pool, "yoga"), "Studio"))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if pilot, err := repo.IsPilot(t.Context(), created.ID); err != nil || pilot {
		t.Fatalf("new business IsPilot = %v, %v; want false", pilot, err)
	}
	if _, err := pool.Exec(t.Context(), `UPDATE businesses SET pilot = true WHERE id = $1`, created.ID); err != nil {
		t.Fatalf("mark pilot: %v", err)
	}
	if pilot, err := repo.IsPilot(t.Context(), created.ID); err != nil || !pilot {
		t.Errorf("pilot business IsPilot = %v, %v; want true", pilot, err)
	}
	if _, err := repo.IsPilot(t.Context(), 424242); !errors.Is(err, business.ErrBusinessNotFound) {
		t.Errorf("missing business error = %v, want ErrBusinessNotFound", err)
	}
}

func TestCategoriesListTheSeededCatalogInOrder(t *testing.T) {
	t.Parallel()

	got, err := postgres.NewCategories(dbtest.New(t)).List(t.Context())
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	want := []string{"gimnasios", "pilates", "yoga", "entrenadores", "estetica", "natacion", "padel", "trekking-outdoor"}
	if len(got) != len(want) {
		t.Fatalf("categories = %+v, want %d", got, len(want))
	}
	for i, slug := range want {
		if got[i].Slug != slug || got[i].ID == 0 || got[i].Name == "" {
			t.Errorf("category %d = %+v, want slug %q", i, got[i], slug)
		}
	}
	if got[6].Name != "Pádel" || got[4].Name != "Estética" || got[7].Name != "Trekking y outdoor" {
		t.Errorf("names = %q, %q, %q; want the accented Spanish names", got[6].Name, got[4].Name, got[7].Name)
	}
}

func itoa(n int64) string { return strconv.FormatInt(n, 10) }

func TestBranchesStoreTheirReference(t *testing.T) {
	t.Parallel()

	_, repo, mine, _ := setupBranches(t)
	b := newBranch(t, mine, "Centro")
	b.Reference = "PB, timbre 4B"
	created, err := repo.Create(t.Context(), b)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if got, _ := repo.Find(t.Context(), mine, created.ID); got.Reference != "PB, timbre 4B" {
		t.Errorf("reference = %q", got.Reference)
	}

	created.Reference = ""
	if err := repo.Save(t.Context(), created); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if got, _ := repo.Find(t.Context(), mine, created.ID); got.Reference != "" {
		t.Errorf("reference after clearing = %q", got.Reference)
	}
}
