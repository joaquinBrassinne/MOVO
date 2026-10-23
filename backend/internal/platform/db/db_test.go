package db_test

import (
	"context"
	"testing"

	"github.com/joaquinBrassinne/MOVO/internal/platform/db"
)

// TestConnectRejectsUnusableDatabaseURL pins the rule that Connect fails fast on
// a connection string it cannot parse, without ever reaching the network.
func TestConnectRejectsUnusableDatabaseURL(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"empty":    "",
		"garbage":  "esto no es una cadena de conexión",
		"bad port": "postgres://movo:movo@localhost:no-es-un-puerto/movo",
	}

	for name, databaseURL := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			pool, err := db.Connect(context.Background(), databaseURL)
			if pool != nil {
				pool.Close()
				t.Fatalf("Connect(%q) returned a pool alongside err = %v, want nil", databaseURL, err)
			}
			if err == nil {
				t.Fatalf("Connect(%q) error = nil, want a parse error", databaseURL)
			}
		})
	}
}
