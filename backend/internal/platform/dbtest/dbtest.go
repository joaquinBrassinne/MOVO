// Package dbtest gives integration tests a real PostgreSQL+PostGIS database with
// the embedded migrations applied.
//
// One container is shared by every test of a test binary; each call to New gets
// its own freshly migrated database, so tests stay independent and can run in
// parallel.
package dbtest

import (
	"context"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/joaquinBrassinne/MOVO/internal/platform/db"
)

// image matches deploy/docker-compose.yml so tests run on the same engine.
const image = "postgis/postgis:16-3.6"

const startTimeout = 2 * time.Minute

var (
	startOnce sync.Once
	container *postgres.PostgresContainer
	startErr  error
	nextDB    atomic.Int64
)

// New returns a pool connected to a new, fully migrated database. It skips the
// test when Docker is unavailable, except under CI, where that is a failure:
// a silently skipped integration suite would look like a green one.
func New(t *testing.T) *pgxpool.Pool {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), startTimeout)
	defer cancel()

	startOnce.Do(func() {
		container, startErr = postgres.Run(ctx, image,
			postgres.WithDatabase("postgres"),
			postgres.WithUsername("movo"),
			postgres.WithPassword("movo"),
			// Waits for the "ready to accept connections" log twice (the image restarts
			// itself after its first start) and then for the port. Do not add
			// testcontainers.WithWaitStrategy here: it replaces this one, and a bare port
			// check opens before Postgres is ready, so tests fail with SQLSTATE 57P03.
			postgres.BasicWaitStrategies(),
		)
	})
	if startErr != nil {
		if os.Getenv("CI") != "" {
			t.Fatalf("start postgres container: %v", startErr)
		}
		t.Skipf("docker is not available, skipping integration test: %v", startErr)
	}

	name := fmt.Sprintf("movo_test_%d", nextDB.Add(1))

	admin, err := connect(ctx, "postgres")
	if err != nil {
		t.Fatalf("connect to the admin database: %v", err)
	}
	defer admin.Close()

	if _, err := admin.Exec(ctx, "CREATE DATABASE "+name); err != nil {
		t.Fatalf("create database %s: %v", name, err)
	}

	pool, err := connect(ctx, name)
	if err != nil {
		t.Fatalf("connect to %s: %v", name, err)
	}
	t.Cleanup(pool.Close)

	if err := db.Migrate(ctx, pool, "up"); err != nil {
		t.Fatalf("migrate %s: %v", name, err)
	}

	return pool
}

func connect(ctx context.Context, database string) (*pgxpool.Pool, error) {
	url, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return nil, fmt.Errorf("connection string: %w", err)
	}

	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, fmt.Errorf("parse connection string: %w", err)
	}
	cfg.ConnConfig.Database = database

	return pgxpool.NewWithConfig(ctx, cfg)
}
