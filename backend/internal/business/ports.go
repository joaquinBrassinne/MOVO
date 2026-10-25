// Package business is the module that lets an owner describe a business, its
// branches and opening hours, and that moves the business through its review.
// This file declares the ports the application layer consumes; adapters
// implement them.
package business

import (
	"context"
	"errors"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/business/domain"
	"github.com/joaquinBrassinne/MOVO/internal/platform/storage"
)

// Errors the adapters report so the application layer can react without knowing
// the storage engine.
var (
	ErrBusinessNotFound = errors.New("business: business not found")
	ErrOwnerHasBusiness = errors.New("business: owner already has a business")
	ErrSlugTaken        = errors.New("business: slug is already taken")
	ErrCategoryNotFound = errors.New("business: category not found")
	ErrBranchNotFound   = errors.New("business: branch not found")

	ErrAddressNotFound     = errors.New("business: address could not be located")
	ErrGeocoderBusy        = errors.New("business: geocoder is busy")
	ErrGeocoderUnavailable = errors.New("business: geocoder is not available")
	ErrBranchLimitReached  = errors.New("business: branch limit of the plan reached")

	ErrServiceNotFound     = errors.New("business: service not found")
	ErrServiceLimitReached = errors.New("business: active service limit of the plan reached")

	ErrImageNotFound     = errors.New("business: image not found")
	ErrImageLimitReached = errors.New("business: image limit reached")
	ErrImageExists       = errors.New("business: image is already registered")
	ErrImageNotUploaded  = errors.New("business: image was not uploaded")
	ErrImageOrderInvalid = errors.New("business: image order must list every image once")
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

// ServiceRepository stores the services of a business and their weekly
// schedules. Like BranchRepository, every method is scoped by business id.
type ServiceRepository interface {
	// Create stores a new service and returns it with ID set.
	Create(ctx context.Context, s domain.Service) (domain.Service, error)
	// List returns every service of the business, active or not, oldest first.
	List(ctx context.Context, businessID int64) ([]domain.Service, error)
	// Find returns ErrServiceNotFound when the service is not in the business.
	Find(ctx context.Context, businessID, serviceID int64) (domain.Service, error)
	// Save persists the details and the active flag (ErrServiceNotFound).
	Save(ctx context.Context, s domain.Service) error
	// Delete removes the service with its schedule (ErrServiceNotFound).
	Delete(ctx context.Context, businessID, serviceID int64) error
	// Schedule returns the weekly slots ordered by weekday and start.
	Schedule(ctx context.Context, businessID, serviceID int64) ([]domain.Slot, error)
	// ReplaceSchedule atomically swaps all the slots (ErrServiceNotFound).
	ReplaceSchedule(ctx context.Context, businessID, serviceID int64, slots []domain.Slot) error
}

// ErrStatusConflict means the business changed between reading it and writing the
// new state, so the action was not applied.
var ErrStatusConflict = errors.New("business: business was changed by someone else")

// AuditEntry is one line of the audit trail.
type AuditEntry struct {
	ActorID  int64
	Action   string
	Entity   string
	EntityID int64
	Metadata map[string]any
}

// AdminRepository is what the super admin actions need. It reads businesses by id
// (the owner-facing BusinessRepository reads them by owner) and writes a state
// change together with its audit entry.
type AdminRepository interface {
	// FindByID returns ErrBusinessNotFound when there is no such business.
	FindByID(ctx context.Context, id int64) (domain.Business, error)
	// Apply stores the status, pilot flag and verification of b and appends entry,
	// in one transaction. previous is the business as it was read: when the stored
	// status, pilot flag or verification differ it returns ErrStatusConflict and
	// writes nothing, so two admins cannot overwrite each other.
	Apply(ctx context.Context, b, previous domain.Business, entry AuditEntry) error
}

// AdminFilter selects which businesses the super admin list shows. An empty
// Status means every business except the deleted ones. Query searches the name, the
// slug, the owner email and the branch addresses, or matches an exact business id.
type AdminFilter struct {
	Status     domain.Status
	Query      string
	CategoryID int64
	Limit      int
	Offset     int
}

// AdminRow is one line of the super admin list.
type AdminRow struct {
	Business     domain.Business
	OwnerEmail   string
	CategoryName string
	BranchCount  int
}

// AdminPage is a page of the list with the total of businesses that match. Counts is
// how many businesses each status has in the whole directory, whatever the filters
// say: it feeds the tabs of the list.
type AdminPage struct {
	Rows   []AdminRow
	Total  int
	Counts map[domain.Status]int
}

// AuditRecord is one entry of the history of a business, as the super admin reads it.
type AuditRecord struct {
	ActorEmail string
	Action     string
	At         time.Time
	Metadata   map[string]any
}

// AdminDirectory lists businesses for the super admin, those waiting for review
// first and then the most recent.
type AdminDirectory interface {
	List(ctx context.Context, filter AdminFilter) (AdminPage, error)
	// Find returns one business as a list row (ErrBusinessNotFound).
	Find(ctx context.Context, id int64) (AdminRow, error)
	// History returns the audit entries of the business, newest first.
	History(ctx context.Context, id int64) ([]AuditRecord, error)
}

// ImageStorage moves photos through presigned URLs. The object server behind it
// is an environment detail (see docs/adr/0002).
type ImageStorage interface {
	PresignUpload(ctx context.Context, req storage.UploadRequest) (storage.PresignedUpload, error)
	PresignDownload(ctx context.Context, key string) (string, error)
	Exists(ctx context.Context, key string) (bool, error)
	Delete(ctx context.Context, key string) error
}

// ImageRepository stores the photos of a business. Position 0 is the cover and
// positions are always contiguous.
type ImageRepository interface {
	// Add registers key at the end of the list. It returns ErrImageLimitReached when
	// the business already has max images and ErrImageExists when the key is taken.
	Add(ctx context.Context, businessID int64, key string, max int) (domain.Image, error)
	// List returns the images ordered by position.
	List(ctx context.Context, businessID int64) ([]domain.Image, error)
	// Remove deletes the image, closes the gap in the positions and returns what it
	// removed so the object can be deleted (ErrImageNotFound).
	Remove(ctx context.Context, businessID, imageID int64) (domain.Image, error)
	// Reorder sets the positions from the order of ids, which must be exactly the
	// ids of the business images (ErrImageOrderInvalid).
	Reorder(ctx context.Context, businessID int64, ids []int64) error
}

// CategoryRepository reads the category catalog.
type CategoryRepository interface {
	// List returns every category in catalog order.
	List(ctx context.Context) ([]domain.Category, error)
}

// Place is a located address.
type Place struct {
	Latitude    float64
	Longitude   float64
	DisplayName string
}

// Geocoder turns a typed address into coordinates. The real service is a public
// server with a strict usage policy (see docs/sdd.md §12.3), so it sits behind this
// port and tests use a fake.
type Geocoder interface {
	// Geocode returns ErrAddressNotFound when nothing matches, ErrGeocoderBusy when
	// the request budget is spent and ErrGeocoderUnavailable when the service fails.
	Geocode(ctx context.Context, query string) (Place, error)
}

// ImageURLs turns a stored object key into a short-lived URL to show the photo.
type ImageURLs interface {
	PresignDownload(ctx context.Context, key string) (string, error)
}
