package vectordb

import (
	"math"
	"sync"
	"testing"
)

func makeTestGraph(n int) (*graph, []float32) {
	const dim = 14
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
	return buildGraph(vectors, labels, n, dim, 8, 100), vectors
}

func TestBuildGraph_SearchInRange(t *testing.T) {
	g, _ := makeTestGraph(200)
	query := make([]float32, 14)
	results := g.search(query, 5, 20)
	if len(results) != 5 {
		t.Fatalf("expected 5 results, got %d", len(results))
	}
}

func TestBuildGraph_AllFraud(t *testing.T) {
	const dim = 14
	n := 50
	vectors := make([]float32, n*dim)
	labels := make([]uint8, n)
	for i := range labels {
		labels[i] = 1
	}
	g := buildGraph(vectors, labels, n, dim, 4, 50)
	results := g.search(make([]float32, dim), 5, 20)
	fraudCount := 0
	for _, id := range results {
		if g.labels[id] == 1 {
			fraudCount++
		}
	}
	if fraudCount != 5 {
		t.Fatalf("expected all fraud (5/5), got %d/5", fraudCount)
	}
}

func TestBuildGraph_Recall_SyntheticClusters(t *testing.T) {
	const dim = 14
	const n = 100
	vectors := make([]float32, n*dim)
	labels := make([]uint8, n)
	for i := 50; i < n; i++ {
		for d := 0; d < dim; d++ {
			vectors[i*dim+d] = 1.0
		}
	}
	g := buildGraph(vectors, labels, n, dim, 4, 50)
	query := make([]float32, dim)
	results := g.search(query, 5, 20)
	for _, id := range results {
		if id >= 50 {
			t.Errorf("query near cluster A returned node %d from cluster B", id)
		}
	}
}

func TestBuildGraph_Concurrent(t *testing.T) {
	g, _ := makeTestGraph(200)
	query := make([]float32, 14)
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results := g.search(query, 5, 20)
			if len(results) != 5 {
				t.Errorf("concurrent search: expected 5, got %d", len(results))
			}
		}()
	}
	wg.Wait()
}

func TestLayer0Neighbors_Sentinel(t *testing.T) {
	g, _ := makeTestGraph(10)
	for i := 0; i < 10; i++ {
		neighbors := g.layer0Neighbors(uint32(i))
		for _, n := range neighbors {
			if n != math.MaxUint32 && int(n) >= g.nodeCount {
				t.Errorf("invalid neighbor %d for node %d", n, i)
			}
		}
	}
}
