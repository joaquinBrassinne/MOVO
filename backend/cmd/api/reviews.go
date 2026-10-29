package main

import (
	"context"
	"errors"
	"log/slog"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/joaquinBrassinne/MOVO/internal/business"
	businesspostgres "github.com/joaquinBrassinne/MOVO/internal/business/adapters/postgres"
	"github.com/joaquinBrassinne/MOVO/internal/platform/httpx"
	"github.com/joaquinBrassinne/MOVO/internal/reviews"
	reviewshttp "github.com/joaquinBrassinne/MOVO/internal/reviews/adapters/http"
	reviewspostgres "github.com/joaquinBrassinne/MOVO/internal/reviews/adapters/postgres"
	"github.com/joaquinBrassinne/MOVO/internal/reviews/app/commands"
	"github.com/joaquinBrassinne/MOVO/internal/reviews/app/queries"
)

// reviewOwners answers which business an owner may act on. It joins the two modules here
// so reviews never reads businesses.
type reviewOwners struct {
	businesses *businesspostgres.Businesses
}

func (o reviewOwners) EditableBusinessOf(ctx context.Context, ownerID int64) (int64, error) {
	b, err := o.businesses.FindByOwner(ctx, ownerID)
	if errors.Is(err, business.ErrBusinessNotFound) {
		return 0, reviews.ErrBusinessNotFound
	}
	if err != nil {
		return 0, err
	}
	if !b.Editable() {
		return 0, reviews.ErrBusinessNotEditable
	}
	return b.ID, nil
}

// newReviewsRoutes composes the reviews module: adapters, use cases and HTTP.
func newReviewsRoutes(log *slog.Logger, pool *pgxpool.Pool, verifier httpx.TokenVerifier) func(chi.Router) {
	repo := reviewspostgres.NewReviews(pool)
	owners := reviewOwners{businesses: businesspostgres.NewBusinesses(pool)}
	return reviewshttp.New(reviewshttp.Deps{
		Log:       log,
		Verifier:  verifier,
		Create:    commands.NewCreateReview(repo, repo, nil),
		List:      queries.NewListReviews(repo),
		Report:    commands.NewReportReview(owners, repo),
		SetStatus: commands.NewSetReviewStatus(repo),
		ListOwn:   queries.NewListOwnReviews(owners, repo),
		ListQueue: queries.NewListReviewsToModerate(repo),
	}).Mount
}
