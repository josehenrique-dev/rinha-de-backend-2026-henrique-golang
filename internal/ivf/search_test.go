package ivf

import "testing"

func TestSearch_EndToEnd(t *testing.T) {
	n := 1024
	vectors := make([]float32, n*Dim)
	labels := make([]uint8, n)

	for i := range n {
		for d := range Dim {
			vectors[i*Dim+d] = float32(i) / float32(n)
		}
		if i < n/2 {
			labels[i] = 1
		}
	}

	idx, err := Build(vectors, labels, n)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	query := [Dim]float32{}
	fc := idx.SearchCount(query, 5)
	if fc < 0 || fc > 5 {
		t.Errorf("fraudCount out of range: %d", fc)
	}

	query2 := [Dim]float32{}
	for d := range Dim {
		query2[d] = 1.0
	}
	fc2 := idx.SearchCount(query2, 5)
	if fc2 < 0 || fc2 > 5 {
		t.Errorf("fraudCount2 out of range: %d", fc2)
	}
}
