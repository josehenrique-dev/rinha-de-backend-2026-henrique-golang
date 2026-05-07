package ivf

import "testing"

func TestMaxVarianceDimension(t *testing.T) {
	vectors := []int16{
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		100, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		200, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	}
	ids := []uint32{0, 1, 2}
	dim := maxVarianceDimension(vectors, ids, 0, 3)
	if dim != 0 {
		t.Errorf("max variance dim = %d, want 0", dim)
	}
}

func TestBalancedSplit_TwoClusters(t *testing.T) {
	n := 8
	vectors := make([]int16, n*Dim)
	ids := make([]uint32, n)
	for i := range n {
		ids[i] = uint32(i)
		vectors[i*Dim] = int16(i * 1000)
	}
	ranges := make([]ivfBuildRange, 2)
	balancedSplit(vectors, ids, ranges, 0, n, 0, 2)
	size0 := ranges[0].end - ranges[0].start
	size1 := ranges[1].end - ranges[1].start
	if size0 != 4 || size1 != 4 {
		t.Errorf("cluster sizes = %d, %d; want 4, 4", size0, size1)
	}
}

func TestComputeClusterStats_Centroid(t *testing.T) {
	vectors := []int16{
		1000, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		3000, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	}
	ids := []uint32{0, 1}
	centroid := make([]int16, Dim)
	bboxMin := make([]int16, Dim)
	bboxMax := make([]int16, Dim)
	computeClusterStats(vectors, ids, ivfBuildRange{0, 2}, centroid, bboxMin, bboxMax)
	if centroid[0] != 2000 {
		t.Errorf("centroid[0] = %d, want 2000", centroid[0])
	}
	if bboxMin[0] != 1000 || bboxMax[0] != 3000 {
		t.Errorf("bbox = [%d, %d], want [1000, 3000]", bboxMin[0], bboxMax[0])
	}
}
