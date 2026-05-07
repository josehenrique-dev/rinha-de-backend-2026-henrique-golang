package ivf

import "testing"

func TestBlocksForRows(t *testing.T) {
	cases := [][2]int{{0, 0}, {1, 1}, {8, 1}, {9, 2}, {16, 2}, {17, 3}}
	for _, tc := range cases {
		got := blocksForRows(tc[0])
		if got != tc[1] {
			t.Errorf("blocksForRows(%d) = %d, want %d", tc[0], got, tc[1])
		}
	}
}

func TestBuildIVFBlocks_Layout(t *testing.T) {
	vectors := make([]int16, 8*Dim)
	for i := range 8 {
		for d := range Dim {
			vectors[i*Dim+d] = int16(i*100 + d)
		}
	}
	listOffsets := []uint32{0, 8}
	blockOffsets := []uint32{0, 1}
	blocks := buildIVFBlocks(vectors, listOffsets, blockOffsets)
	if len(blocks) != blockStride {
		t.Fatalf("blocks len = %d, want %d", len(blocks), blockStride)
	}
	for d := range Dim {
		for lane := range 8 {
			got := blocks[d*blockSize+lane]
			want := int16(lane*100 + d)
			if got != want {
				t.Errorf("blocks[dim=%d, lane=%d] = %d, want %d", d, lane, got, want)
			}
		}
	}
}

func TestBuildCentroidBlocks_Layout(t *testing.T) {
	centroids := make([]int16, 8*Dim)
	for c := range 8 {
		for d := range Dim {
			centroids[c*Dim+d] = int16(c*100 + d)
		}
	}
	blocks := buildCentroidBlocks(centroids, 8)
	if len(blocks) != blockStride {
		t.Fatalf("centroid blocks len = %d, want %d", len(blocks), blockStride)
	}
	for d := range Dim {
		for lane := range 8 {
			got := blocks[d*blockSize+lane]
			want := int16(lane*100 + d)
			if got != want {
				t.Errorf("centroid block[dim=%d, lane=%d] = %d, want %d", d, lane, got, want)
			}
		}
	}
}
