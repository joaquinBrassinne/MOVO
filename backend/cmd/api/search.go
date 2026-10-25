package main

import (
	"log/slog"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	searchhttp "github.com/joaquinBrassinne/MOVO/internal/search/adapters/http"
	searchpostgres "github.com/joaquinBrassinne/MOVO/internal/search/adapters/postgres"
	"github.com/joaquinBrassinne/MOVO/internal/search/app/commands"
	"github.com/joaquinBrassinne/MOVO/internal/search/app/queries"
)

// newSearchRoutes composes the search module: adapters, use cases and HTTP.
func newSearchRoutes(log *slog.Logger, pool *pgxpool.Pool) func(chi.Router) {
	branches := searchpostgres.NewBranches(pool)
	record := commands.NewRecordMetric(searchpostgres.NewMetrics(pool), nil)
	return searchhttp.New(queries.NewNearbyBranches(branches), queries.NewFeaturedBranches(branches, nil), log).
		WithWhatsApp(commands.NewOpenWhatsApp(branches, record, log)).Mount
}
