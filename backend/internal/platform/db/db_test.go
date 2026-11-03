package db_test

import (
	"context"
	"testing"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/platform/config"
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

// TestConnectWithPoolRefusesInvalidSizing pins that an impossible pool sizing is
// refused before any connection is attempted.
func TestConnectWithPoolRefusesInvalidSizing(t *testing.T) {
	t.Parallel()

	cases := map[string]config.Pool{
		"zero max":      {MaxConns: 0, MaxConnLifetime: time.Hour, MaxConnIdleTime: time.Minute},
		"negative min":  {MaxConns: 4, MinConns: -1, MaxConnLifetime: time.Hour, MaxConnIdleTime: time.Minute},
		"min above max": {MaxConns: 2, MinConns: 3, MaxConnLifetime: time.Hour, MaxConnIdleTime: time.Minute},
		"zero lifetime": {MaxConns: 4, MaxConnLifetime: 0, MaxConnIdleTime: time.Minute},
		"zero idle":     {MaxConns: 4, MaxConnLifetime: time.Hour, MaxConnIdleTime: 0},
	}
	for name, sizing := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			pool, err := db.ConnectWithPool(context.Background(), "postgres://movo:movo@127.0.0.1:1/movo", sizing)
			if pool != nil {
				pool.Close()
				t.Fatalf("ConnectWithPool(%+v) returned a pool, want nil", sizing)
			}
			if err == nil {
				t.Fatalf("ConnectWithPool(%+v) error = nil, want a sizing error", sizing)
			}
		})
	}
}

// TestConnectWithPoolFailsAtPingWhenServerIsUnreachable pins that a valid sizing
// reaches the handshake and fails there, not on validation.
func TestConnectWithPoolFailsAtPingWhenServerIsUnreachable(t *testing.T) {
	t.Parallel()

	sizing := config.Pool{MaxConns: 4, MinConns: 0, MaxConnLifetime: time.Hour, MaxConnIdleTime: 30 * time.Minute}
	pool, err := db.ConnectWithPool(context.Background(), "postgres://movo:movo@127.0.0.1:1/movo?connect_timeout=1", sizing)
	if pool != nil {
		pool.Close()
		t.Fatal("ConnectWithPool against an unreachable server returned a pool, want nil")
	}
	if err == nil {
		t.Fatal("ConnectWithPool against an unreachable server error = nil, want a ping error")
	}
}
