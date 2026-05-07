package ivf

import "testing"

func TestQuantizeVector_Values(t *testing.T) {
	cases := []struct {
		in  float32
		out int16
	}{
		{0.0, 0},
		{1.0, 10000},
		{-1.0, -10000},
		{-0.5, 0},
		{0.5, 5000},
		{1.5, 10000},
		{0.0001, 1},
	}
	for _, tc := range cases {
		var v [Dim]float32
		v[0] = tc.in
		q := quantizeVector(v)
		if q[0] != tc.out {
			t.Errorf("quantizeVector(%v)[0] = %d, want %d", tc.in, q[0], tc.out)
		}
	}
}

func TestQuantizedDistance_Self(t *testing.T) {
	v := [Dim]int16{100, 200, 300, 400, 500, 600, 700, 800, 900, 1000, 1100, 1200, 1300, 1400}
	ref := []int16{100, 200, 300, 400, 500, 600, 700, 800, 900, 1000, 1100, 1200, 1300, 1400}
	if d := quantizedDistance(v, ref, maxInt64); d != 0 {
		t.Errorf("distance to self = %d, want 0", d)
	}
}

func TestQuantizedDistance_Known(t *testing.T) {
	q := [Dim]int16{}
	ref := []int16{0, 0, 0, 0, 0, 0, 3, 4, 0, 0, 0, 0, 0, 0}
	if d := quantizedDistance(q, ref, maxInt64); d != 25 {
		t.Errorf("distance = %d, want 25", d)
	}
}

func TestQuantizedDistance_Cutoff(t *testing.T) {
	q := [Dim]int16{0, 0, 0, 0, 0, 0, 10000, 0, 0, 0, 0, 0, 0, 0}
	ref := make([]int16, Dim)
	full := quantizedDistance(q, ref, maxInt64)
	early := quantizedDistance(q, ref, 1)
	if full != 100000000 {
		t.Errorf("full = %d, want 100000000", full)
	}
	if early >= full {
		t.Errorf("cutoff should return early: %d >= %d", early, full)
	}
}
