package domain

import (
	"errors"
	"strings"
	"time"
	"unicode/utf8"
)

// Validation and lifecycle errors. The HTTP layer maps each one to a stable
// error code.
var (
	ErrNameInvalid        = errors.New("business: name is not valid")
	ErrDescriptionInvalid = errors.New("business: description is not valid")
	ErrCategoryRequired   = errors.New("business: category is required")
	ErrInvalidTransition  = errors.New("business: status transition is not allowed")
	ErrNotEditable        = errors.New("business: business can no longer be edited")
)

const (
	maxNameLen        = 100
	maxDescriptionLen = 1000
)

// Status is where a business is in its lifecycle.
type Status string

// The statuses of a business. Visibility in the public search additionally
// depends on the subscription and is computed when reading, not stored here.
const (
	StatusDraft         Status = "draft"
	StatusPendingReview Status = "pending_review"
	StatusActive        Status = "active"
	StatusPaused        Status = "paused"
	StatusDeleted       Status = "deleted"
)

// Business is the single business of an owner.
type Business struct {
	ID               int64
	OwnerID          int64
	CategoryID       int64
	Name             string
	Slug             string
	Description      string
	WhatsApp         string
	Status           Status
	Pilot            bool
	VerifiedAt       *time.Time
	VerifiedBy       int64
	PaymentReference string
	CreatedAt        time.Time
}

// NewBusinessInput is what an owner provides to create a business.
type NewBusinessInput struct {
	OwnerID     int64
	CategoryID  int64
	Name        string
	Description string
	WhatsApp    string
}

// ProfileInput is what an owner provides to edit the business profile.
type ProfileInput struct {
	CategoryID  int64
	Name        string
	Description string
	WhatsApp    string
}

// NewBusiness validates the input and builds a draft ready to be stored: text is
// trimmed, the WhatsApp number is normalized and the slug is derived from the
// name. Making the slug unique is the repository's job (see SlugWithSuffix).
func NewBusiness(in NewBusinessInput) (Business, error) {
	name, description, err := validateProfile(in.Name, in.Description)
	if err != nil {
		return Business{}, err
	}
	if in.CategoryID <= 0 {
		return Business{}, ErrCategoryRequired
	}
	slug, err := Slugify(name)
	if err != nil {
		return Business{}, err
	}
	whatsapp, err := NormalizeWhatsApp(in.WhatsApp)
	if err != nil {
		return Business{}, err
	}

	return Business{
		OwnerID:     in.OwnerID,
		CategoryID:  in.CategoryID,
		Name:        name,
		Slug:        slug,
		Description: description,
		WhatsApp:    whatsapp,
		Status:      StatusDraft,
	}, nil
}

// UpdateProfile validates the input and returns the business with the new
// profile. The slug is kept: public URLs must not change when a name does.
func (b Business) UpdateProfile(in ProfileInput) (Business, error) {
	if !b.Editable() {
		return Business{}, ErrNotEditable
	}
	name, description, err := validateProfile(in.Name, in.Description)
	if err != nil {
		return Business{}, err
	}
	if in.CategoryID <= 0 {
		return Business{}, ErrCategoryRequired
	}
	whatsapp, err := NormalizeWhatsApp(in.WhatsApp)
	if err != nil {
		return Business{}, err
	}

	b.CategoryID, b.Name, b.Description, b.WhatsApp = in.CategoryID, name, description, whatsapp
	return b, nil
}

func validateProfile(name, description string) (string, string, error) {
	name = strings.TrimSpace(name)
	if name == "" || utf8.RuneCountInString(name) > maxNameLen {
		return "", "", ErrNameInvalid
	}
	description = strings.TrimSpace(description)
	if utf8.RuneCountInString(description) > maxDescriptionLen {
		return "", "", ErrDescriptionInvalid
	}
	return name, description, nil
}

// Editable reports whether the owner may still change the business.
func (b Business) Editable() bool { return b.Status != StatusDeleted }

// Verified reports whether a super admin vouched for the business identity.
func (b Business) Verified() bool { return b.VerifiedAt != nil }

// Submit sends a draft to the manual review.
func (b Business) Submit() (Business, error) {
	return b.transition(StatusPendingReview, StatusDraft)
}

// Approve activates a business that is waiting for review.
func (b Business) Approve() (Business, error) {
	return b.transition(StatusActive, StatusPendingReview)
}

// Pause takes an active business out of circulation.
func (b Business) Pause() (Business, error) {
	return b.transition(StatusPaused, StatusActive)
}

// Reactivate brings a paused business back.
func (b Business) Reactivate() (Business, error) {
	return b.transition(StatusActive, StatusPaused)
}

// Delete is the logical removal; it is allowed from any status but deleted.
func (b Business) Delete() (Business, error) {
	return b.transition(StatusDeleted, StatusDraft, StatusPendingReview, StatusActive, StatusPaused)
}

// MarkVerified records who vouched for the business and when.
func (b Business) MarkVerified(by int64) Business {
	now := time.Now().UTC()
	b.VerifiedAt = &now
	b.VerifiedBy = by
	return b
}

// RevokeVerification removes the verified seal.
func (b Business) RevokeVerification() Business {
	b.VerifiedAt = nil
	b.VerifiedBy = 0
	return b
}

func (b Business) transition(to Status, from ...Status) (Business, error) {
	for _, allowed := range from {
		if b.Status == allowed {
			b.Status = to
			return b, nil
		}
	}
	return Business{}, ErrInvalidTransition
}
