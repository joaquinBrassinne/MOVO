package migrations_test

import (
	"strings"
	"testing"

	"github.com/joaquinBrassinne/MOVO/migrations"
)

// TestExtensionsDownDropsDependentsFirst pins the order that took a real
// failure to find: the postgis/postgis image ships postgis_topology and
// postgis_tiger_geocoder, so dropping postgis first fails with SQLSTATE 2BP01.
func TestExtensionsDownDropsDependentsFirst(t *testing.T) {
	t.Parallel()

	body, err := migrations.FS.ReadFile("00001_extensions.sql")
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}

	up, down, found := strings.Cut(string(body), "-- +goose Down")
	if !found {
		t.Fatal("migration has no -- +goose Down section")
	}

	// Comments explain the decision and may name CASCADE; only statements count.
	down = withoutComments(down)

	if !strings.Contains(up, "CREATE EXTENSION IF NOT EXISTS postgis;") {
		t.Error("up section does not create the postgis extension")
	}
	if strings.Contains(strings.ToUpper(down), "CASCADE") {
		t.Error("down uses CASCADE, which also drops objects this migration knows nothing about")
	}

	tiger := strings.Index(down, "DROP EXTENSION IF EXISTS postgis_tiger_geocoder;")
	topology := strings.Index(down, "DROP EXTENSION IF EXISTS postgis_topology;")
	postgis := strings.LastIndex(down, "DROP EXTENSION IF EXISTS postgis;")

	if tiger < 0 || topology < 0 || postgis < 0 {
		t.Fatalf("down does not drop all three extensions (tiger=%d topology=%d postgis=%d)",
			tiger, topology, postgis)
	}
	if tiger > postgis || topology > postgis {
		t.Error("down drops postgis before its dependents, which fails with SQLSTATE 2BP01")
	}
}

// withoutComments drops whole-line SQL comments so prose cannot satisfy or
// trip a check meant for statements.
func withoutComments(sql string) string {
	var statements strings.Builder

	for line := range strings.SplitSeq(sql, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "--") {
			continue
		}

		statements.WriteString(line)
		statements.WriteString("\n")
	}

	return statements.String()
}
