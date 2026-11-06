package search

import (
	"context"
	"errors"
)

// ErrBranchNotFound is returned when a branch does not exist or is not public. The
// two cases look the same from outside on purpose.
var ErrBranchNotFound = errors.New("search: branch not found")

// Contact is what the WhatsApp button needs about a public branch. The number is
// used to build the redirect and never leaves the server in a response body.
type Contact struct {
	BusinessID   int64
	BranchID     int64
	BusinessName string
	WhatsApp     string
	// ServiceID and ServiceName are set only when the asked service is active and
	// belongs to the business (and to this branch when it is tied to one).
	ServiceID   int64
	ServiceName string
}

// ContactFinder reads the contact of a visible branch. A zero serviceID asks for none.
// A service that does not qualify is not an error: the contact comes back without it.
type ContactFinder interface {
	ContactFor(ctx context.Context, branchID, serviceID int64) (Contact, error)
}
