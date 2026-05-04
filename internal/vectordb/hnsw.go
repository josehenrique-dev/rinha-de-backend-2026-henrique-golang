package vectordb

import (
	"math"

	"github.com/coder/hnsw"
	"github.com/josehenrique-dev/rinha-2026/internal/loader"
)

type Index struct {
	graph *hnsw.Graph[uint32]
	ds    *loader.Dataset
}

func euclidean(a, b []float32) float32 {
	var sum float32
	for i := range a {
		d := a[i] - b[i]
		sum += d * d
	}
	return float32(math.Sqrt(float64(sum)))
}

func Build(ds *loader.Dataset) (*Index, error) {
	g := hnsw.NewGraph[uint32]()
	g.M = 16
	g.Distance = euclidean

	for i := 0; i < ds.Count; i++ {
		vec := ds.Vectors[i*ds.Dim : i*ds.Dim+ds.Dim]
		g.Add(hnsw.MakeNode(uint32(i), vec))
	}

	return &Index{graph: g, ds: ds}, nil
}

func (idx *Index) Search(query [14]float32, k int) float32 {
	neighbors := idx.graph.Search(query[:], k)

	fraudCount := 0
	for _, n := range neighbors {
		if idx.ds.Labels[n.Key] == 1 {
			fraudCount++
		}
	}
	return float32(fraudCount) / float32(k)
}
