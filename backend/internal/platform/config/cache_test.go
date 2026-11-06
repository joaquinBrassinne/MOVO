package config

import (
	"testing"
	"time"
)

func TestLoadCacheDefaults(t *testing.T) {
	got, err := LoadCache(func(string) string { return "" })
	if err != nil {
		t.Fatal(err)
	}
	if got.RedisURL != "" || got.SearchTTL != time.Minute || got.CellDegrees != 0.002 {
		t.Errorf("defaults = %+v", got)
	}
}

func TestLoadCacheReadsValues(t *testing.T) {
	env := map[string]string{"REDIS_URL": " redis://r:6379 ", "SEARCH_CACHE_TTL": "30s", "SEARCH_CACHE_CELL_DEG": "0.01"}
	got, err := LoadCache(func(k string) string { return env[k] })
	if err != nil {
		t.Fatal(err)
	}
	if got.RedisURL != "redis://r:6379" || got.SearchTTL != 30*time.Second || got.CellDegrees != 0.01 {
		t.Errorf("got %+v", got)
	}
}

func TestLoadCacheRefusesBadValues(t *testing.T) {
	for _, env := range []map[string]string{
		{"SEARCH_CACHE_TTL": "soon"}, {"SEARCH_CACHE_TTL": "0s"}, {"SEARCH_CACHE_TTL": "-1m"}, {"SEARCH_CACHE_TTL": "2h"},
		{"SEARCH_CACHE_CELL_DEG": "x"}, {"SEARCH_CACHE_CELL_DEG": "0"}, {"SEARCH_CACHE_CELL_DEG": "NaN"}, {"SEARCH_CACHE_CELL_DEG": "1"},
	} {
		if _, err := LoadCache(func(k string) string { return env[k] }); err == nil {
			t.Errorf("LoadCache(%v) accepted a bad value", env)
		}
	}
	if _, err := LoadCache(nil); err == nil {
		t.Error("nil lookup accepted")
	}
}
