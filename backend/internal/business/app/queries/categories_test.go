package queries_test

import (
	"context"
	"errors"
	"testing"

	"github.com/joaquinBrassinne/MOVO/internal/business/app/queries"
	"github.com/joaquinBrassinne/MOVO/internal/business/domain"
)

type stubCategories struct {
	list []domain.Category
	err  error
}

func (s stubCategories) List(context.Context) ([]domain.Category, error) { return s.list, s.err }

func TestListCategories(t *testing.T) {
	t.Parallel()

	got, err := queries.NewListCategories(stubCategories{list: []domain.Category{{ID: 1, Slug: "yoga", Name: "Yoga"}}}).Handle(t.Context())
	if err != nil || len(got) != 1 || got[0].Slug != "yoga" {
		t.Fatalf("Handle = %+v, %v", got, err)
	}

	empty, err := queries.NewListCategories(stubCategories{}).Handle(t.Context())
	if err != nil || empty == nil || len(empty) != 0 {
		t.Errorf("empty catalog = %#v, %v; want a non-nil empty list", empty, err)
	}

	boom := errors.New("database down")
	if _, err := queries.NewListCategories(stubCategories{err: boom}).Handle(t.Context()); !errors.Is(err, boom) {
		t.Errorf("error = %v, want the repository error", err)
	}
}
