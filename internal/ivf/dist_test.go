package ivf

import "testing"

func TestDistInt16_Correctness(t *testing.T) {
	cases := []struct {
		q    [Dim]int16
		v    [Dim]int16
		want int32
	}{
		{
			[Dim]int16{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			[Dim]int16{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			0,
		},
		{
			[Dim]int16{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			[Dim]int16{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			1,
		},
		{
			[Dim]int16{3, 4, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			[Dim]int16{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			25,
		},
		{
			[Dim]int16{100, 200, 300, 400, 500, 600, 700, 800, 900, 1000, 1100, 1200, 1300, 1400},
			[Dim]int16{50, 100, 150, 200, 250, 300, 350, 400, 450, 500, 550, 600, 650, 700},
			2537500,
		},
	}
	for _, tc := range cases {
		got := distInt16(tc.q, tc.v[:])
		if got != tc.want {
			t.Errorf("distInt16(%v, %v) = %d, want %d", tc.q, tc.v, got, tc.want)
		}
	}
}

func BenchmarkDistInt16(b *testing.B) {
	q := [Dim]int16{100, 200, 300, 400, 500, 600, 700, 800, 900, 1000, 1100, 1200, 1300, 1400}
	v := []int16{50, 100, 150, 200, 250, 300, 350, 400, 450, 500, 550, 600, 650, 700}
	b.ResetTimer()
	for range b.N {
		distInt16(q, v)
	}
}
