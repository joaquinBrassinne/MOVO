// Package postgres implements the search ports over PostgreSQL and PostGIS.
package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/joaquinBrassinne/MOVO/internal/search"
)

// Branches implements search.BranchFinder.
type Branches struct {
	pool *pgxpool.Pool
}

// NewBranches builds the finder over pool.
func NewBranches(pool *pgxpool.Pool) *Branches {
	return &Branches{pool: pool}
}

// nearbySQL orders by distance band, then featured, then exact distance. branch_id
// breaks ties so pages are stable. $5 is a category slug, empty for all categories.
const nearbySQL = `
WITH origin AS (SELECT ST_SetSRID(ST_MakePoint($2::float8, $1::float8), 4326)::geography AS p)
SELECT v.branch_id, v.business_id, v.business_name, v.business_slug, v.branch_name,
       v.address, v.city, v.reference,
       ST_Y(v.location::geometry), ST_X(v.location::geometry),
       ST_Distance(v.location, o.p) AS distance_m,
       v.rating_avg::float8, v.rating_count, v.verified_at IS NOT NULL, v.featured
FROM visible_branches v, origin o
WHERE ST_DWithin(v.location, o.p, $3::float8)
  AND ($5::text = '' OR v.category_id = (SELECT id FROM categories WHERE slug = $5::text))
ORDER BY floor(ST_Distance(v.location, o.p) / $4::float8), v.featured DESC,
         ST_Distance(v.location, o.p), v.branch_id
LIMIT $6 OFFSET $7`

// Nearby returns one page of visible branches within q.RadiusM of the point. The
// query is validated by the use case, not here, which lets it ask for one extra row.
func (r *Branches) Nearby(ctx context.Context, q search.Query) ([]search.Branch, error) {
	rows, err := r.pool.Query(ctx, nearbySQL,
		q.Lat, q.Lng, float64(q.RadiusM), float64(search.DistanceBandM), q.Category, q.Limit, q.Offset)
	if err != nil {
		return nil, fmt.Errorf("postgres: search nearby: %w", err)
	}
	defer rows.Close()

	branches := []search.Branch{}
	for rows.Next() {
		var b search.Branch
		if err := rows.Scan(&b.BranchID, &b.BusinessID, &b.BusinessName, &b.BusinessSlug, &b.BranchName,
			&b.Address, &b.City, &b.Reference, &b.Lat, &b.Lng, &b.DistanceM,
			&b.RatingAvg, &b.RatingCount, &b.Verified, &b.Featured); err != nil {
			return nil, fmt.Errorf("postgres: scan branch: %w", err)
		}
		branches = append(branches, b)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres: search nearby: %w", err)
	}
	return branches, nil
}
