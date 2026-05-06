package vectordb

import (
	"fmt"

	"github.com/josehenrique-dev/rinha-2026/internal/loader"
)

const (
	defaultM              = 6
	defaultEfConstruction = 200
	defaultEfSearch       = 5
)

type Index struct {
	g        *graph
	indexMem []byte
}

func Build(ds *loader.Dataset) (*Index, error) {
	g := buildGraph(ds.Vectors, ds.Labels, ds.Count, ds.Dim, defaultM, defaultEfConstruction)
	return &Index{g: g}, nil
}

func Save(idx *Index, path string) error {
	if err := saveGraph(idx.g, path); err != nil {
		return fmt.Errorf("save graph: %w", err)
	}
	return nil
}

func Load(path string, ds *loader.Dataset) (*Index, error) {
	g, mem, err := loadGraph(path, ds.Vectors, ds.Labels, ds.Dim)
	if err != nil {
		return nil, fmt.Errorf("load graph: %w", err)
	}
	return &Index{g: g, indexMem: mem}, nil
}

func (idx *Index) Search(query [14]float32, k int) float32 {
	return float32(idx.SearchCount(query, k)) / float32(k)
}

func (idx *Index) SearchCount(query [14]float32, k int) int {
	var buf [5]uint32
	n := idx.g.searchInto(query[:], defaultEfSearch, buf[:k])
	fraudCount := 0
	for i := 0; i < n; i++ {
		if idx.g.labels[buf[i]] == 1 {
			fraudCount++
		}
	}
	return fraudCount
}

func (idx *Index) SearchWithEf(query [14]float32, k, ef int) int {
	var buf [5]uint32
	n := idx.g.searchInto(query[:], ef, buf[:k])
	fraudCount := 0
	for i := 0; i < n; i++ {
		if idx.g.labels[buf[i]] == 1 {
			fraudCount++
		}
	}
	return fraudCount
}

func (idx *Index) Close() {
	freeMmap(idx.indexMem)
}
