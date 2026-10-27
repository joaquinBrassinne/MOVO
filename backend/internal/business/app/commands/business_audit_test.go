package commands_test

import (
	"testing"

	"github.com/joaquinBrassinne/MOVO/internal/business/app/commands"
)

func TestUpdateBusinessRecordsEachChangeInTheAuditTrail(t *testing.T) {
	t.Parallel()

	store := newFakeBusinesses(1, 2)
	created, err := commands.NewCreateBusiness(store).Handle(t.Context(), createInput(7, "Old Name"))
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	update := commands.NewUpdateBusiness(store)

	// Name and category change; the description and the WhatsApp stay as they were.
	if _, err := update.Handle(t.Context(), commands.UpdateBusinessInput{
		OwnerID: 7, CategoryID: 2, Name: "New Name", Description: created.Description, WhatsApp: created.WhatsApp,
	}); err != nil {
		t.Fatalf("Handle: %v", err)
	}

	if len(store.audit) != 1 {
		t.Fatalf("audit entries = %d, want one per saved change", len(store.audit))
	}
	entry := store.audit[0]
	if entry.Action != "business.profile_update" || entry.ActorID != 7 || entry.Entity != "business" || entry.EntityID != created.ID {
		t.Errorf("entry = %+v", entry)
	}
	changes, _ := entry.Metadata["changes"].(map[string]map[string]any)
	if len(changes) != 2 || changes["name"]["from"] != "Old Name" || changes["name"]["to"] != "New Name" ||
		changes["category_id"]["from"] != created.CategoryID || changes["category_id"]["to"] != int64(2) {
		t.Errorf("changes = %v, want only the name and the category, each with its old and new value", changes)
	}
}

func TestUpdateBusinessWithNothingNewSavesAndRecordsNothing(t *testing.T) {
	t.Parallel()

	store := newFakeBusinesses(1)
	created, err := commands.NewCreateBusiness(store).Handle(t.Context(), createInput(7, "Same Name"))
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	saves := store.saves

	got, err := commands.NewUpdateBusiness(store).Handle(t.Context(), commands.UpdateBusinessInput{
		OwnerID: 7, CategoryID: created.CategoryID, Name: "  Same Name ", Description: created.Description, WhatsApp: created.WhatsApp,
	})
	if err != nil || got.Name != "Same Name" {
		t.Fatalf("Handle = %+v, %v", got, err)
	}
	if store.saves != saves || len(store.audit) != 0 {
		t.Errorf("saves %d -> %d, audit %d; want nothing written when nothing changed", saves, store.saves, len(store.audit))
	}
}

func TestUpdateBusinessThatFailsRecordsNothing(t *testing.T) {
	t.Parallel()

	store := newFakeBusinesses(1)
	if _, err := commands.NewCreateBusiness(store).Handle(t.Context(), createInput(7, "Old Name")); err != nil {
		t.Fatalf("create: %v", err)
	}

	// Category 9 does not exist: the store refuses, so there is no change to record.
	if _, err := commands.NewUpdateBusiness(store).Handle(t.Context(), commands.UpdateBusinessInput{
		OwnerID: 7, CategoryID: 9, Name: "New Name", WhatsApp: "5491198765432",
	}); err == nil {
		t.Fatal("an unknown category must fail")
	}
	if len(store.audit) != 0 {
		t.Errorf("audit entries = %d, want none for a change that was not applied", len(store.audit))
	}
}
