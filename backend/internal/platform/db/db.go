// Package db opens the PostgreSQL connection pool and runs the migrations
// compiled into the binaries.
package db

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"maps"
	"slices"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/joaquinBrassinne/MOVO/migrations"
)

// connectTimeout bounds the initial handshake, so a wrong or unreachable
// DATABASE_URL fails the process instead of hanging it.
const connectTimeout = 5 * time.Second

// migrationCommands lists the goose commands accepted by Migrate.
var migrationCommands = map[string]func(context.Context, *sql.DB, string, ...goose.OptionsFunc) error{
	"up":      goose.UpContext,
	"down":    goose.DownContext,
	"status":  goose.StatusContext,
	"version": goose.VersionContext,
}

// Connect parses the connection string, opens a pgx pool and verifies it with a
// bounded ping. The caller owns the returned pool and must Close it during
// shutdown.
func Connect(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("db: parse DATABASE_URL: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("db: open pool: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, connectTimeout)
	defer cancel()

	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("db: ping: %w", err)
	}

	return pool, nil
}

// Migrations returns the migration tree embedded in the binary.
func Migrations() fs.FS {
	return migrations.FS
}

// gooseMu serializes Migrate: goose keeps its base filesystem and dialect in
// package globals, so concurrent calls (parallel integration tests, each with its
// own database) would race on them.
var gooseMu sync.Mutex

// Migrate runs one goose command (up, down, status or version) against pool
// using the embedded migrations. An unsupported command is reported before the
// database is touched.
func Migrate(ctx context.Context, pool *pgxpool.Pool, command string) error {
	run, ok := migrationCommands[command]
	if !ok {
		return fmt.Errorf("db: unknown migration command %q, want one of %v",
			command, slices.Sorted(maps.Keys(migrationCommands)))
	}

	gooseMu.Lock()
	defer gooseMu.Unlock()

	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("db: set migration dialect: %w", err)
	}

	// OpenDBFromPool shares the pool; closing this handle does not close it, so
	// a close error could not be acted on either: it is discarded deliberately.
	sqlDB := stdlib.OpenDBFromPool(pool)
	defer func() { _ = sqlDB.Close() }()

	if err := run(ctx, sqlDB, "."); err != nil {
		return fmt.Errorf("db: migrate %s: %w", command, err)
	}

	return nil
}
