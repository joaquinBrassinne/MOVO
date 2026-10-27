// Package benefits is the module that lets a business offer benefits and lets a
// customer turn them into coupons. This file declares the ports the application
// layer consumes; adapters implement them.
package benefits

import (
	"context"
	"errors"

	"github.com/joaquinBrassinne/MOVO/internal/benefits/domain"
)

// Errors the adapters report so the application layer can react without knowing the
// storage engine.
var (
	ErrBenefitNotFound = errors.New("benefits: benefit not found")
	// ErrBusinessNotFound means the owner has not created a business yet.
	ErrBusinessNotFound = errors.New("benefits: owner has no business")
	// ErrBusinessNotEditable means the business was deleted and can no longer change.
	ErrBusinessNotEditable = errors.New("benefits: business can no longer change")
)

// OwnerBusinesses resolves the business an owner may change. The composition root
// builds it from the business module, so benefits never reads businesses and the
// business id always comes from the session, never from a request.
type OwnerBusinesses interface {
	// EditableBusinessOf returns the id of the owner's business, ErrBusinessNotFound
	// when there is none, or ErrBusinessNotEditable when it was deleted.
	EditableBusinessOf(ctx context.Context, ownerID int64) (int64, error)
}

// BenefitRepository stores benefits. Every method is scoped by business id, so a
// benefit id of another business behaves as if it did not exist.
type BenefitRepository interface {
	// Create stores a new benefit and returns it with ID set.
	Create(ctx context.Context, b domain.Benefit) (domain.Benefit, error)
	// List returns every benefit of the business, oldest first, inactive ones included.
	List(ctx context.Context, businessID int64) ([]domain.Benefit, error)
	// Find returns ErrBenefitNotFound when the business has no such benefit.
	Find(ctx context.Context, businessID, benefitID int64) (domain.Benefit, error)
	// Save persists the details and the active flag; ErrBenefitNotFound as in Find.
	Save(ctx context.Context, b domain.Benefit) error
}
