package commands

import (
	"context"
	"fmt"

	"github.com/joaquinBrassinne/MOVO/internal/business"
	"github.com/joaquinBrassinne/MOVO/internal/business/domain"
)

// editableBusiness returns the business of the owner, or domain.ErrNotEditable
// once it was deleted. The business id used by every branch operation comes from
// here, never from the request.
func editableBusiness(ctx context.Context, businesses business.BusinessRepository, ownerID int64) (domain.Business, error) {
	b, err := businesses.FindByOwner(ctx, ownerID)
	if err != nil {
		return domain.Business{}, fmt.Errorf("find business: %w", err)
	}
	if !b.Editable() {
		return domain.Business{}, domain.ErrNotEditable
	}
	return b, nil
}

// AddBranchInput is a new branch of the owner's business.
type AddBranchInput struct {
	OwnerID int64
	Branch  domain.NewBranchInput
}

// AddBranch adds a branch to the owner's business.
type AddBranch struct {
	businesses   business.BusinessRepository
	branches     business.BranchRepository
	entitlements business.EntitlementsProvider
}

// NewAddBranch wires the AddBranch use case.
func NewAddBranch(businesses business.BusinessRepository, branches business.BranchRepository, entitlements business.EntitlementsProvider) *AddBranch {
	return &AddBranch{businesses: businesses, branches: branches, entitlements: entitlements}
}

// Handle validates the branch and stores it. It returns
// business.ErrBranchLimitReached when the plan has no room for another active branch.
func (c *AddBranch) Handle(ctx context.Context, in AddBranchInput) (domain.Branch, error) {
	owned, err := editableBusiness(ctx, c.businesses, in.OwnerID)
	if err != nil {
		return domain.Branch{}, err
	}

	branch, err := domain.NewBranch(in.Branch)
	if err != nil {
		return domain.Branch{}, err
	}
	branch.BusinessID = owned.ID

	// The repository counts the active branches and stores this one under a single
	// lock, so the plan limit holds for concurrent requests. The ones a downgrade
	// left inactive do not use up the plan.
	ent, err := c.entitlements.For(ctx, owned.ID)
	if err != nil {
		return domain.Branch{}, fmt.Errorf("resolve entitlements: %w", err)
	}
	stored, err := c.branches.CreateIfRoom(ctx, branch, ent.CanAddBranch)
	if err != nil {
		return domain.Branch{}, fmt.Errorf("create branch: %w", err)
	}
	return stored, nil
}

// EditBranchInput is the new details of an existing branch.
type EditBranchInput struct {
	OwnerID  int64
	BranchID int64
	Branch   domain.NewBranchInput
}

// EditBranch changes the details of a branch of the owner's business.
type EditBranch struct {
	businesses business.BusinessRepository
	branches   business.BranchRepository
}

// NewEditBranch wires the EditBranch use case.
func NewEditBranch(businesses business.BusinessRepository, branches business.BranchRepository) *EditBranch {
	return &EditBranch{businesses: businesses, branches: branches}
}

// Handle applies the new details. A branch of another business is reported as
// business.ErrBranchNotFound, exactly like one that does not exist.
func (c *EditBranch) Handle(ctx context.Context, in EditBranchInput) (domain.Branch, error) {
	owned, err := editableBusiness(ctx, c.businesses, in.OwnerID)
	if err != nil {
		return domain.Branch{}, err
	}

	current, err := c.branches.Find(ctx, owned.ID, in.BranchID)
	if err != nil {
		return domain.Branch{}, fmt.Errorf("find branch: %w", err)
	}
	updated, err := current.Update(in.Branch)
	if err != nil {
		return domain.Branch{}, err
	}

	if err := c.branches.Save(ctx, updated); err != nil {
		return domain.Branch{}, fmt.Errorf("save branch: %w", err)
	}
	return updated, nil
}

// RetireBranchInput identifies the branch to retire.
type RetireBranchInput struct {
	OwnerID  int64
	BranchID int64
}

// RetireBranch takes a branch of the owner's business off the map. A branch is never
// deleted: metrics, redemptions and reviews reference it (SDD 4), so retiring it
// sets active = false and keeps its history. The owner can show it again with
// SetBranchActive, which checks the plan.
type RetireBranch struct {
	businesses business.BusinessRepository
	branches   business.BranchRepository
}

// NewRetireBranch wires the RetireBranch use case.
func NewRetireBranch(businesses business.BusinessRepository, branches business.BranchRepository) *RetireBranch {
	return &RetireBranch{businesses: businesses, branches: branches}
}

// Handle deactivates the branch. Retiring one that is already inactive changes
// nothing. A branch of another business is business.ErrBranchNotFound.
func (c *RetireBranch) Handle(ctx context.Context, in RetireBranchInput) error {
	owned, err := editableBusiness(ctx, c.businesses, in.OwnerID)
	if err != nil {
		return err
	}
	if _, err := c.branches.SetActive(ctx, owned.ID, in.BranchID, false, nil); err != nil {
		return fmt.Errorf("retire branch: %w", err)
	}
	return nil
}

// SetBranchHoursInput replaces the opening hours of a branch.
type SetBranchHoursInput struct {
	OwnerID  int64
	BranchID int64
	Slots    []domain.Slot
}

// SetBranchHours replaces the whole weekly schedule of a branch.
type SetBranchHours struct {
	businesses business.BusinessRepository
	branches   business.BranchRepository
}

// NewSetBranchHours wires the SetBranchHours use case.
func NewSetBranchHours(businesses business.BusinessRepository, branches business.BranchRepository) *SetBranchHours {
	return &SetBranchHours{businesses: businesses, branches: branches}
}

// Handle validates the slots (no overlaps on a weekday) and swaps them in. An
// empty list marks the branch as closed all week.
func (c *SetBranchHours) Handle(ctx context.Context, in SetBranchHoursInput) error {
	owned, err := editableBusiness(ctx, c.businesses, in.OwnerID)
	if err != nil {
		return err
	}
	if err := domain.ValidateSlots(in.Slots); err != nil {
		return err
	}
	if err := c.branches.ReplaceHours(ctx, owned.ID, in.BranchID, in.Slots); err != nil {
		return fmt.Errorf("replace hours: %w", err)
	}
	return nil
}

// SetBranchActiveInput turns a branch on or off.
type SetBranchActiveInput struct {
	OwnerID  int64
	BranchID int64
	Active   bool
}

// SetBranchActive shows or hides a branch of the owner's business.
type SetBranchActive struct {
	businesses   business.BusinessRepository
	branches     business.BranchRepository
	entitlements business.EntitlementsProvider
}

// NewSetBranchActive wires the SetBranchActive use case.
func NewSetBranchActive(businesses business.BusinessRepository, branches business.BranchRepository, entitlements business.EntitlementsProvider) *SetBranchActive {
	return &SetBranchActive{businesses: businesses, branches: branches, entitlements: entitlements}
}

// Handle switches the flag. Hiding is always allowed; showing checks the plan, except
// when the branch is already shown, which changes nothing.
func (c *SetBranchActive) Handle(ctx context.Context, in SetBranchActiveInput) (domain.Branch, error) {
	owned, err := editableBusiness(ctx, c.businesses, in.OwnerID)
	if err != nil {
		return domain.Branch{}, err
	}
	// Hiding never asks the plan, so it keeps working if the entitlements cannot be read.
	var fits business.Fits
	if in.Active {
		ent, err := c.entitlements.For(ctx, owned.ID)
		if err != nil {
			return domain.Branch{}, fmt.Errorf("resolve entitlements: %w", err)
		}
		fits = ent.CanAddBranch
	}

	updated, err := c.branches.SetActive(ctx, owned.ID, in.BranchID, in.Active, fits)
	if err != nil {
		return domain.Branch{}, fmt.Errorf("set branch active: %w", err)
	}
	return updated, nil
}
