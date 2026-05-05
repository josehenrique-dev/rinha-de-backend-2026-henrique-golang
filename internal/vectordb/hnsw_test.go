package vectordb_test

import (
	"sync"
	"testing"

	"github.com/josehenrique-dev/rinha-2026/internal/loader"
	"github.com/josehenrique-dev/rinha-2026/internal/vectordb"
)

func TestBuildAndSearch(t *testing.T) {
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

	ds := &loader.Dataset{
		Vectors: vectors,
		Labels:  labels,
		Dim:     dim,
		Count:   n,
	}

	idx, err := vectordb.Build(ds)
	if err != nil {
		t.Fatalf("Build error: %v", err)
	}

	query := [14]float32{}
	score := idx.Search(query, 5)

	if score < 0 || score > 1 {
		t.Errorf("fraud_score out of range [0,1]: %f", score)
	}
}

func TestSearchAllFraud(t *testing.T) {
	const dim = 14
	const n = 20

	vectors := make([]float32, n*dim)
	labels := make([]uint8, n)
	for i := range labels {
		labels[i] = 1
	}

	ds := &loader.Dataset{
		Vectors: vectors,
		Labels:  labels,
		Dim:     dim,
		Count:   n,
	}

	idx, err := vectordb.Build(ds)
	if err != nil {
		t.Fatalf("Build error: %v", err)
	}

	score := idx.Search([14]float32{}, 5)
	if score != 1.0 {
		t.Errorf("expected score 1.0 (all fraud), got %f", score)
	}
}

func TestSearchAllLegit(t *testing.T) {
	const dim = 14
	const n = 20

	vectors := make([]float32, n*dim)
	labels := make([]uint8, n)

	ds := &loader.Dataset{
		Vectors: vectors,
		Labels:  labels,
		Dim:     dim,
		Count:   n,
	}

	idx, err := vectordb.Build(ds)
	if err != nil {
		t.Fatalf("Build error: %v", err)
	}

	score := idx.Search([14]float32{}, 5)
	if score != 0.0 {
		t.Errorf("expected score 0.0 (all legit), got %f", score)
	}
}

func TestSearch_Concurrent(t *testing.T) {
	const dim = 14
	const n = 100
	vectors := make([]float32, n*dim)
	labels := make([]uint8, n)
	for i := 0; i < n; i++ {
		for d := 0; d < dim; d++ {
			vectors[i*dim+d] = float32(i) / float32(n)
		}
	}
	ds := &loader.Dataset{Vectors: vectors, Labels: labels, Dim: dim, Count: n}
	idx, err := vectordb.Build(ds)
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			score := idx.Search([14]float32{}, 5)
			if score < 0 || score > 1 {
				t.Errorf("concurrent score out of range: %f", score)
			}
		}()
	}
	wg.Wait()
}
