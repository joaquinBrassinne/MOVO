// Package business is the module that lets an owner describe a business, its
// branches and opening hours, and that moves the business through its review.
// This file declares the ports the application layer consumes; adapters
// implement them.
package business

import (
	"context"
	"errors"

	"github.com/joaquinBrassinne/MOVO/internal/business/domain"
)

// Errors the adapters report so the application layer can react without knowing
// the storage engine.
var (
	ErrBusinessNotFound   = errors.New("business: business not found")
	ErrOwnerHasBusiness   = errors.New("business: owner already has a business")
	ErrSlugTaken          = errors.New("business: slug is already taken")
	ErrCategoryNotFound   = errors.New("business: category not found")
	ErrBranchNotFound     = errors.New("business: branch not found")
	ErrBranchLimitReached = errors.New("business: branch limit of the plan reached")
)

// BusinessRepository stores businesses. One owner has at most one business.
type BusinessRepository interface {
	// Create stores a new business and returns it with ID and CreatedAt set. It
	// returns ErrOwnerHasBusiness, ErrSlugTaken or ErrCategoryNotFound.
	Create(ctx context.Context, b domain.Business) (domain.Business, error)
	// FindByOwner returns ErrBusinessNotFound when the owner has no business. The
	// owner id always comes from the session, never from the request body.
	FindByOwner(ctx context.Context, ownerID int64) (domain.Business, error)
	// Save persists the mutable fields (profile, status, payment reference).
	// The slug and the owner never change. It returns ErrBusinessNotFound and
	// ErrCategoryNotFound.
	Save(ctx context.Context, b domain.Business) error
}

// BranchRepository stores the branches of a business and their opening hours.
// Every method is scoped by business id so an owner can never reach the branch
// of another business.
type BranchRepository interface {
	// Create stores a new branch and returns it with ID set.
	Create(ctx context.Context, b domain.Branch) (domain.Branch, error)
	// List returns the branches of the business, oldest first.
	List(ctx context.Context, businessID int64) ([]domain.Branch, error)
	// Find returns ErrBranchNotFound when the branch is not in the business.
	Find(ctx context.Context, businessID, branchID int64) (domain.Branch, error)
	// Save persists the details of an existing branch (ErrBranchNotFound).
	Save(ctx context.Context, b domain.Branch) error
	// Delete removes the branch with its hours (ErrBranchNotFound).
	Delete(ctx context.Context, businessID, branchID int64) error
	// Hours returns the opening slots of the branch, ordered by weekday and start.
	Hours(ctx context.Context, businessID, branchID int64) ([]domain.Slot, error)
	// ReplaceHours atomically swaps all the slots of the branch (ErrBranchNotFound).
	ReplaceHours(ctx context.Context, businessID, branchID int64, slots []domain.Slot) error
}

// Entitlements answers what a business may have. The limits are checked when
// something is created or activated, never when listing.
type Entitlements interface {
	// CanAddBranch reports whether a business with current active branches may
	// create one more.
	CanAddBranch(current int) bool
	// CanActivateService reports whether a business with currentActive active
	// services may create or activate one more.
	CanActivateService(currentActive int) bool
}

// EntitlementsProvider resolves the entitlements of a business. The plan behind
// them is the billing module's business; this module only asks questions.
type EntitlementsProvider interface {
	For(ctx context.Context, businessID int64) (Entitlements, error)
}
