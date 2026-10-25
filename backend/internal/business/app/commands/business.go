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

	if err := c.businesses.Save(ctx, updated); err != nil {
		return domain.Business{}, fmt.Errorf("save business: %w", err)
	}
	return updated, nil
}

// SubmitBusinessInput asks for the manual review.
type SubmitBusinessInput struct {
	OwnerID          int64
	PaymentReference string
}

// SubmitBusiness sends the owner's draft business to review.
type SubmitBusiness struct {
	businesses business.BusinessRepository
}

// NewSubmitBusiness wires the SubmitBusiness use case.
func NewSubmitBusiness(businesses business.BusinessRepository) *SubmitBusiness {
	return &SubmitBusiness{businesses: businesses}
}

// Handle moves the business from draft to pending_review, recording the optional
// operation number. A business that is not a draft yields domain.ErrInvalidTransition.
func (c *SubmitBusiness) Handle(ctx context.Context, in SubmitBusinessInput) (domain.Business, error) {
	current, err := c.businesses.FindByOwner(ctx, in.OwnerID)
	if err != nil {
		return domain.Business{}, fmt.Errorf("find business: %w", err)
	}

	submitted, err := current.Submit()
	if err != nil {
		return domain.Business{}, err
	}
	if submitted, err = submitted.SetPaymentReference(in.PaymentReference); err != nil {
		return domain.Business{}, err
	}

	if err := c.businesses.Save(ctx, submitted); err != nil {
		return domain.Business{}, fmt.Errorf("save business: %w", err)
	}
	return submitted, nil
}
