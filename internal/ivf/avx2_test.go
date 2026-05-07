//go:build amd64

package ivf

import (
	"testing"
	"unsafe"
)

func TestBlock8Distances_MatchScalar(t *testing.T) {
	query := [Dim]int16{100, 200, 300, 400, 500, 600, 700, 800, 900, 1000, 1100, 1200, 1300, 1400}

	vectors := make([]int16, 8*Dim)
	for lane := range 8 {
		for d := range Dim {
			vectors[lane*Dim+d] = int16(lane*50 + d*10)
		}
	}

	blocks := make([]int16, blockStride)
	for lane := range 8 {
		for d := range Dim {
			blocks[d*blockSize+lane] = vectors[lane*Dim+d]
		}
	}

	var out [8]int64
	if useIVFAVX2 {
		quantizedBlock8DistancesAVX2(&query[0], unsafe.Pointer(&blocks[0]), &out[0])
	} else {
		t.Skip("AVX2 not available")
	}

	for lane := range 8 {
		ref := make([]int16, Dim)
		for d := range Dim {
			ref[d] = vectors[lane*Dim+d]
		}
		want := quantizedDistance(query, ref, maxInt64)
		if out[lane] != want {
			t.Errorf("lane %d: got %d, want %d", lane, out[lane], want)
		}
	}
}
