package main

import (
	"log/slog"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	searchhttp "github.com/joaquinBrassinne/MOVO/internal/search/adapters/http"
	searchpostgres "github.com/joaquinBrassinne/MOVO/internal/search/adapters/postgres"
	"github.com/joaquinBrassinne/MOVO/internal/search/app/queries"
)

// newSearchRoutes composes the search module: adapter, use case and HTTP.
func newSearchRoutes(log *slog.Logger, pool *pgxpool.Pool) func(chi.Router) {
	return searchhttp.New(queries.NewNearbyBranches(searchpostgres.NewBranches(pool)), log).Mount
}
