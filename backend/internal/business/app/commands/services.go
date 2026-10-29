package commands

import (
	"context"
	"fmt"

	"github.com/joaquinBrassinne/MOVO/internal/business"
	"github.com/joaquinBrassinne/MOVO/internal/business/domain"
)

// servicesDeps are the collaborators the service use cases share.
type servicesDeps struct {
	businesses   business.BusinessRepository
	branches     business.BranchRepository
	services     business.ServiceRepository
	entitlements business.EntitlementsProvider
}

// checkBranch accepts 0 (the whole business) or a branch of the same business; a
// branch of another business is reported as business.ErrBranchNotFound.
func (d servicesDeps) checkBranch(ctx context.Context, businessID, branchID int64) error {
	if branchID == 0 {
		return nil
	}
	if _, err := d.branches.Find(ctx, businessID, branchID); err != nil {
		return fmt.Errorf("find branch: %w", err)
	}
	return nil
}

// fitsOneMore is the plan's answer to "may this business have one more active
// service". The repository asks it under its lock on the business; inactive
// services never count.
func (d servicesDeps) fitsOneMore(ctx context.Context, businessID int64) (business.Fits, error) {
	ent, err := d.entitlements.For(ctx, businessID)
	if err != nil {
		return nil, fmt.Errorf("resolve entitlements: %w", err)
	}
	return ent.CanActivateService, nil
}

// AddServiceInput is a new service of the owner's business.
type AddServiceInput struct {
	OwnerID int64
	Service domain.ServiceInput
}

// AddService creates an active service.
type AddService struct{ servicesDeps }

// NewAddService wires the AddService use case.
func NewAddService(businesses business.BusinessRepository, branches business.BranchRepository, services business.ServiceRepository, entitlements business.EntitlementsProvider) *AddService {
	return &AddService{servicesDeps{businesses, branches, services, entitlements}}
}

// Handle validates and stores the service. It returns business.ErrServiceLimitReached
// when the plan has no room for another active service.
func (c *AddService) Handle(ctx context.Context, in AddServiceInput) (domain.Service, error) {
	owned, err := editableBusiness(ctx, c.businesses, in.OwnerID)
	if err != nil {
		return domain.Service{}, err
	}
	service, err := domain.NewService(in.Service)
	if err != nil {
		return domain.Service{}, err
	}
	service.BusinessID = owned.ID

	if err := c.checkBranch(ctx, owned.ID, service.BranchID); err != nil {
		return domain.Service{}, err
	}
	fits, err := c.fitsOneMore(ctx, owned.ID)
	if err != nil {
		return domain.Service{}, err
	}

	stored, err := c.services.CreateIfRoom(ctx, service, fits)
	if err != nil {
		return domain.Service{}, fmt.Errorf("create service: %w", err)
	}
	return stored, nil
}

// EditServiceInput is the new details of an existing service.
type EditServiceInput struct {
	OwnerID   int64
	ServiceID int64
	Service   domain.ServiceInput
}

// EditService changes the details of a service. Its active flag is untouched.
type EditService struct{ servicesDeps }

// NewEditService wires the EditService use case.
func NewEditService(businesses business.BusinessRepository, branches business.BranchRepository, services business.ServiceRepository) *EditService {
	return &EditService{servicesDeps{businesses: businesses, branches: branches, services: services}}
}

// Handle applies the new details. A service of another business is reported as
// business.ErrServiceNotFound, exactly like one that does not exist.
func (c *EditService) Handle(ctx context.Context, in EditServiceInput) (domain.Service, error) {
	owned, err := editableBusiness(ctx, c.businesses, in.OwnerID)
	if err != nil {
		return domain.Service{}, err
	}
	current, err := c.services.Find(ctx, owned.ID, in.ServiceID)
	if err != nil {
		return domain.Service{}, fmt.Errorf("find service: %w", err)
	}
	updated, err := current.Update(in.Service)
	if err != nil {
		return domain.Service{}, err
	}
	if err := c.checkBranch(ctx, owned.ID, updated.BranchID); err != nil {
		return domain.Service{}, err
	}

	if err := c.services.Save(ctx, updated); err != nil {
		return domain.Service{}, fmt.Errorf("save service: %w", err)
	}
	return updated, nil
}

// SetServiceActiveInput turns a service on or off.
type SetServiceActiveInput struct {
	OwnerID   int64
	ServiceID int64
	Active    bool
}

// SetServiceActive activates or deactivates a service.
type SetServiceActive struct{ servicesDeps }

// NewSetServiceActive wires the SetServiceActive use case.
func NewSetServiceActive(businesses business.BusinessRepository, services business.ServiceRepository, entitlements business.EntitlementsProvider) *SetServiceActive {
	return &SetServiceActive{servicesDeps{businesses: businesses, services: services, entitlements: entitlements}}
}

// Handle switches the flag. Deactivating is always allowed; activating checks the
// plan, except when the service is already active, which changes nothing.
func (c *SetServiceActive) Handle(ctx context.Context, in SetServiceActiveInput) (domain.Service, error) {
	owned, err := editableBusiness(ctx, c.businesses, in.OwnerID)
	if err != nil {
		return domain.Service{}, err
	}
	// Deactivating never asks the plan, so it keeps working if the entitlements
	// cannot be read.
	var fits business.Fits
	if in.Active {
		if fits, err = c.fitsOneMore(ctx, owned.ID); err != nil {
			return domain.Service{}, err
		}
	}

	updated, err := c.services.SetActive(ctx, owned.ID, in.ServiceID, in.Active, fits)
	if err != nil {
		return domain.Service{}, fmt.Errorf("set service active: %w", err)
	}
	return updated, nil
}

// RemoveServiceInput identifies the service to remove.
type RemoveServiceInput struct {
	OwnerID   int64
	ServiceID int64
}

// RemoveService deletes a service with its schedule.
type RemoveService struct{ servicesDeps }

// NewRemoveService wires the RemoveService use case.
func NewRemoveService(businesses business.BusinessRepository, services business.ServiceRepository) *RemoveService {
	return &RemoveService{servicesDeps{businesses: businesses, services: services}}
}

// Handle removes the service of the owner's business.
func (c *RemoveService) Handle(ctx context.Context, in RemoveServiceInput) error {
	owned, err := editableBusiness(ctx, c.businesses, in.OwnerID)
	if err != nil {
		return err
	}
	if err := c.services.Delete(ctx, owned.ID, in.ServiceID); err != nil {
		return fmt.Errorf("delete service: %w", err)
	}
	return nil
}

// SetServiceScheduleInput replaces the weekly schedule of a service.
type SetServiceScheduleInput struct {
	OwnerID   int64
	ServiceID int64
	Slots     []domain.Slot
}

// SetServiceSchedule replaces the whole weekly schedule of a service.
type SetServiceSchedule struct{ servicesDeps }

// NewSetServiceSchedule wires the SetServiceSchedule use case.
func NewSetServiceSchedule(businesses business.BusinessRepository, services business.ServiceRepository) *SetServiceSchedule {
	return &SetServiceSchedule{servicesDeps{businesses: businesses, services: services}}
}

// Handle validates the slots (no overlaps on a weekday) and swaps them in. An
// empty list leaves the service without fixed schedule.
func (c *SetServiceSchedule) Handle(ctx context.Context, in SetServiceScheduleInput) error {
	owned, err := editableBusiness(ctx, c.businesses, in.OwnerID)
	if err != nil {
		return err
	}
	if err := domain.ValidateSlots(in.Slots); err != nil {
		return err
	}
	if err := c.services.ReplaceSchedule(ctx, owned.ID, in.ServiceID, in.Slots); err != nil {
		return fmt.Errorf("replace schedule: %w", err)
	}
	return nil
}
