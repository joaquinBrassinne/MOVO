package search

import (
	"context"
	"errors"
)

// ErrBusinessNotFound is returned when no business has the slug or none of its
// branches is public. The two cases look the same from outside on purpose.
var ErrBusinessNotFound = errors.New("search: business not found")

// Slot is an opening or class time of one weekday (0 = Sunday .. 6 = Saturday), as
// "HH:MM" strings.
type Slot struct {
	Weekday int
	Start   string
	End     string
}

// ProfileBranch is a public branch of the profile with its opening hours.
type ProfileBranch struct {
	BranchID    int64
	Name        string
	Address     string
	City        string
	Reference   string
	Lat, Lng    float64
	RatingAvg   float64
	RatingCount int
	Hours       []Slot
}

// ProfileService is an active service. BranchID is zero when the whole business
// offers it.
type ProfileService struct {
	ID          int64
	BranchID    int64
	Name        string
	Description string
	Price       int
	Schedule    []Slot
}

// Profile is the public page of a business: only its public branches and active
// services. It carries no contact data; the WhatsApp number stays on the server.
type Profile struct {
	BusinessID   int64
	Name         string
	Slug         string
	Description  string
	CategorySlug string
	CategoryName string
	Verified     bool
	Branches     []ProfileBranch
	Services     []ProfileService
	// ImageKeys are the object keys of the photos, in the order the owner chose.
	ImageKeys []string
}

// ProfileFinder reads the public profile of a business by slug. It returns
// ErrBusinessNotFound when the business is not public.
type ProfileFinder interface {
	Profile(ctx context.Context, slug string) (Profile, error)
}

// ImageURLs turns a stored object key into a short-lived URL to show the photo.
type ImageURLs interface {
	PresignDownload(ctx context.Context, key string) (string, error)
}

// BusinessResolver finds the business behind a public slug.
type BusinessResolver interface {
	// PublicBusinessID returns ErrBusinessNotFound when no business has the slug or
	// none of its branches is public.
	PublicBusinessID(ctx context.Context, slug string) (int64, error)
}
