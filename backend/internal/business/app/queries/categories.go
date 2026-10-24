package queries

import (
	"context"
	"fmt"

	"github.com/joaquinBrassinne/MOVO/internal/business"
	"github.com/joaquinBrassinne/MOVO/internal/business/domain"
)

// ListCategories reads the catalog the onboarding form offers.
type ListCategories struct {
	categories business.CategoryRepository
}

// NewListCategories wires the ListCategories query.
func NewListCategories(categories business.CategoryRepository) *ListCategories {
	return &ListCategories{categories: categories}
}

// Handle returns the categories; the list is never nil so JSON renders [].
func (q *ListCategories) Handle(ctx context.Context) ([]domain.Category, error) {
	list, err := q.categories.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list categories: %w", err)
	}
	if list == nil {
		list = []domain.Category{}
	}
	return list, nil
}
