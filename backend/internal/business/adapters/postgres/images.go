package postgres

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/joaquinBrassinne/MOVO/internal/business"
	"github.com/joaquinBrassinne/MOVO/internal/business/domain"
)

// Images implements business.ImageRepository. Writes lock the business row first,
// so concurrent uploads of one business queue up and the cap and the contiguous
// positions hold.
type Images struct {
	pool *pgxpool.Pool
}

// NewImages builds the repository over pool.
func NewImages(pool *pgxpool.Pool) *Images {
	return &Images{pool: pool}
}

// lock serializes the writers of one business and returns when it is theirs.
func lock(ctx context.Context, tx pgx.Tx, businessID int64) error {
	var id int64
	err := tx.QueryRow(ctx, `SELECT id FROM businesses WHERE id = $1 FOR UPDATE`, businessID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return business.ErrBusinessNotFound
	}
	if err != nil {
		return fmt.Errorf("postgres: lock business: %w", err)
	}
	return nil
}

// Add registers key at the end of the list.
func (r *Images) Add(ctx context.Context, businessID int64, key string, max int) (domain.Image, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.Image{}, fmt.Errorf("postgres: begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := lock(ctx, tx, businessID); err != nil {
		return domain.Image{}, err
	}

	var count int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM business_images WHERE business_id = $1`, businessID).Scan(&count); err != nil {
		return domain.Image{}, fmt.Errorf("postgres: count images: %w", err)
	}
	if count >= max {
		return domain.Image{}, business.ErrImageLimitReached
	}

	image := domain.Image{BusinessID: businessID, ObjectKey: key, Position: count}
	err = tx.QueryRow(ctx,
		`INSERT INTO business_images (business_id, object_key, position) VALUES ($1, $2, $3) RETURNING id`,
		businessID, key, count).Scan(&image.ID)

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation && pgErr.ConstraintName == "business_images_object_key_key" {
		return domain.Image{}, business.ErrImageExists
	}
	if err != nil {
		return domain.Image{}, fmt.Errorf("postgres: insert image: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.Image{}, fmt.Errorf("postgres: commit: %w", err)
	}
	return image, nil
}

// List returns the images ordered by position.
func (r *Images) List(ctx context.Context, businessID int64) ([]domain.Image, error) {
	return listImages(ctx, r.pool, businessID)
}

type querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

func listImages(ctx context.Context, q querier, businessID int64) ([]domain.Image, error) {
	rows, err := q.Query(ctx,
		`SELECT id, business_id, object_key, position FROM business_images WHERE business_id = $1 ORDER BY position`, businessID)
	if err != nil {
		return nil, fmt.Errorf("postgres: list images: %w", err)
	}
	defer rows.Close()

	images := []domain.Image{}
	for rows.Next() {
		var img domain.Image
		if err := rows.Scan(&img.ID, &img.BusinessID, &img.ObjectKey, &img.Position); err != nil {
			return nil, fmt.Errorf("postgres: scan image: %w", err)
		}
		images = append(images, img)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres: list images: %w", err)
	}
	return images, nil
}

// Remove deletes the image and closes the gap it leaves in the positions.
func (r *Images) Remove(ctx context.Context, businessID, imageID int64) (domain.Image, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.Image{}, fmt.Errorf("postgres: begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := lock(ctx, tx, businessID); err != nil {
		if errors.Is(err, business.ErrBusinessNotFound) {
			return domain.Image{}, business.ErrImageNotFound
		}
		return domain.Image{}, err
	}

	removed := domain.Image{BusinessID: businessID}
	err = tx.QueryRow(ctx,
		`DELETE FROM business_images WHERE id = $1 AND business_id = $2 RETURNING id, object_key, position`,
		imageID, businessID).Scan(&removed.ID, &removed.ObjectKey, &removed.Position)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Image{}, business.ErrImageNotFound
	}
	if err != nil {
		return domain.Image{}, fmt.Errorf("postgres: delete image: %w", err)
	}

	if _, err := tx.Exec(ctx,
		`UPDATE business_images SET position = position - 1, updated_at = now() WHERE business_id = $1 AND position > $2`,
		businessID, removed.Position); err != nil {
		return domain.Image{}, fmt.Errorf("postgres: close position gap: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.Image{}, fmt.Errorf("postgres: commit: %w", err)
	}
	return removed, nil
}

// Reorder sets the positions from the order of ids, which must be exactly the
// ids of the business images. The unique (business_id, position) constraint is
// deferred, so the intermediate states of the update are not checked.
func (r *Images) Reorder(ctx context.Context, businessID int64, ids []int64) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("postgres: begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := lock(ctx, tx, businessID); err != nil {
		if errors.Is(err, business.ErrBusinessNotFound) {
			return business.ErrImageOrderInvalid
		}
		return err
	}

	current, err := listImages(ctx, tx, businessID)
	if err != nil {
		return err
	}
	have := make([]int64, 0, len(current))
	for _, img := range current {
		have = append(have, img.ID)
	}
	want := slices.Clone(ids)
	slices.Sort(have)
	slices.Sort(want)
	if len(want) == 0 || !slices.Equal(have, want) {
		return business.ErrImageOrderInvalid
	}

	for position, id := range ids {
		if _, err := tx.Exec(ctx,
			`UPDATE business_images SET position = $3, updated_at = now() WHERE id = $1 AND business_id = $2`,
			id, businessID, position); err != nil {
			return fmt.Errorf("postgres: set position: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("postgres: commit: %w", err)
	}
	return nil
}
