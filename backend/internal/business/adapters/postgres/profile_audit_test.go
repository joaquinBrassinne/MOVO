package postgres_test

import (
	"errors"
	"testing"

	"github.com/joaquinBrassinne/MOVO/internal/business"
	"github.com/joaquinBrassinne/MOVO/internal/business/adapters/postgres"
	"github.com/joaquinBrassinne/MOVO/internal/business/domain"
	"github.com/joaquinBrassinne/MOVO/internal/platform/dbtest"
)

func profileEntry(owner, businessID int64) business.AuditEntry {
	return business.AuditEntry{
		ActorID: owner, Action: "business.profile_update", Entity: "business", EntityID: businessID,
		Metadata: map[string]any{"changes": map[string]any{"name": map[string]any{"from": "Yoga Center", "to": "Yoga Renamed"}}},
	}
}

func TestSaveProfileStoresTheProfileAndItsAuditEntryTogether(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	repo := postgres.NewBusinesses(pool)
	owner := newOwner(t, pool, "owner@example.com")
	created, err := repo.Create(t.Context(), draft(t, owner, categoryID(t, pool, "yoga"), "Yoga Center"))
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	updated, err := created.UpdateProfile(profileInput(created, "Yoga Renamed", "Una descripción nueva", "5491198765432", categoryID(t, pool, "pilates")))
	if err != nil {
		t.Fatalf("UpdateProfile: %v", err)
	}
	if err := repo.SaveProfile(t.Context(), updated, profileEntry(owner, created.ID)); err != nil {
		t.Fatalf("SaveProfile: %v", err)
	}

	stored, err := repo.FindByOwner(t.Context(), owner)
	if err != nil || stored.Name != "Yoga Renamed" || stored.Description != "Una descripción nueva" || stored.WhatsApp != "5491198765432" || stored.Slug != created.Slug {
		t.Fatalf("stored = %+v, %v; want the new profile and the same slug", stored, err)
	}
	var action, name string
	var actor int64
	if err := pool.QueryRow(t.Context(),
		`SELECT action, actor_id, metadata->'changes'->'name'->>'to' FROM audit_log WHERE entity = 'business' AND entity_id = $1`,
		created.ID).Scan(&action, &actor, &name); err != nil || action != "business.profile_update" || actor != owner || name != "Yoga Renamed" {
		t.Errorf("audit = %q by %d, to %q, %v; want the entry written with the change", action, actor, name, err)
	}
}

func TestSaveProfileThatFailsLeavesNeitherTheChangeNorTheEntry(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	repo := postgres.NewBusinesses(pool)
	owner := newOwner(t, pool, "owner@example.com")
	created, err := repo.Create(t.Context(), draft(t, owner, categoryID(t, pool, "yoga"), "Yoga Center"))
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	broken, err := created.UpdateProfile(profileInput(created, "Yoga Renamed", "x", "5491198765432", 999_999))
	if err != nil {
		t.Fatalf("UpdateProfile: %v", err)
	}
	if err := repo.SaveProfile(t.Context(), broken, profileEntry(owner, created.ID)); !errors.Is(err, business.ErrCategoryNotFound) {
		t.Fatalf("error = %v, want ErrCategoryNotFound", err)
	}
	missing := created
	missing.ID = 999_999
	if err := repo.SaveProfile(t.Context(), missing, profileEntry(owner, missing.ID)); !errors.Is(err, business.ErrBusinessNotFound) {
		t.Errorf("unknown business error = %v, want ErrBusinessNotFound", err)
	}

	var entries int
	if err := pool.QueryRow(t.Context(), `SELECT count(*) FROM audit_log WHERE entity = 'business'`).Scan(&entries); err != nil || entries != 0 {
		t.Errorf("audit entries = %d, %v; want none", entries, err)
	}
	if stored, _ := repo.FindByOwner(t.Context(), owner); stored.Name != "Yoga Center" {
		t.Errorf("name = %q, want it untouched", stored.Name)
	}
}

func profileInput(_ domain.Business, name, description, whatsapp string, category int64) domain.ProfileInput {
	return domain.ProfileInput{CategoryID: category, Name: name, Description: description, WhatsApp: whatsapp}
}
