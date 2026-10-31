// Command worker runs the scheduled tasks of MOVO. Every task takes a PostgreSQL
// advisory lock, so a second instance skips a task another one is running.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	benefitspostgres "github.com/joaquinBrassinne/MOVO/internal/benefits/adapters/postgres"
	"github.com/joaquinBrassinne/MOVO/internal/benefits/app/commands"
	"github.com/joaquinBrassinne/MOVO/internal/platform/advisory"
	"github.com/joaquinBrassinne/MOVO/internal/platform/config"
	businesspostgres "github.com/joaquinBrassinne/MOVO/internal/business/adapters/postgres"
	businesscommands "github.com/joaquinBrassinne/MOVO/internal/business/app/commands"
	"github.com/joaquinBrassinne/MOVO/internal/platform/db"
	"github.com/joaquinBrassinne/MOVO/internal/platform/logger"
	"github.com/joaquinBrassinne/MOVO/internal/platform/storage"
	reviewspostgres "github.com/joaquinBrassinne/MOVO/internal/reviews/adapters/postgres"
	reviewscommands "github.com/joaquinBrassinne/MOVO/internal/reviews/app/commands"
)

// dailyEvery is how often the daily tasks run. They also run once at start-up, which is
// harmless because every task is idempotent.
const dailyEvery = 24 * time.Hour

// Advisory lock keys, one per task. Never reuse or renumber one.
const (
	lockExpireCoupons      int64 = 4001
	lockPurgeHiddenReviews int64 = 4002
	lockPurgeObjects       int64 = 4003
)

// taskTimeout is the most a task may run before it is cut off, so a stuck call cannot
// hold its lock and starve the following rounds.
const taskTimeout = 10 * time.Minute

// purgeObjectsBatch is how many queued deletes one round tries; the rest wait for the next.
const purgeObjectsBatch = 200

// bounded returns run limited to d: its context ends at the deadline.
func bounded(d time.Duration, run func(context.Context) error) func(context.Context) error {
	return func(ctx context.Context) error {
		ctx, cancel := context.WithTimeout(ctx, d)
		defer cancel()
		return run(ctx)
	}
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "worker: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load(os.Getenv)
	if err != nil {
		return err
	}
	log := logger.New(cfg.LogLevel, os.Stdout)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	storageCfg, err := config.LoadStorage(os.Getenv)
	if err != nil {
		return err
	}

	expire := commands.NewExpireCoupons(benefitspostgres.NewCoupons(pool), nil)
	purgeObjects := businesscommands.NewPurgeObjects(storage.NewS3(storageCfg), businesspostgres.NewCleanup(pool), log)
	purge := reviewscommands.NewPurgeHiddenReviews(reviewspostgres.NewReviews(pool), nil)
	tasks := []task{
		{name: "expire-coupons", lock: lockExpireCoupons, run: func(ctx context.Context) error {
			n, err := expire.Handle(ctx)
			if err == nil {
				log.Info("expired overdue coupons", slog.Int64("count", n))
			}
			return err
		}},
		{name: "purge-hidden-reviews", lock: lockPurgeHiddenReviews, run: func(ctx context.Context) error {
			n, err := purge.Handle(ctx)
			if err == nil {
				log.Info("purged hidden reviews", slog.Int64("count", n))
			}
			return err
		}},
		{name: "purge-objects", lock: lockPurgeObjects, run: func(ctx context.Context) error {
			deleted, failed, err := purgeObjects.Handle(ctx, purgeObjectsBatch)
			if err == nil {
				log.Info("purged queued objects", slog.Int("deleted", deleted), slog.Int("failed", failed))
			}
			return err
		}},
	}
	for i := range tasks {
		tasks[i].run = bounded(taskTimeout, tasks[i].run)
	}

	log.Info("worker started", slog.Duration("every", dailyEvery))
	runDaily(ctx, log, pool, tasks)
	log.Info("worker stopped")
	return nil
}

type task struct {
	name string
	lock int64
	run  func(context.Context) error
}

// runDaily runs the tasks now and then every dailyEvery until ctx ends. A failing task
// is logged and tried again at the next round; it never stops the worker.
func runDaily(ctx context.Context, log *slog.Logger, pool *pgxpool.Pool, tasks []task) {
	ticker := time.NewTicker(dailyEvery)
	defer ticker.Stop()
	for {
		for _, t := range tasks {
			ran, err := advisory.RunExclusive(ctx, pool, t.lock, t.run)
			switch {
			case err != nil:
				log.Error("task failed", slog.String("task", t.name), slog.Any("error", err))
			case !ran:
				log.Info("task skipped: another instance holds it", slog.String("task", t.name))
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
