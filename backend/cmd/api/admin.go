package main

import (
	"log/slog"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	adminhttp "github.com/joaquinBrassinne/MOVO/internal/admin/adapters/http"
	adminpostgres "github.com/joaquinBrassinne/MOVO/internal/admin/adapters/postgres"
	"github.com/joaquinBrassinne/MOVO/internal/admin/app/queries"
	"github.com/joaquinBrassinne/MOVO/internal/platform/httpx"
)

// newAdminRoutes composes the super admin's read side: adapters, use cases and HTTP.
func newAdminRoutes(log *slog.Logger, pool *pgxpool.Pool, verifier httpx.TokenVerifier) func(chi.Router) {
	return adminhttp.New(adminhttp.Deps{
		Log:      log,
		Verifier: verifier,
		Overview: queries.NewGetOverview(adminpostgres.NewOverview(pool), nil),
	}).Mount
}
