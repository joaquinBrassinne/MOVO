package logger_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/platform/logger"
)

// TestNewWritesJSONLines covers S6: every record is a JSON object carrying the
// time, level and msg keys.
func TestNewWritesJSONLines(t *testing.T) {
	var buf bytes.Buffer

	log := logger.New("info", &buf)
	log.Info("first", "attempt", 1)
	log.Warn("second")

	lines := strings.Split(strings.TrimSuffix(buf.String(), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("got %d log lines, want 2; output = %q", len(lines), buf.String())
	}

	wantMsgs := []string{"first", "second"}
	wantLevels := []string{"INFO", "WARN"}

	for i, line := range lines {
		var record map[string]any
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			t.Fatalf("line %d is not a JSON object: %v; line = %q", i, err, line)
		}
		for _, key := range []string{"time", "level", "msg"} {
			if _, ok := record[key]; !ok {
				t.Errorf("line %d is missing key %q; line = %q", i, key, line)
			}
		}
		if record["msg"] != wantMsgs[i] {
			t.Errorf("line %d msg = %v, want %q", i, record["msg"], wantMsgs[i])
		}
		if record["level"] != wantLevels[i] {
			t.Errorf("line %d level = %v, want %q", i, record["level"], wantLevels[i])
		}
		if _, err := time.Parse(time.RFC3339Nano, record["time"].(string)); err != nil {
			t.Errorf("line %d time = %v, want an RFC3339 timestamp: %v", i, record["time"], err)
		}
	}
}

// TestNewFiltersBelowLevel covers S6: a Debug record is suppressed at info and
// emitted at debug.
func TestNewFiltersBelowLevel(t *testing.T) {
	t.Run("info suppresses debug", func(t *testing.T) {
		var buf bytes.Buffer

		log := logger.New("info", &buf)
		log.Debug("suppressed")
		if buf.Len() != 0 {
			t.Fatalf("Debug record at info level produced output %q, want none", buf.String())
		}

		log.Error("kept")
		if !strings.Contains(buf.String(), `"msg":"kept"`) {
			t.Errorf("Error record at info level was not emitted; output = %q", buf.String())
		}
	})

	t.Run("debug emits debug", func(t *testing.T) {
		var buf bytes.Buffer

		log := logger.New("debug", &buf)
		log.Debug("kept")
		if !strings.Contains(buf.String(), `"msg":"kept"`) {
			t.Errorf("Debug record at debug level was not emitted; output = %q", buf.String())
		}
	})

	t.Run("error suppresses warn", func(t *testing.T) {
		var buf bytes.Buffer

		log := logger.New("error", &buf)
		log.Warn("suppressed")
		if buf.Len() != 0 {
			t.Fatalf("Warn record at error level produced output %q, want none", buf.String())
		}
	})
}

// TestNewAcceptsConfiguredLevelCasing covers S6 for the values produced by
// config.Load.
func TestNewAcceptsConfiguredLevelCasing(t *testing.T) {
	for _, level := range []string{"debug", "info", "warn", "error", "WARN"} {
		var buf bytes.Buffer

		log := logger.New(level, &buf)
		log.Error("kept")
		if !strings.Contains(buf.String(), `"msg":"kept"`) {
			t.Errorf("level %q suppressed an Error record; output = %q", level, buf.String())
		}
	}
}

// TestNewToleratesNilWriter covers S6 robustness: a nil writer never panics.
func TestNewToleratesNilWriter(t *testing.T) {
	log := logger.New("info", nil)
	log.Info("no panic")
}
