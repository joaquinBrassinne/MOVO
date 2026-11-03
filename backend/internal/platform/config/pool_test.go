package config

import (
	"fmt"
	"runtime"
	"testing"
	"time"
)

func TestLoadPoolDefaultsMatchPgxDefaults(t *testing.T) {
	got, err := LoadPool(func(string) string { return "" })
	if err != nil {
		t.Fatal(err)
	}
	wantMax := int32(4)
	if n := int32(runtime.NumCPU()); n > wantMax {
		wantMax = n
	}
	if got.MaxConns != wantMax || got.MinConns != 0 || got.MaxConnLifetime != time.Hour || got.MaxConnIdleTime != 30*time.Minute {
		t.Errorf("defaults = %+v, want MaxConns %d, MinConns 0, lifetime 1h, idle 30m", got, wantMax)
	}
}

func TestLoadPoolReadsValues(t *testing.T) {
	env := map[string]string{
		"DB_MAX_CONNS":          " 25 ",
		"DB_MIN_CONNS":          "5",
		"DB_MAX_CONN_LIFETIME":  "2h",
		"DB_MAX_CONN_IDLE_TIME": "10m",
	}
	got, err := LoadPool(func(k string) string { return env[k] })
	if err != nil {
		t.Fatal(err)
	}
	if got.MaxConns != 25 || got.MinConns != 5 || got.MaxConnLifetime != 2*time.Hour || got.MaxConnIdleTime != 10*time.Minute {
		t.Errorf("got %+v", got)
	}
}

func TestLoadPoolAllowsMinConnsEqualToMaxConns(t *testing.T) {
	env := map[string]string{"DB_MAX_CONNS": "3", "DB_MIN_CONNS": "3"}
	got, err := LoadPool(func(k string) string { return env[k] })
	if err != nil {
		t.Fatal(err)
	}
	if got.MinConns != 3 || got.MaxConns != 3 {
		t.Errorf("got %+v", got)
	}
}

func TestLoadPoolRefusesBadValues(t *testing.T) {
	for _, env := range []map[string]string{
		{"DB_MAX_CONNS": "many"},
		{"DB_MAX_CONNS": "0"},
		{"DB_MAX_CONNS": "-2"},
		{"DB_MAX_CONNS": "1.5"},
		{"DB_MIN_CONNS": "-1"},
		{"DB_MIN_CONNS": "x"},
		{"DB_MAX_CONNS": "2", "DB_MIN_CONNS": "3"},
		{"DB_MAX_CONN_LIFETIME": "soon"},
		{"DB_MAX_CONN_LIFETIME": "0s"},
		{"DB_MAX_CONN_LIFETIME": "-1h"},
		{"DB_MAX_CONN_IDLE_TIME": "forever"},
		{"DB_MAX_CONN_IDLE_TIME": "0"},
	} {
		if _, err := LoadPool(func(k string) string { return env[k] }); err == nil {
			t.Errorf("LoadPool(%v) accepted a bad value", env)
		}
	}
	if _, err := LoadPool(nil); err == nil {
		t.Error("nil lookup accepted")
	}
}

// TestLoadPoolRefusesMinConnsAboveTheDefaultMaxConns pins that MinConns is checked against the
// effective MaxConns, including the default one when DB_MAX_CONNS is unset.
func TestLoadPoolRefusesMinConnsAboveTheDefaultMaxConns(t *testing.T) {
	minConns := fmt.Sprint(DefaultMaxConns() + 1)
	env := map[string]string{"DB_MIN_CONNS": minConns}
	if _, err := LoadPool(func(k string) string { return env[k] }); err == nil {
		t.Errorf("DB_MIN_CONNS=%s with the default DB_MAX_CONNS accepted, want an error", minConns)
	}
}
