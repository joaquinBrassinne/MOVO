package config_test

import (
	"strings"
	"testing"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/platform/config"
)

const testDatabaseURL = "postgres://movo:movo@localhost:5432/movo?sslmode=disable"

// env returns a LookupEnv-like source backed by values.
func env(values map[string]string) func(string) string {
	return func(key string) string {
		return values[key]
	}
}

// TestLoadDefaults covers S3, S4 and S5: every optional variable falls back to
// its documented default when it is unset.
func TestLoadDefaults(t *testing.T) {
	cfg, err := config.Load(env(map[string]string{"DATABASE_URL": testDatabaseURL}))
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}

	if cfg.DatabaseURL != testDatabaseURL {
		t.Errorf("DatabaseURL = %q, want %q", cfg.DatabaseURL, testDatabaseURL)
	}
	if cfg.HTTPAddr != ":8080" {
		t.Errorf("HTTPAddr = %q, want %q", cfg.HTTPAddr, ":8080")
	}
	if cfg.LogLevel != "info" {
		t.Errorf("LogLevel = %q, want %q", cfg.LogLevel, "info")
	}
	if cfg.ShutdownTimeout != 10*time.Second {
		t.Errorf("ShutdownTimeout = %v, want %v", cfg.ShutdownTimeout, 10*time.Second)
	}
}

// TestLoadAPIDocs: the interactive docs are off unless API_DOCS says otherwise, and a
// value that is not a boolean stops the start-up instead of silently leaving them off.
func TestLoadAPIDocs(t *testing.T) {
	for _, tt := range []struct {
		input   string
		want    bool
		wantErr bool
	}{
		{input: "", want: false},
		{input: "false", want: false},
		{input: "true", want: true},
		{input: " 1 ", want: true},
		{input: "yes please", wantErr: true},
	} {
		cfg, err := config.Load(env(map[string]string{"DATABASE_URL": testDatabaseURL, "API_DOCS": tt.input}))
		if tt.wantErr {
			if err == nil || !strings.Contains(err.Error(), "API_DOCS") {
				t.Errorf("API_DOCS=%q: error = %v, want one naming API_DOCS", tt.input, err)
			}
			continue
		}
		if err != nil || cfg.APIDocs != tt.want {
			t.Errorf("API_DOCS=%q: APIDocs = %v, err = %v, want %v", tt.input, cfg.APIDocs, err, tt.want)
		}
	}
}

// TestLoadEmptyOptionalVariablesUseDefaults covers S3, S4 and S5: an empty
// value is treated as unset.
func TestLoadEmptyOptionalVariablesUseDefaults(t *testing.T) {
	cfg, err := config.Load(env(map[string]string{
		"DATABASE_URL":     testDatabaseURL,
		"HTTP_ADDR":        "",
		"LOG_LEVEL":        "",
		"SHUTDOWN_TIMEOUT": "",
	}))
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}

	if cfg.HTTPAddr != ":8080" {
		t.Errorf("HTTPAddr = %q, want %q", cfg.HTTPAddr, ":8080")
	}
	if cfg.LogLevel != "info" {
		t.Errorf("LogLevel = %q, want %q", cfg.LogLevel, "info")
	}
	if cfg.ShutdownTimeout != 10*time.Second {
		t.Errorf("ShutdownTimeout = %v, want %v", cfg.ShutdownTimeout, 10*time.Second)
	}
}

// TestLoadRequiresDatabaseURL covers S2.
func TestLoadRequiresDatabaseURL(t *testing.T) {
	tests := []struct {
		name   string
		values map[string]string
	}{
		{name: "missing", values: map[string]string{}},
		{name: "empty", values: map[string]string{"DATABASE_URL": ""}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := config.Load(env(tt.values))
			if err == nil {
				t.Fatalf("Load() error = nil, want an error naming DATABASE_URL; got config %+v", cfg)
			}
			if !strings.Contains(err.Error(), "DATABASE_URL") {
				t.Errorf("Load() error = %q, want it to contain %q", err, "DATABASE_URL")
			}
		})
	}
}

// TestLoadHTTPAddrVerbatim covers S3: a provided address is used unchanged.
func TestLoadHTTPAddrVerbatim(t *testing.T) {
	const addr = "127.0.0.1:9090"

	cfg, err := config.Load(env(map[string]string{
		"DATABASE_URL": testDatabaseURL,
		"HTTP_ADDR":    addr,
	}))
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}
	if cfg.HTTPAddr != addr {
		t.Errorf("HTTPAddr = %q, want %q", cfg.HTTPAddr, addr)
	}
}

// TestLoadLogLevelNormalizes covers S4: accepted values are matched
// case-insensitively and stored lowercase.
func TestLoadLogLevelNormalizes(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{input: "debug", want: "debug"},
		{input: "info", want: "info"},
		{input: "warn", want: "warn"},
		{input: "error", want: "error"},
		{input: "DEBUG", want: "debug"},
		{input: "Info", want: "info"},
		{input: "WaRn", want: "warn"},
		{input: "ERROR", want: "error"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			cfg, err := config.Load(env(map[string]string{
				"DATABASE_URL": testDatabaseURL,
				"LOG_LEVEL":    tt.input,
			}))
			if err != nil {
				t.Fatalf("Load() error = %v, want nil", err)
			}
			if cfg.LogLevel != tt.want {
				t.Errorf("LogLevel = %q, want %q", cfg.LogLevel, tt.want)
			}
		})
	}
}

// TestLoadRejectsUnknownLogLevel covers S4: the error names LOG_LEVEL and
// echoes the offending value.
func TestLoadRejectsUnknownLogLevel(t *testing.T) {
	const value = "verbose"

	cfg, err := config.Load(env(map[string]string{
		"DATABASE_URL": testDatabaseURL,
		"LOG_LEVEL":    value,
	}))
	if err == nil {
		t.Fatalf("Load() error = nil, want an error naming LOG_LEVEL; got config %+v", cfg)
	}
	if !strings.Contains(err.Error(), "LOG_LEVEL") {
		t.Errorf("Load() error = %q, want it to contain %q", err, "LOG_LEVEL")
	}
	if !strings.Contains(err.Error(), value) {
		t.Errorf("Load() error = %q, want it to echo %q", err, value)
	}
}

// TestLoadParsesShutdownTimeout covers S5: the duration uses Go/Docker style
// syntax.
func TestLoadParsesShutdownTimeout(t *testing.T) {
	tests := []struct {
		input string
		want  time.Duration
	}{
		{input: "10s", want: 10 * time.Second},
		{input: "500ms", want: 500 * time.Millisecond},
		{input: "2m30s", want: 150 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			cfg, err := config.Load(env(map[string]string{
				"DATABASE_URL":     testDatabaseURL,
				"SHUTDOWN_TIMEOUT": tt.input,
			}))
			if err != nil {
				t.Fatalf("Load() error = %v, want nil", err)
			}
			if cfg.ShutdownTimeout != tt.want {
				t.Errorf("ShutdownTimeout = %v, want %v", cfg.ShutdownTimeout, tt.want)
			}
		})
	}
}

// TestLoadRejectsInvalidShutdownTimeout covers S5: unparsable and non-positive
// values are rejected with an error naming SHUTDOWN_TIMEOUT.
func TestLoadRejectsInvalidShutdownTimeout(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{name: "unparsable", input: "abc"},
		{name: "zero", input: "0s"},
		{name: "negative", input: "-1s"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := config.Load(env(map[string]string{
				"DATABASE_URL":     testDatabaseURL,
				"SHUTDOWN_TIMEOUT": tt.input,
			}))
			if err == nil {
				t.Fatalf("Load() error = nil, want an error naming SHUTDOWN_TIMEOUT; got config %+v", cfg)
			}
			if !strings.Contains(err.Error(), "SHUTDOWN_TIMEOUT") {
				t.Errorf("Load() error = %q, want it to contain %q", err, "SHUTDOWN_TIMEOUT")
			}
		})
	}
}

// TestLoadReadsOnlyInjectedSource covers S1: the process environment is never
// consulted, so tests never have to mutate it.
func TestLoadReadsOnlyInjectedSource(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://process-env:secret@localhost:5432/process_env")
	t.Setenv("HTTP_ADDR", "127.0.0.1:1")
	t.Setenv("LOG_LEVEL", "error")

	const injectedURL = "postgres://injected:injected@localhost:5432/injected"

	cfg, err := config.Load(env(map[string]string{"DATABASE_URL": injectedURL}))
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}
	if cfg.DatabaseURL != injectedURL {
		t.Errorf("DatabaseURL = %q, want %q (injected source)", cfg.DatabaseURL, injectedURL)
	}
	if cfg.HTTPAddr != ":8080" {
		t.Errorf("HTTPAddr = %q, want %q (process env must be ignored)", cfg.HTTPAddr, ":8080")
	}
	if cfg.LogLevel != "info" {
		t.Errorf("LogLevel = %q, want %q (process env must be ignored)", cfg.LogLevel, "info")
	}
}
