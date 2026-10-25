package postgres

import (
	"context"
	"fmt"

	"github.com/joaquinBrassinne/MOVO/internal/search"
)

// featuredSQL keeps each featured business once, through its nearest branch inside
// the radius, and orders the businesses by a hash of the id and the rotation seed:
// the order is stable while the seed holds and changes when it moves, so exposure is
// shared among the Pro businesses. $4 is the seed. The spatial candidates come from a
// materialized CTE for the reason given at nearbySQL.
const featuredSQL = `
WITH origin AS (SELECT ST_SetSRID(ST_MakePoint($2::float8, $1::float8), 4326)::geography AS p),
     near AS MATERIALIZED (
         SELECT br.id FROM branches br, origin o WHERE ST_DWithin(br.location, o.p, $3::float8))
SELECT branch_id, business_id, business_name, business_slug, branch_name, address, city, reference,
       lat, lng, distance_m, rating_avg, rating_count, verified, featured, category_slug, category_name
FROM (
    SELECT DISTINCT ON (v.business_id)
           v.branch_id, v.business_id, v.business_name, v.business_slug, v.branch_name,
           v.address, v.city, v.reference,
           ST_Y(v.location::geometry) AS lat, ST_X(v.location::geometry) AS lng,
           ST_Distance(v.location, o.p) AS distance_m,
           v.rating_avg::float8 AS rating_avg, v.rating_count, v.verified_at IS NOT NULL AS verified, v.featured,
           c.slug AS category_slug, c.name AS category_name
    FROM near n
    JOIN visible_branches v ON v.branch_id = n.id
    JOIN categories c ON c.id = v.category_id, origin o
    WHERE v.featured
    ORDER BY v.business_id, ST_Distance(v.location, o.p), v.branch_id
) nearest
ORDER BY md5(business_id::text || $4::bigint::text), business_id
LIMIT $5`

// Featured returns up to q.Limit featured businesses within q.RadiusM, rotated by seed.
func (r *Branches) Featured(ctx context.Context, q search.Query, seed int64) ([]search.Branch, error) {
	rows, err := r.pool.Query(ctx, featuredSQL, q.Lat, q.Lng, float64(q.RadiusM), seed, q.Limit)
	if err != nil {
		return nil, fmt.Errorf("postgres: featured branches: %w", err)
	}
	return collect(rows)
}
