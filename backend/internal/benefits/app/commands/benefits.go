// Package commands holds the benefit use cases that change state.
package commands

import (
	"context"
	"fmt"

	"github.com/joaquinBrassinne/MOVO/internal/benefits"
	"github.com/joaquinBrassinne/MOVO/internal/benefits/domain"
)

// AddBenefitInput is a new benefit of the owner's business.
type AddBenefitInput struct {
	OwnerID int64
	Benefit domain.BenefitInput
}

// AddBenefit creates an active benefit.
type AddBenefit struct {
	owners   benefits.OwnerBusinesses
	benefits benefits.BenefitRepository
}

// NewAddBenefit wires the AddBenefit use case.
func NewAddBenefit(owners benefits.OwnerBusinesses, repo benefits.BenefitRepository) *AddBenefit {
	return &AddBenefit{owners: owners, benefits: repo}
}

// Handle validates and stores the benefit under the owner's business.
func (c *AddBenefit) Handle(ctx context.Context, in AddBenefitInput) (domain.Benefit, error) {
	businessID, err := c.owners.EditableBusinessOf(ctx, in.OwnerID)
	if err != nil {
		return domain.Benefit{}, fmt.Errorf("resolve business: %w", err)
	}
	benefit, err := domain.NewBenefit(in.Benefit)
	if err != nil {
		return domain.Benefit{}, err
	}
	benefit.BusinessID = businessID

	stored, err := c.benefits.Create(ctx, benefit)
	if err != nil {
		return domain.Benefit{}, fmt.Errorf("create benefit: %w", err)
	}
	return stored, nil
}

// EditBenefitInput is the new details of an existing benefit.
type EditBenefitInput struct {
	OwnerID   int64
	BenefitID int64
	Benefit   domain.BenefitInput
}

// EditBenefit changes the details and the monthly limit of a benefit. Its active flag
// is untouched.
type EditBenefit struct {
	owners   benefits.OwnerBusinesses
	benefits benefits.BenefitRepository
}

// NewEditBenefit wires the EditBenefit use case.
func NewEditBenefit(owners benefits.OwnerBusinesses, repo benefits.BenefitRepository) *EditBenefit {
	return &EditBenefit{owners: owners, benefits: repo}
}

// Handle applies the new details. A benefit of another business is reported as
// benefits.ErrBenefitNotFound, exactly like one that does not exist.
func (c *EditBenefit) Handle(ctx context.Context, in EditBenefitInput) (domain.Benefit, error) {
	businessID, err := c.owners.EditableBusinessOf(ctx, in.OwnerID)
	if err != nil {
		return domain.Benefit{}, fmt.Errorf("resolve business: %w", err)
	}
	current, err := c.benefits.Find(ctx, businessID, in.BenefitID)
	if err != nil {
		return domain.Benefit{}, fmt.Errorf("find benefit: %w", err)
	}
	updated, err := current.Update(in.Benefit)
	if err != nil {
		return domain.Benefit{}, err
	}

	if err := c.benefits.Save(ctx, updated); err != nil {
		return domain.Benefit{}, fmt.Errorf("save benefit: %w", err)
	}
	return updated, nil
}

// SetBenefitActiveInput turns a benefit on or off.
type SetBenefitActiveInput struct {
	OwnerID   int64
	BenefitID int64
	Active    bool
}

// SetBenefitActive activates or deactivates a benefit. Benefits are never deleted:
// their coupons are history.
type SetBenefitActive struct {
	owners   benefits.OwnerBusinesses
	benefits benefits.BenefitRepository
}

// NewSetBenefitActive wires the SetBenefitActive use case.
func NewSetBenefitActive(owners benefits.OwnerBusinesses, repo benefits.BenefitRepository) *SetBenefitActive {
	return &SetBenefitActive{owners: owners, benefits: repo}
}

// Handle switches the flag; asking for the state it already has changes nothing.
func (c *SetBenefitActive) Handle(ctx context.Context, in SetBenefitActiveInput) (domain.Benefit, error) {
	businessID, err := c.owners.EditableBusinessOf(ctx, in.OwnerID)
	if err != nil {
		return domain.Benefit{}, fmt.Errorf("resolve business: %w", err)
	}
	current, err := c.benefits.Find(ctx, businessID, in.BenefitID)
	if err != nil {
		return domain.Benefit{}, fmt.Errorf("find benefit: %w", err)
	}
	if current.Active == in.Active {
		return current, nil
	}

	updated := current.Deactivate()
	if in.Active {
		updated = current.Activate()
	}
	if err := c.benefits.Save(ctx, updated); err != nil {
		return domain.Benefit{}, fmt.Errorf("save benefit: %w", err)
	}
	return updated, nil
}
