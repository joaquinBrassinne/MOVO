package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/joaquinBrassinne/MOVO/internal/business/domain"
)

// Categories implements business.CategoryRepository.
type Categories struct {
	pool *pgxpool.Pool
}

// NewCategories builds the repository over pool.
func NewCategories(pool *pgxpool.Pool) *Categories {
	return &Categories{pool: pool}
}

// List returns the catalog in insertion order, which is the order the seed gives.
func (r *Categories) List(ctx context.Context) ([]domain.Category, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, slug, name FROM categories ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("postgres: list categories: %w", err)
	}
	defer rows.Close()

	categories := []domain.Category{}
	for rows.Next() {
		var c domain.Category
		if err := rows.Scan(&c.ID, &c.Slug, &c.Name); err != nil {
			return nil, fmt.Errorf("postgres: scan category: %w", err)
		}
		categories = append(categories, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres: list categories: %w", err)
	}
	return categories, nil
}
