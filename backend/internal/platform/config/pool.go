package config

import (
	"errors"
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// Environment variables understood by LoadPool (SDD §7.5).
const (
	envDBMaxConns        = "DB_MAX_CONNS"
	envDBMinConns        = "DB_MIN_CONNS"
	envDBMaxConnLifetime = "DB_MAX_CONN_LIFETIME"
	envDBMaxConnIdleTime = "DB_MAX_CONN_IDLE_TIME"
)

// Defaults of the connection pool. They are the defaults pgxpool applies when none is set, so
// an unset environment keeps the behavior the service had before these variables existed.
const (
	defaultDBMinConns        = 0
	defaultDBMaxConnLifetime = time.Hour
	defaultDBMaxConnIdleTime = 30 * time.Minute
	defaultDBMinMaxConns     = 4
)

// Pool holds the sizing of the PostgreSQL connection pool (SDD 7.5).
type Pool struct {
	// MaxConns is the maximum number of connections the pool opens.
	MaxConns int32
	// MinConns is the number of connections the pool keeps open while idle; MinConns <= MaxConns.
	MinConns int32
	// MaxConnLifetime is the age after which a connection is closed and replaced.
	MaxConnLifetime time.Duration
	// MaxConnIdleTime is how long an idle connection may stay open before it is closed.
	MaxConnIdleTime time.Duration
}

// DefaultMaxConns is the pool size pgxpool uses when DB_MAX_CONNS is unset: the greater of 4
// and the number of CPUs.
func DefaultMaxConns() int32 {
	return max(int32(defaultDBMinMaxConns), int32(runtime.NumCPU()))
}

// LoadPool builds Pool from get, normally os.Getenv. An unset or empty variable keeps the
// default; anything that is not a valid value, or a MinConns above MaxConns, is refused at
// start-up.
func LoadPool(get func(string) string) (Pool, error) {
	if get == nil {
		return Pool{}, errors.New("config: environment lookup must not be nil")
	}

	out := Pool{
		MaxConns:        DefaultMaxConns(),
		MinConns:        defaultDBMinConns,
		MaxConnLifetime: defaultDBMaxConnLifetime,
		MaxConnIdleTime: defaultDBMaxConnIdleTime,
	}
	if raw := strings.TrimSpace(get(envDBMaxConns)); raw != "" {
		n, err := strconv.ParseInt(raw, 10, 32)
		if err != nil || n <= 0 {
			return Pool{}, fmt.Errorf("config: %s %q must be a whole number above zero", envDBMaxConns, raw)
		}
		out.MaxConns = int32(n)
	}
	if raw := strings.TrimSpace(get(envDBMinConns)); raw != "" {
		n, err := strconv.ParseInt(raw, 10, 32)
		if err != nil || n < 0 {
			return Pool{}, fmt.Errorf("config: %s %q must be a whole number of zero or more", envDBMinConns, raw)
		}
		out.MinConns = int32(n)
	}
	if out.MinConns > out.MaxConns {
		return Pool{}, fmt.Errorf("config: %s (%d) must not exceed %s (%d)", envDBMinConns, out.MinConns, envDBMaxConns, out.MaxConns)
	}
	if raw := strings.TrimSpace(get(envDBMaxConnLifetime)); raw != "" {
		d, err := time.ParseDuration(raw)
		if err != nil || d <= 0 {
			return Pool{}, fmt.Errorf("config: %s %q must be a duration above zero, like 1h", envDBMaxConnLifetime, raw)
		}
		out.MaxConnLifetime = d
	}
	if raw := strings.TrimSpace(get(envDBMaxConnIdleTime)); raw != "" {
		d, err := time.ParseDuration(raw)
		if err != nil || d <= 0 {
			return Pool{}, fmt.Errorf("config: %s %q must be a duration above zero, like 30m", envDBMaxConnIdleTime, raw)
		}
		out.MaxConnIdleTime = d
	}
	return out, nil
}
