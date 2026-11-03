package config

import "testing"

func TestLoadSearchRate(t *testing.T) {
	for _, tc := range []struct {
		name, value string
		want        int
		wantErr     bool
	}{
		{"unset keeps the default", "", 300, false},
		{"spaces are trimmed", " 60 ", 60, false},
		{"zero turns it off", "0", 0, false},
		{"the upper bound", "100000", 100000, false},
		{"not a number", "fast", 0, true},
		{"negative", "-1", 0, true},
		{"too large", "100001", 0, true},
		{"fraction", "1.5", 0, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := LoadSearchRate(func(string) string { return tc.value })
			if (err != nil) != tc.wantErr || (err == nil && got.PerMinute != tc.want) {
				t.Errorf("LoadSearchRate(%q) = %+v, %v; want %d, error %v", tc.value, got, err, tc.want, tc.wantErr)
			}
		})
	}
	if _, err := LoadSearchRate(nil); err == nil {
		t.Error("nil lookup accepted")
	}
}
