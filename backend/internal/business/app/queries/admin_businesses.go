package queries

import (
	"context"
	"errors"
	"fmt"
	"unicode/utf8"

	"github.com/joaquinBrassinne/MOVO/internal/business"
	"github.com/joaquinBrassinne/MOVO/internal/business/domain"
)

// Paging bounds of the super admin list.
const (
	defaultAdminLimit = 50
	maxAdminLimit     = 100
	maxAdminQueryLen  = 100
)

// ErrAdminFilterInvalid means the status or the paging of the list is not valid.
var ErrAdminFilterInvalid = errors.New("business: admin list filter is not valid")

// ListBusinesses reads the super admin list of businesses.
type ListBusinesses struct {
	directory business.AdminDirectory
}

// NewListBusinesses wires the ListBusinesses query.
func NewListBusinesses(directory business.AdminDirectory) *ListBusinesses {
	return &ListBusinesses{directory: directory}
}

// Handle validates the filter and returns one page. A zero Limit means 50; more
// than 100 or a negative offset is ErrAdminFilterInvalid. The caller must already
// have checked the super_admin role.
func (q *ListBusinesses) Handle(ctx context.Context, filter business.AdminFilter) (business.AdminPage, error) {
	switch filter.Status {
	case "", domain.StatusDraft, domain.StatusPendingReview, domain.StatusActive, domain.StatusPaused, domain.StatusDeleted:
	default:
		return business.AdminPage{}, ErrAdminFilterInvalid
	}
	if filter.Limit == 0 {
		filter.Limit = defaultAdminLimit
	}
	if filter.Limit < 0 || filter.Limit > maxAdminLimit || filter.Offset < 0 ||
		filter.CategoryID < 0 || utf8.RuneCountInString(filter.Query) > maxAdminQueryLen {
		return business.AdminPage{}, ErrAdminFilterInvalid
	}

	page, err := q.directory.List(ctx, filter)
	if err != nil {
		return business.AdminPage{}, fmt.Errorf("list businesses: %w", err)
	}
	if page.Rows == nil {
		page.Rows = []business.AdminRow{}
	}
	return page, nil
}
