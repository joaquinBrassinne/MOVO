// Package commands holds the write use cases of the business module.
package commands

import (
	"context"
	"errors"
	"fmt"

	"github.com/joaquinBrassinne/MOVO/internal/business"
	"github.com/joaquinBrassinne/MOVO/internal/business/domain"
)

// maxSlugAttempts bounds the search for a free slug: base, base-2, ... base-10.
const maxSlugAttempts = 10

// ErrSlugExhausted means no free slug was found for a very popular name.
var ErrSlugExhausted = errors.New("business: could not find a free slug")

// CreateBusinessInput is what an owner sends to create the business. The owner
// comes from the session.
type CreateBusinessInput struct {
	OwnerID     int64
	CategoryID  int64
	Name        string
	Description string
	WhatsApp    string
}

// CreateBusiness creates the draft business of an owner.
type CreateBusiness struct {
	businesses business.BusinessRepository
}

// NewCreateBusiness wires the CreateBusiness use case.
func NewCreateBusiness(businesses business.BusinessRepository) *CreateBusiness {
	return &CreateBusiness{businesses: businesses}
}

// Handle validates the input and stores the business, suffixing the slug when
// another business already has it. Validation failures are the domain errors;
// business.ErrOwnerHasBusiness and business.ErrCategoryNotFound come from the store.
func (c *CreateBusiness) Handle(ctx context.Context, in CreateBusinessInput) (domain.Business, error) {
	draft, err := domain.NewBusiness(domain.NewBusinessInput{
		OwnerID:     in.OwnerID,
		CategoryID:  in.CategoryID,
		Name:        in.Name,
		Description: in.Description,
		WhatsApp:    in.WhatsApp,
	})
	if err != nil {
		return domain.Business{}, err
	}

	base := draft.Slug
	for attempt := 1; attempt <= maxSlugAttempts; attempt++ {
		draft.Slug = domain.SlugWithSuffix(base, attempt)
		stored, err := c.businesses.Create(ctx, draft)
		if errors.Is(err, business.ErrSlugTaken) {
			continue
		}
		if err != nil {
			return domain.Business{}, fmt.Errorf("create business: %w", err)
		}
		return stored, nil
	}
	return domain.Business{}, ErrSlugExhausted
}

// UpdateBusinessInput is the new profile of the owner's business.
type UpdateBusinessInput struct {
	OwnerID     int64
	CategoryID  int64
	Name        string
	Description string
	WhatsApp    string
}

// UpdateBusiness edits the profile of the owner's business.
type UpdateBusiness struct {
	businesses business.BusinessRepository
}

// NewUpdateBusiness wires the UpdateBusiness use case.
func NewUpdateBusiness(businesses business.BusinessRepository) *UpdateBusiness {
	return &UpdateBusiness{businesses: businesses}
}

// Handle applies the new profile to the business of in.OwnerID.
func (c *UpdateBusiness) Handle(ctx context.Context, in UpdateBusinessInput) (domain.Business, error) {
	current, err := c.businesses.FindByOwner(ctx, in.OwnerID)
	if err != nil {
		return domain.Business{}, fmt.Errorf("find business: %w", err)
	}

	updated, err := current.UpdateProfile(domain.ProfileInput{
		CategoryID:  in.CategoryID,
		Name:        in.Name,
		Description: in.Description,
		WhatsApp:    in.WhatsApp,
	})
	if err != nil {
		return domain.Business{}, err
	}

	// Nothing to save, and nothing to record, when the owner sent what was already there.
	changes := profileChanges(current, updated)
	if len(changes) == 0 {
		return updated, nil
	}
	entry := business.AuditEntry{
		ActorID:  in.OwnerID,
		Action:   ActionProfileUpdate,
		Entity:   "business",
		EntityID: current.ID,
		Metadata: map[string]any{"changes": changes},
	}
	if err := c.businesses.SaveProfile(ctx, updated, entry); err != nil {
		return domain.Business{}, fmt.Errorf("save business: %w", err)
	}
	return updated, nil
}

// ActionProfileUpdate is the audit action of an owner changing the profile of the business.
const ActionProfileUpdate = "business.profile_update"

// profileChanges lists each field of the profile that differs, with its old and new value.
func profileChanges(before, after domain.Business) map[string]map[string]any {
	changes := map[string]map[string]any{}
	note := func(field string, from, to any) {
		if from != to {
			changes[field] = map[string]any{"from": from, "to": to}
		}
	}
	note("name", before.Name, after.Name)
	note("category_id", before.CategoryID, after.CategoryID)
	note("description", before.Description, after.Description)
	note("whatsapp", before.WhatsApp, after.WhatsApp)
	return changes
}

// SubmitBusinessInput asks for the manual review.
type SubmitBusinessInput struct {
	OwnerID        int64
	PayerEmail     string
	PayerName      string
	MPReturnParams map[string]string
}

// SubmitBusiness sends the owner's draft business to review.
type SubmitBusiness struct {
	businesses business.BusinessRepository
}

// NewSubmitBusiness wires the SubmitBusiness use case.
func NewSubmitBusiness(businesses business.BusinessRepository) *SubmitBusiness {
	return &SubmitBusiness{businesses: businesses}
}

// Handle moves the business from draft to pending_review, recording the Mercado Pago
// email and name the owner declares. A business that is not a draft yields
// domain.ErrInvalidTransition; a missing or invalid declaration leaves it a draft.
func (c *SubmitBusiness) Handle(ctx context.Context, in SubmitBusinessInput) (domain.Business, error) {
	current, err := c.businesses.FindByOwner(ctx, in.OwnerID)
	if err != nil {
		return domain.Business{}, fmt.Errorf("find business: %w", err)
	}

	submitted, err := current.Submit()
	if err != nil {
		return domain.Business{}, err
	}
	if submitted, err = submitted.DeclarePayer(in.PayerEmail, in.PayerName); err != nil {
		return domain.Business{}, err
	}
	submitted = submitted.RecordReturnParams(in.MPReturnParams)

	if err := c.businesses.Save(ctx, submitted); err != nil {
		return domain.Business{}, fmt.Errorf("save business: %w", err)
	}
	return submitted, nil
}
