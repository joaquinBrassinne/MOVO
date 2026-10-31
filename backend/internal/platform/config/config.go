// Package config loads the process configuration from the environment.
//
// Loading goes through an injectable source so tests never mutate the process
// environment.
package config

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Environment variables understood by Load.
const (
	envHTTPAddr        = "HTTP_ADDR"
	envDatabaseURL     = "DATABASE_URL"
	envLogLevel        = "LOG_LEVEL"
	envShutdownTimeout = "SHUTDOWN_TIMEOUT"
	envAPIDocs         = "API_DOCS"
)

// Defaults applied when the corresponding variable is unset or empty.
const (
	defaultHTTPAddr        = ":8080"
	defaultLogLevel        = "info"
	defaultShutdownTimeout = 10 * time.Second
)

// logLevels holds the accepted LOG_LEVEL values, in documentation order.
var logLevels = []string{"debug", "info", "warn", "error"}

// Config holds the runtime configuration of a MOVO binary.
type Config struct {
	// HTTPAddr is the address the HTTP server listens on, used verbatim.
	HTTPAddr string
	// DatabaseURL is the required PostgreSQL connection string (with PostGIS).
	DatabaseURL string
	// LogLevel is the minimum level emitted by the logger, always lowercase.
	LogLevel string
	// ShutdownTimeout bounds the graceful shutdown budget.
	ShutdownTimeout time.Duration
	// APIDocs serves the interactive API documentation at /docs. Off by default: it is a
	// development aid, not part of the public surface.
	APIDocs bool
}

// Load builds a Config from the values returned by get, which is normally
// os.Getenv. Every variable is optional except DATABASE_URL; missing or empty
// values fall back to the documented defaults.
func Load(get func(string) string) (Config, error) {
	if get == nil {
		return Config{}, errors.New("config: environment lookup must not be nil")
	}

	databaseURL := strings.TrimSpace(get(envDatabaseURL))
	if databaseURL == "" {
		return Config{}, fmt.Errorf("config: %s is required and must not be empty", envDatabaseURL)
	}

	logLevel, err := parseLogLevel(get(envLogLevel))
	if err != nil {
		return Config{}, err
	}

	shutdownTimeout, err := parseShutdownTimeout(get(envShutdownTimeout))
	if err != nil {
		return Config{}, err
	}

	apiDocs, err := parseAPIDocs(get(envAPIDocs))
	if err != nil {
		return Config{}, err
	}

	return Config{
		HTTPAddr:        valueOrDefault(get(envHTTPAddr), defaultHTTPAddr),
		DatabaseURL:     databaseURL,
		LogLevel:        logLevel,
		ShutdownTimeout: shutdownTimeout,
		APIDocs:         apiDocs,
	}, nil
}

// parseAPIDocs reads API_DOCS as a boolean; unset or empty means off.
func parseAPIDocs(raw string) (bool, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return false, nil
	}

	enabled, err := strconv.ParseBool(trimmed)
	if err != nil {
		return false, fmt.Errorf("config: %s %q must be true or false: %w", envAPIDocs, raw, err)
	}
	return enabled, nil
}

// parseLogLevel normalizes LOG_LEVEL to a lowercase member of logLevels.
func parseLogLevel(raw string) (string, error) {
	candidate := strings.ToLower(strings.TrimSpace(raw))
	if candidate == "" {
		return defaultLogLevel, nil
	}
	for _, level := range logLevels {
		if candidate == level {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("config: %s %q is not supported, want one of %s",
		envLogLevel, raw, strings.Join(logLevels, ", "))
}

// parseShutdownTimeout parses SHUTDOWN_TIMEOUT as a positive duration.
func parseShutdownTimeout(raw string) (time.Duration, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return defaultShutdownTimeout, nil
	}

	timeout, err := time.ParseDuration(trimmed)
	if err != nil {
		return 0, fmt.Errorf("config: %s %q is not a valid duration such as 10s: %w",
			envShutdownTimeout, raw, err)
	}
	if timeout <= 0 {
		return 0, fmt.Errorf("config: %s %q must be positive", envShutdownTimeout, raw)
	}
	return timeout, nil
}

// valueOrDefault returns value unless it is empty.
func valueOrDefault(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
