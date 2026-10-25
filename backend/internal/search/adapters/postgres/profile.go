package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/joaquinBrassinne/MOVO/internal/search"
)

// The branches come from visible_branches, so a business with no public branch has no
// profile. Hours, services and photos are then read for exactly those branches.
const (
	profileBranchesSQL = `
SELECT v.business_id, v.business_name, v.business_slug, b.description, c.slug, c.name,
       v.verified_at IS NOT NULL,
       v.branch_id, v.branch_name, v.address, v.city, v.reference, COALESCE(br.external_booking_url, ''),
       ST_Y(v.location::geometry), ST_X(v.location::geometry),
       v.rating_avg::float8, v.rating_count
FROM visible_branches v
JOIN businesses b ON b.id = v.business_id
JOIN categories c ON c.id = b.category_id
JOIN branches br ON br.id = v.branch_id
WHERE v.business_slug = $1
ORDER BY v.branch_id`

	profileHoursSQL = `
SELECT branch_id, weekday, to_char(opens_at, 'HH24:MI'), to_char(closes_at, 'HH24:MI')
FROM business_hours
WHERE branch_id = ANY($1)
ORDER BY branch_id, weekday, opens_at`

	// A service tied to a branch that is not public stays out with it.
	profileServicesSQL = `
SELECT id, COALESCE(branch_id, 0), name, description, COALESCE(price, 0)
FROM services
WHERE business_id = $1 AND active AND (branch_id IS NULL OR branch_id = ANY($2))
ORDER BY id`

	profileSchedulesSQL = `
SELECT service_id, weekday, to_char(starts_at, 'HH24:MI'), to_char(ends_at, 'HH24:MI')
FROM service_schedules
WHERE service_id = ANY($1)
ORDER BY service_id, weekday, starts_at`

	profileImagesSQL = `SELECT object_key FROM business_images WHERE business_id = $1 ORDER BY position`
)

// Profile returns the public profile of the business, or search.ErrBusinessNotFound.
func (r *Branches) Profile(ctx context.Context, slug string) (search.Profile, error) {
	// One snapshot, so the branches, hours and services belong to the same moment.
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return search.Profile{}, fmt.Errorf("postgres: begin profile read: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	p, err := readProfileBranches(ctx, tx, slug)
	if err != nil {
		return search.Profile{}, err
	}
	if err := readProfileHours(ctx, tx, &p); err != nil {
		return search.Profile{}, err
	}
	if err := readProfileServices(ctx, tx, &p); err != nil {
		return search.Profile{}, err
	}
	if err := readProfileImages(ctx, tx, &p); err != nil {
		return search.Profile{}, err
	}
	return p, nil
}

func readProfileBranches(ctx context.Context, tx pgx.Tx, slug string) (search.Profile, error) {
	rows, err := tx.Query(ctx, profileBranchesSQL, slug)
	if err != nil {
		return search.Profile{}, fmt.Errorf("postgres: profile branches of %q: %w", slug, err)
	}
	defer rows.Close()

	var p search.Profile
	for rows.Next() {
		var b search.ProfileBranch
		if err := rows.Scan(&p.BusinessID, &p.Name, &p.Slug, &p.Description, &p.CategorySlug, &p.CategoryName,
			&p.Verified, &b.BranchID, &b.Name, &b.Address, &b.City, &b.Reference, &b.BookingURL,
			&b.Lat, &b.Lng, &b.RatingAvg, &b.RatingCount); err != nil {
			return search.Profile{}, fmt.Errorf("postgres: scan profile branch: %w", err)
		}
		p.Branches = append(p.Branches, b)
	}
	if err := rows.Err(); err != nil {
		return search.Profile{}, fmt.Errorf("postgres: read profile branches: %w", err)
	}
	if len(p.Branches) == 0 {
		return search.Profile{}, search.ErrBusinessNotFound
	}
	return p, nil
}

func readProfileHours(ctx context.Context, tx pgx.Tx, p *search.Profile) error {
	ids := make([]int64, len(p.Branches))
	index := make(map[int64]int, len(p.Branches))
	for i, b := range p.Branches {
		ids[i], index[b.BranchID] = b.BranchID, i
	}
	rows, err := tx.Query(ctx, profileHoursSQL, ids)
	if err != nil {
		return fmt.Errorf("postgres: profile hours: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var branch int64
		var s search.Slot
		if err := rows.Scan(&branch, &s.Weekday, &s.Start, &s.End); err != nil {
			return fmt.Errorf("postgres: scan profile hours: %w", err)
		}
		b := &p.Branches[index[branch]]
		b.Hours = append(b.Hours, s)
	}
	return rows.Err()
}

func readProfileServices(ctx context.Context, tx pgx.Tx, p *search.Profile) error {
	branchIDs := make([]int64, len(p.Branches))
	for i, b := range p.Branches {
		branchIDs[i] = b.BranchID
	}
	rows, err := tx.Query(ctx, profileServicesSQL, p.BusinessID, branchIDs)
	if err != nil {
		return fmt.Errorf("postgres: profile services: %w", err)
	}
	defer rows.Close()

	var ids []int64
	index := map[int64]int{}
	for rows.Next() {
		var s search.ProfileService
		if err := rows.Scan(&s.ID, &s.BranchID, &s.Name, &s.Description, &s.Price); err != nil {
			return fmt.Errorf("postgres: scan profile service: %w", err)
		}
		index[s.ID] = len(p.Services)
		ids = append(ids, s.ID)
		p.Services = append(p.Services, s)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("postgres: read profile services: %w", err)
	}
	rows.Close()
	if len(ids) == 0 {
		return nil
	}

	slots, err := tx.Query(ctx, profileSchedulesSQL, ids)
	if err != nil {
		return fmt.Errorf("postgres: profile schedules: %w", err)
	}
	defer slots.Close()
	for slots.Next() {
		var service int64
		var s search.Slot
		if err := slots.Scan(&service, &s.Weekday, &s.Start, &s.End); err != nil {
			return fmt.Errorf("postgres: scan profile schedule: %w", err)
		}
		svc := &p.Services[index[service]]
		svc.Schedule = append(svc.Schedule, s)
	}
	return slots.Err()
}

func readProfileImages(ctx context.Context, tx pgx.Tx, p *search.Profile) error {
	rows, err := tx.Query(ctx, profileImagesSQL, p.BusinessID)
	if err != nil {
		return fmt.Errorf("postgres: profile images: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return fmt.Errorf("postgres: scan profile image: %w", err)
		}
		p.ImageKeys = append(p.ImageKeys, key)
	}
	return rows.Err()
}

// PublicBusinessID returns the id of the business with that slug when it has a public
// branch, or search.ErrBusinessNotFound.
func (r *Branches) PublicBusinessID(ctx context.Context, slug string) (int64, error) {
	var id int64
	err := r.pool.QueryRow(ctx, `SELECT business_id FROM visible_branches WHERE business_slug = $1 LIMIT 1`, slug).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, search.ErrBusinessNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("postgres: public business %q: %w", slug, err)
	}
	return id, nil
}
