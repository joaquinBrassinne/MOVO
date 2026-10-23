// Command migrate applies the SQL migrations embedded in the binary.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/joaquinBrassinne/MOVO/internal/platform/config"
	"github.com/joaquinBrassinne/MOVO/internal/platform/db"
	"github.com/joaquinBrassinne/MOVO/internal/platform/logger"
)

const usage = "usage: migrate <up|down|status|version>"

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "migrate: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) != 1 {
		return errors.New(usage)
	}
	command := args[0]

	cfg, err := config.Load(os.Getenv)
	if err != nil {
		return err
	}
	log := logger.New(cfg.LogLevel, os.Stdout)

	ctx := context.Background()

	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	if err := db.Migrate(ctx, pool, command); err != nil {
		log.Error("migration failed", "command", command, "error", err)
		return err
	}

	log.Info("migration finished", "command", command)

	return nil
}
