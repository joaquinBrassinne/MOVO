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

	if err := c.checkLimit(ctx, owned.ID); err != nil {
		return domain.Branch{}, err
	}

	stored, err := c.branches.Create(ctx, branch)
	if err != nil {
		return domain.Branch{}, fmt.Errorf("create branch: %w", err)
	}
	return stored, nil
}

// checkLimit counts only active branches: the ones a downgrade left inactive do
// not use up the plan.
func (c *AddBranch) checkLimit(ctx context.Context, businessID int64) error {
	ent, err := c.entitlements.For(ctx, businessID)
	if err != nil {
		return fmt.Errorf("resolve entitlements: %w", err)
	}
	existing, err := c.branches.List(ctx, businessID)
	if err != nil {
		return fmt.Errorf("list branches: %w", err)
	}

	active := 0
	for _, b := range existing {
		if b.Active {
			active++
		}
	}
	if !ent.CanAddBranch(active) {
		return business.ErrBranchLimitReached
	}
	return nil
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

// RemoveBranchInput identifies the branch to remove.
type RemoveBranchInput struct {
	OwnerID  int64
	BranchID int64
}

// RemoveBranch deletes a branch of the owner's business.
type RemoveBranch struct {
	businesses business.BusinessRepository
	branches   business.BranchRepository
}

// NewRemoveBranch wires the RemoveBranch use case.
func NewRemoveBranch(businesses business.BusinessRepository, branches business.BranchRepository) *RemoveBranch {
	return &RemoveBranch{businesses: businesses, branches: branches}
}

// Handle removes the branch with its hours.
func (c *RemoveBranch) Handle(ctx context.Context, in RemoveBranchInput) error {
	owned, err := editableBusiness(ctx, c.businesses, in.OwnerID)
	if err != nil {
		return err
	}
	if err := c.branches.Delete(ctx, owned.ID, in.BranchID); err != nil {
		return fmt.Errorf("delete branch: %w", err)
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
