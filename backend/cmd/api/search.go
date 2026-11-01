package main

import (
	"log/slog"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/joaquinBrassinne/MOVO/internal/search"
	searchhttp "github.com/joaquinBrassinne/MOVO/internal/search/adapters/http"
	searchmemory "github.com/joaquinBrassinne/MOVO/internal/search/adapters/memory"
	searchpostgres "github.com/joaquinBrassinne/MOVO/internal/search/adapters/postgres"
	"github.com/joaquinBrassinne/MOVO/internal/search/app/commands"
	"github.com/joaquinBrassinne/MOVO/internal/search/app/queries"
)

// Anti-inflation of the metrics: a visit or click repeated from the same address within the
// window is not counted again. Thirty minutes ignores refreshes and bounces but counts the
// person who comes back later; the table is bounded so a flood cannot grow the memory.
const (
	metricDedupeWindow  = 30 * time.Minute
	metricDedupeMaxKeys = 100_000
)

// newSearchRoutes composes the search module: adapters, use cases and HTTP.
func newSearchRoutes(log *slog.Logger, pool *pgxpool.Pool, images search.ImageURLs) func(chi.Router) {
	branches := searchpostgres.NewBranches(pool)
	// The same visitor repeating an event within the window counts once (SDD 6.3b).
	record := commands.NewRecordMetric(searchpostgres.NewMetrics(pool), nil).
		WithDeduper(searchmemory.NewDeduper(metricDedupeWindow, metricDedupeMaxKeys, nil))
	return searchhttp.New(queries.NewNearbyBranches(branches), queries.NewFeaturedBranches(branches, nil), log).
		WithWhatsApp(commands.NewOpenWhatsApp(branches, record, log)).
		WithProfile(queries.NewBusinessProfile(branches, images, nil)).
		WithProfileView(commands.NewRecordProfileView(branches, record, log)).Mount
}
