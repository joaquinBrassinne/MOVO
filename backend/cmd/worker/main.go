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
	"github.com/joaquinBrassinne/MOVO/internal/platform/db"
	"github.com/joaquinBrassinne/MOVO/internal/platform/logger"
)

// dailyEvery is how often the daily tasks run. They also run once at start-up, which is
// harmless because every task is idempotent.
const dailyEvery = 24 * time.Hour

// Advisory lock keys, one per task. Never reuse or renumber one.
const lockExpireCoupons int64 = 4001

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

	expire := commands.NewExpireCoupons(benefitspostgres.NewCoupons(pool), nil)
	tasks := []task{{name: "expire-coupons", lock: lockExpireCoupons, run: func(ctx context.Context) error {
		n, err := expire.Handle(ctx)
		if err == nil {
			log.Info("expired overdue coupons", slog.Int64("count", n))
		}
		return err
	}}}

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
