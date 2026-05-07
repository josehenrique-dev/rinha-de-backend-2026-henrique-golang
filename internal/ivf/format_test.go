package ivf

import (
	"os"
	"testing"
)

func TestSaveLoad_Roundtrip(t *testing.T) {
	vectors := make([]float32, 16*Dim)
	labels := make([]uint8, 16)
	for i := range 16 {
		for d := range Dim {
			vectors[i*Dim+d] = float32(i) / 15.0
		}
		if i%3 == 0 {
			labels[i] = 1
		}
	}

	idx, err := Build(vectors, labels, 16)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	f, err := os.CreateTemp("", "ivf-test-*.bin")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	defer os.Remove(f.Name())

	if err := Save(idx, f.Name()); err != nil {
		t.Fatalf("Save: %v", err)
	}

	idx2, err := Load(f.Name())
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	defer idx2.Close()

	if idx2.ivf.clusters != idx.ivf.clusters {
		t.Errorf("clusters: got %d, want %d", idx2.ivf.clusters, idx.ivf.clusters)
	}
	if len(idx2.blocks) != len(idx.blocks) {
		t.Errorf("blocks len: got %d, want %d", len(idx2.blocks), len(idx.blocks))
	}
	if len(idx2.labels) != len(idx.labels) {
		t.Errorf("labels len: got %d, want %d", len(idx2.labels), len(idx.labels))
	}
}
