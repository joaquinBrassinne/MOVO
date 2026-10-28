package main

import (
	"log/slog"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	reviewshttp "github.com/joaquinBrassinne/MOVO/internal/reviews/adapters/http"
	reviewspostgres "github.com/joaquinBrassinne/MOVO/internal/reviews/adapters/postgres"
	"github.com/joaquinBrassinne/MOVO/internal/reviews/app/commands"
)

// newReviewsRoutes composes the reviews module: adapters, use cases and HTTP.
func newReviewsRoutes(log *slog.Logger, pool *pgxpool.Pool) func(chi.Router) {
	repo := reviewspostgres.NewReviews(pool)
	return reviewshttp.New(reviewshttp.Deps{
		Log:    log,
		Create: commands.NewCreateReview(repo, repo, nil),
	}).Mount
}
