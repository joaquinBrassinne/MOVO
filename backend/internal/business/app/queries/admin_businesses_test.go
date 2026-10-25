package queries_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/joaquinBrassinne/MOVO/internal/business"
	"github.com/joaquinBrassinne/MOVO/internal/business/app/queries"
	"github.com/joaquinBrassinne/MOVO/internal/business/domain"
)

type spyDirectory struct {
	got  business.AdminFilter
	page business.AdminPage
}

func (s *spyDirectory) Find(context.Context, int64) (business.AdminRow, error) {
	panic("not used by the query")
}

func (s *spyDirectory) History(context.Context, int64) ([]business.AuditRecord, error) {
	panic("not used by the query")
}

func (s *spyDirectory) List(_ context.Context, filter business.AdminFilter) (business.AdminPage, error) {
	s.got = filter
	return s.page, nil
}

func TestListBusinessesAppliesTheDefaultLimitAndPassesTheFilter(t *testing.T) {
	t.Parallel()

	spy := &spyDirectory{}
	page, err := queries.NewListBusinesses(spy).Handle(t.Context(), business.AdminFilter{Status: domain.StatusPendingReview, Offset: 20})
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if spy.got.Limit != 50 || spy.got.Status != domain.StatusPendingReview || spy.got.Offset != 20 {
		t.Errorf("filter = %+v, want the default limit 50 with status and offset kept", spy.got)
	}
	if page.Rows == nil {
		t.Error("rows must be a non-nil empty list so JSON renders []")
	}
}

func TestListBusinessesRejectsInvalidFilters(t *testing.T) {
	t.Parallel()

	tests := map[string]business.AdminFilter{
		"unknown status":    {Status: "approved"},
		"negative limit":    {Limit: -1},
		"limit too large":   {Limit: 101},
		"negative offset":   {Offset: -1},
		"negative category": {CategoryID: -1},
		"search too long":   {Query: strings.Repeat("a", 101)},
	}
	for name, filter := range tests {
		t.Run(name, func(t *testing.T) {
			spy := &spyDirectory{}
			if _, err := queries.NewListBusinesses(spy).Handle(t.Context(), filter); !errors.Is(err, queries.ErrAdminFilterInvalid) {
				t.Fatalf("error = %v, want ErrAdminFilterInvalid", err)
			}
			if spy.got != (business.AdminFilter{}) {
				t.Error("an invalid filter must not reach the directory")
			}
		})
	}
}
