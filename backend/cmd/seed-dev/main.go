// Command seed-dev loads a few public businesses into a local database so the views of
// the search (Home, Mapa, business page) have data to show. It is a development tool:
// it refuses to run against a database that is not on this machine or the compose host.
package main

import (
	"context"
	_ "embed"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"os/signal"

	"github.com/jackc/pgx/v5"

	identitydomain "github.com/joaquinBrassinne/MOVO/internal/identity/domain"
)

// The owner of the first test business signs in with these, to try the owner panel.
const (
	demoOwnerEmail = "alma-haus-training-club@seed.movo.test"
	demoPassword   = "movo-demo-1234"
)

//go:embed seed.sql
var seedSQL string

//go:embed seed_coupons.sql
var seedCouponsSQL string

//go:embed seed_reviews.sql
var seedReviewsSQL string

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	if err := run(ctx, os.Getenv("DATABASE_URL")); err != nil {
		slog.Error("seed-dev failed", "error", err)
		os.Exit(1)
	}
	slog.Info("seed-dev finished", "owner_email", demoOwnerEmail, "owner_password", demoPassword)
}

func run(ctx context.Context, databaseURL string) error {
	if databaseURL == "" {
		return fmt.Errorf("DATABASE_URL is not set")
	}
	if err := requireLocal(databaseURL); err != nil {
		return err
	}

	conn, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer func() { _ = conn.Close(ctx) }()

	for _, script := range []string{seedSQL, seedCouponsSQL, seedReviewsSQL} {
		if _, err := conn.Exec(ctx, script); err != nil {
			return fmt.Errorf("seed: %w", err)
		}
	}

	// One owner can sign in, to try the panel: the first test business.
	hash, err := identitydomain.HashPassword(demoPassword)
	if err != nil {
		return fmt.Errorf("hash demo password: %w", err)
	}
	if _, err := conn.Exec(ctx, `UPDATE users SET password_hash = $1 WHERE email = $2`, hash, demoOwnerEmail); err != nil {
		return fmt.Errorf("set demo owner password: %w", err)
	}
	return nil
}

// requireLocal keeps test businesses out of a real database.
func requireLocal(databaseURL string) error {
	u, err := url.Parse(databaseURL)
	if err != nil {
		return fmt.Errorf("parse DATABASE_URL: %w", err)
	}
	switch u.Hostname() {
	case "localhost", "127.0.0.1", "::1", "postgres":
		return nil
	}
	return fmt.Errorf("refusing to seed %q: seed-dev only runs against a local database", u.Hostname())
}
