package loader_test

import (
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/henriquefernandes/rinha-2026/internal/loader"
)

func TestLoadAndClose(t *testing.T) {
	dir := t.TempDir()
	vectorsPath := filepath.Join(dir, "vectors.bin")
	labelsPath := filepath.Join(dir, "labels.bin")

	const count = 10
	const dim = 14
	vectors := make([]float32, count*dim)
	for i := range vectors {
		vectors[i] = float32(i) * 0.01
	}
	labels := make([]uint8, count)
	labels[0] = 1

	if err := writeTestBinary(vectorsPath, vectors); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(labelsPath, labels, 0644); err != nil {
		t.Fatal(err)
	}

	ds, err := loader.Load(vectorsPath, labelsPath, dim)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	defer ds.Close()

	if ds.Count != count {
		t.Errorf("expected count %d, got %d", count, ds.Count)
	}
	if ds.Dim != dim {
		t.Errorf("expected dim %d, got %d", dim, ds.Dim)
	}
	if ds.Labels[0] != 1 {
		t.Errorf("expected label 1, got %d", ds.Labels[0])
	}
	if ds.Vectors[0] != vectors[0] {
		t.Errorf("expected vector[0] %f, got %f", vectors[0], ds.Vectors[0])
	}
}

func writeTestBinary(path string, data []float32) error {
	b := make([]byte, len(data)*4)
	for i, v := range data {
		bits := math.Float32bits(v)
		b[i*4] = byte(bits)
		b[i*4+1] = byte(bits >> 8)
		b[i*4+2] = byte(bits >> 16)
		b[i*4+3] = byte(bits >> 24)
	}
	return os.WriteFile(path, b, 0644)
}
