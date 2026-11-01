package main

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/joaquinBrassinne/MOVO/internal/billing/adapters/configcatalog"
	"github.com/joaquinBrassinne/MOVO/internal/billing/adapters/pilotplan"
	billingpostgres "github.com/joaquinBrassinne/MOVO/internal/billing/adapters/postgres"
	billingqueries "github.com/joaquinBrassinne/MOVO/internal/billing/app/queries"
	businesspostgres "github.com/joaquinBrassinne/MOVO/internal/business/adapters/postgres"
	"github.com/joaquinBrassinne/MOVO/internal/platform/config"
)

// proMetrics answers whether a business has the Pro metrics from its entitlements, the same
// way the API does: a pilot gets the capabilities of its pilot plan.
type proMetrics struct {
	query *billingqueries.EntitlementsFor
}

func (p proMetrics) AdvancedMetrics(ctx context.Context, businessID int64) (bool, error) {
	granted, err := p.query.Handle(ctx, businessID)
	if err != nil {
		return false, err
	}
	return granted.AdvancedMetrics(), nil
}

// newProMetrics builds the permission from the plan settings of the environment.
func newProMetrics(pool *pgxpool.Pool, plans config.Plans) (proMetrics, error) {
	catalog, err := configcatalog.New(plans)
	if err != nil {
		return proMetrics{}, err
	}
	resolver := pilotplan.New(businesspostgres.NewBusinesses(pool), billingpostgres.NewSubscriptions(pool))
	return proMetrics{query: billingqueries.NewEntitlementsFor(catalog, resolver)}, nil
}
