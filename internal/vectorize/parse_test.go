package vectorize

import "testing"

func TestParseHourWeekday(t *testing.T) {
	cases := []struct {
		s       string
		hour    int
		weekday int
	}{
		{"2026-03-11T20:23:35Z", 20, 2},
		{"2026-03-09T00:00:00Z", 0, 0},
		{"2026-03-15T23:59:59Z", 23, 6},
		{"2026-03-10T12:00:00Z", 12, 1},
	}
	for _, tc := range cases {
		h, wd, err := parseHourWeekday(tc.s)
		if err != nil {
			t.Errorf("parseHourWeekday(%q) unexpected error: %v", tc.s, err)
			continue
		}
		if h != tc.hour {
			t.Errorf("parseHourWeekday(%q) hour: got %d want %d", tc.s, h, tc.hour)
		}
		if wd != tc.weekday {
			t.Errorf("parseHourWeekday(%q) weekday: got %d want %d", tc.s, wd, tc.weekday)
		}
	}
}
