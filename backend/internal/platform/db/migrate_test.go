package db_test

import (
	"context"
	"io/fs"
	"strings"
	"testing"

	"github.com/pressly/goose/v3"

	"github.com/joaquinBrassinne/MOVO/internal/platform/db"
)

const firstMigration = "00001_extensions.sql"

// TestMigrationsEmbedThePostGISExtension pins the rule that the first migration
// is compiled into the binary and enables the PostGIS extension on the way up.
func TestMigrationsEmbedThePostGISExtension(t *testing.T) {
	migrations := db.Migrations()
	if migrations == nil {
		t.Fatal("Migrations() = nil, want the embedded migration tree")
	}

	names, err := fs.Glob(migrations, "*.sql")
	if err != nil {
		t.Fatalf("Glob over the embedded migrations: %v", err)
	}
	if len(names) == 0 {
		t.Fatal("no embedded .sql migrations, want at least one")
	}
	if names[0] != firstMigration {
		t.Fatalf("first embedded migration = %q, want %q", names[0], firstMigration)
	}

	body, err := fs.ReadFile(migrations, firstMigration)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", firstMigration, err)
	}

	wants := []string{
		"-- +goose Up",
		"-- +goose Down",
		"CREATE EXTENSION IF NOT EXISTS postgis",
	}
	for _, want := range wants {
		if !strings.Contains(string(body), want) {
			t.Errorf("%s does not contain %q", firstMigration, want)
		}
	}
}

// TestMigrationsParseInOrder pins that goose reads every embedded file as a SQL
// migration and that their versions are strictly ascending.
func TestMigrationsParseInOrder(t *testing.T) {
	migrations := db.Migrations()
	if migrations == nil {
		t.Fatal("Migrations() = nil, want the embedded migration tree")
	}

	goose.SetBaseFS(migrations)
	if err := goose.SetDialect("postgres"); err != nil {
		t.Fatalf("SetDialect(postgres): %v", err)
	}

	collected, err := goose.CollectMigrations(".", 0, goose.MaxVersion)
	if err != nil {
		t.Fatalf("CollectMigrations: %v", err)
	}
	if len(collected) == 0 {
		t.Fatal("CollectMigrations returned no migrations")
	}

	for i, migration := range collected {
		if migration.Source == "" {
			t.Errorf("migration %d has no source file", migration.Version)
		}
		if _, err := fs.Stat(migrations, migration.Source); err != nil {
			t.Errorf("migration %d source %q is not embedded: %v", migration.Version, migration.Source, err)
		}
		if i > 0 && collected[i-1].Version >= migration.Version {
			t.Errorf("migrations are not in ascending order: %d before %d",
				collected[i-1].Version, migration.Version)
		}
	}
}

// TestMigrateRejectsUnknownCommand pins that an unsupported command is reported
// before the database is touched.
func TestMigrateRejectsUnknownCommand(t *testing.T) {
	t.Parallel()

	err := db.Migrate(context.Background(), nil, "sideways")
	if err == nil {
		t.Fatal("Migrate(sideways) error = nil, want an unsupported-command error")
	}
	if !strings.Contains(err.Error(), "sideways") {
		t.Fatalf("error = %q, want it to name the rejected command", err)
	}
}
