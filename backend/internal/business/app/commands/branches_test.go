package commands_test

import (
	"errors"
	"testing"

	"github.com/joaquinBrassinne/MOVO/internal/business"
	"github.com/joaquinBrassinne/MOVO/internal/business/app/commands"
	"github.com/joaquinBrassinne/MOVO/internal/business/domain"
)

var validBranch = domain.NewBranchInput{
	Name: "Centro", Address: "Córdoba 1234", City: "Rosario", Latitude: -32.95, Longitude: -60.65,
}

// twoOwners builds a store where owner 7 and owner 8 each have a business.
func twoOwners(t *testing.T) (*fakeBusinesses, *fakeBranches) {
	t.Helper()

	businesses, branches := newFakeBusinesses(1), newFakeBranches()
	create := commands.NewCreateBusiness(businesses)
	for _, owner := range []int64{7, 8} {
		if _, err := create.Handle(t.Context(), createInput(owner, "Studio")); err != nil {
			t.Fatalf("create business of owner %d: %v", owner, err)
		}
	}
	return businesses, branches
}

func TestAddBranchAttachesItToTheCallersBusiness(t *testing.T) {
	t.Parallel()

	businesses, branches := twoOwners(t)
	owned, _ := businesses.FindByOwner(t.Context(), 7)

	got, err := commands.NewAddBranch(businesses, branches, unlimited()).Handle(t.Context(), commands.AddBranchInput{OwnerID: 7, Branch: validBranch})
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if got.ID == 0 || got.BusinessID != owned.ID || !got.Active {
		t.Errorf("branch = %+v, want it stored for business %d", got, owned.ID)
	}
}

func TestAddBranchRejections(t *testing.T) {
	t.Parallel()

	businesses, branches := twoOwners(t)
	add := commands.NewAddBranch(businesses, branches, unlimited())

	if _, err := add.Handle(t.Context(), commands.AddBranchInput{OwnerID: 99, Branch: validBranch}); !errors.Is(err, business.ErrBusinessNotFound) {
		t.Errorf("owner without business error = %v, want ErrBusinessNotFound", err)
	}
	if _, err := add.Handle(t.Context(), commands.AddBranchInput{OwnerID: 7}); !errors.Is(err, domain.ErrBranchNameInvalid) {
		t.Errorf("invalid branch error = %v, want ErrBranchNameInvalid", err)
	}
	if len(branches.byID) != 0 {
		t.Errorf("stored branches = %d, want none", len(branches.byID))
	}

	owned, _ := businesses.FindByOwner(t.Context(), 7)
	deleted, _ := owned.Delete()
	if err := businesses.Save(t.Context(), deleted); err != nil {
		t.Fatalf("delete business: %v", err)
	}
	if _, err := add.Handle(t.Context(), commands.AddBranchInput{OwnerID: 7, Branch: validBranch}); !errors.Is(err, domain.ErrNotEditable) {
		t.Errorf("deleted business error = %v, want ErrNotEditable", err)
	}
}

func TestEditBranchCannotReachAnotherBusiness(t *testing.T) {
	t.Parallel()

	businesses, branches := twoOwners(t)
	mine, err := commands.NewAddBranch(businesses, branches, unlimited()).Handle(t.Context(), commands.AddBranchInput{OwnerID: 7, Branch: validBranch})
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	edit := commands.NewEditBranch(businesses, branches)

	renamed := validBranch
	renamed.Name = "Norte"
	got, err := edit.Handle(t.Context(), commands.EditBranchInput{OwnerID: 7, BranchID: mine.ID, Branch: renamed})
	if err != nil || got.Name != "Norte" {
		t.Fatalf("own edit = %+v, %v", got, err)
	}

	renamed.Name = "Hijacked"
	if _, err := edit.Handle(t.Context(), commands.EditBranchInput{OwnerID: 8, BranchID: mine.ID, Branch: renamed}); !errors.Is(err, business.ErrBranchNotFound) {
		t.Fatalf("foreign edit error = %v, want ErrBranchNotFound", err)
	}
	if branches.byID[mine.ID].Name != "Norte" {
		t.Errorf("name = %q, a foreign edit must not change it", branches.byID[mine.ID].Name)
	}
}

func TestRemoveBranchCannotReachAnotherBusiness(t *testing.T) {
	t.Parallel()

	businesses, branches := twoOwners(t)
	mine, err := commands.NewAddBranch(businesses, branches, unlimited()).Handle(t.Context(), commands.AddBranchInput{OwnerID: 7, Branch: validBranch})
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	remove := commands.NewRemoveBranch(businesses, branches)

	if err := remove.Handle(t.Context(), commands.RemoveBranchInput{OwnerID: 8, BranchID: mine.ID}); !errors.Is(err, business.ErrBranchNotFound) {
		t.Fatalf("foreign remove error = %v, want ErrBranchNotFound", err)
	}
	if len(branches.byID) != 1 {
		t.Fatal("a foreign remove must not delete the branch")
	}
	if err := remove.Handle(t.Context(), commands.RemoveBranchInput{OwnerID: 7, BranchID: mine.ID}); err != nil {
		t.Fatalf("own remove: %v", err)
	}
	if len(branches.byID) != 0 {
		t.Error("the branch must be gone")
	}
}

func TestSetBranchHours(t *testing.T) {
	t.Parallel()

	businesses, branches := twoOwners(t)
	mine, err := commands.NewAddBranch(businesses, branches, unlimited()).Handle(t.Context(), commands.AddBranchInput{OwnerID: 7, Branch: validBranch})
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	set := commands.NewSetBranchHours(businesses, branches)

	week := []domain.Slot{{Weekday: 1, Start: 9 * 60, End: 13 * 60}, {Weekday: 1, Start: 16 * 60, End: 20 * 60}}
	if err := set.Handle(t.Context(), commands.SetBranchHoursInput{OwnerID: 7, BranchID: mine.ID, Slots: week}); err != nil {
		t.Fatalf("set hours: %v", err)
	}
	if got := branches.hours[mine.ID]; len(got) != 2 {
		t.Fatalf("stored slots = %v, want 2", got)
	}

	overlapping := []domain.Slot{{Weekday: 1, Start: 9 * 60, End: 13 * 60}, {Weekday: 1, Start: 12 * 60, End: 20 * 60}}
	if err := set.Handle(t.Context(), commands.SetBranchHoursInput{OwnerID: 7, BranchID: mine.ID, Slots: overlapping}); !errors.Is(err, domain.ErrSlotsOverlap) {
		t.Errorf("overlap error = %v, want ErrSlotsOverlap", err)
	}
	if len(branches.hours[mine.ID]) != 2 {
		t.Error("a rejected schedule must leave the previous one untouched")
	}

	if err := set.Handle(t.Context(), commands.SetBranchHoursInput{OwnerID: 8, BranchID: mine.ID, Slots: week}); !errors.Is(err, business.ErrBranchNotFound) {
		t.Errorf("foreign hours error = %v, want ErrBranchNotFound", err)
	}

	if err := set.Handle(t.Context(), commands.SetBranchHoursInput{OwnerID: 7, BranchID: mine.ID}); err != nil {
		t.Fatalf("clearing hours: %v", err)
	}
	if len(branches.hours[mine.ID]) != 0 {
		t.Error("an empty list must close the branch all week")
	}
}

func TestAddBranchRespectsThePlanLimit(t *testing.T) {
	t.Parallel()

	businesses, branches := twoOwners(t)
	add := commands.NewAddBranch(businesses, branches, fixedEntitlements{limits: limits{branches: 2}})

	var first domain.Branch
	for i := range 2 {
		got, err := add.Handle(t.Context(), commands.AddBranchInput{OwnerID: 7, Branch: validBranch})
		if err != nil {
			t.Fatalf("branch %d within the limit: %v", i+1, err)
		}
		if i == 0 {
			first = got
		}
	}

	if _, err := add.Handle(t.Context(), commands.AddBranchInput{OwnerID: 7, Branch: validBranch}); !errors.Is(err, business.ErrBranchLimitReached) {
		t.Fatalf("third branch error = %v, want ErrBranchLimitReached", err)
	}
	if n, _ := branches.List(t.Context(), 1); len(n) != 2 {
		t.Errorf("stored branches = %d, a rejected branch must not be stored", len(n))
	}

	// Another business has its own count.
	if _, err := add.Handle(t.Context(), commands.AddBranchInput{OwnerID: 8, Branch: validBranch}); err != nil {
		t.Errorf("another business must not be affected: %v", err)
	}

	// Branches left inactive by a downgrade do not use up the limit.
	first.Active = false
	if err := branches.Save(t.Context(), first); err != nil {
		t.Fatalf("deactivate: %v", err)
	}
	if _, err := add.Handle(t.Context(), commands.AddBranchInput{OwnerID: 7, Branch: validBranch}); err != nil {
		t.Errorf("an inactive branch must not count against the limit: %v", err)
	}
}
