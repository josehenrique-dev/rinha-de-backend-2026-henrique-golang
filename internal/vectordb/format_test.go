package vectordb

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveLoad_RoundTrip(t *testing.T) {
	const dim = 14
	const n = 100
	vectors := make([]float32, n*dim)
	labels := make([]uint8, n)
	for i := 0; i < n; i++ {
		for d := 0; d < dim; d++ {
			vectors[i*dim+d] = float32(i) / float32(n)
		}
		if i%5 == 0 {
			labels[i] = 1
		}
	}

	g := buildGraph(vectors, labels, n, dim, 4, 50)
	query := make([]float32, dim)
	originalResults := g.search(query, 5, 20)

	dir := t.TempDir()
	path := filepath.Join(dir, "index.bin")

	if err := saveGraph(g, path); err != nil {
		t.Fatalf("saveGraph: %v", err)
	}

	g2, mem, err := loadGraph(path, vectors, labels, dim)
	if err != nil {
		t.Fatalf("loadGraph: %v", err)
	}
	defer freeMmap(mem)

	loadedResults := g2.search(query, 5, 20)

	if len(loadedResults) != len(originalResults) {
		t.Fatalf("result length mismatch: %d vs %d", len(loadedResults), len(originalResults))
	}
	origSet := make(map[uint32]bool)
	for _, id := range originalResults {
		origSet[id] = true
	}
	for _, id := range loadedResults {
		if !origSet[id] {
			t.Errorf("loaded result %d not in original results", id)
		}
	}
}

func TestLoad_WrongMagic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.bin")
	os.WriteFile(path, []byte("BADMAGIC"), 0644)
	_, _, err := loadGraph(path, nil, nil, 14)
	if err == nil {
		t.Fatal("expected error for wrong magic")
	}
}
