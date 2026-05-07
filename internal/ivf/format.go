package ivf

import (
	"fmt"
	"log"
)

func Build(vectors []float32, labels []uint8, nVectors int) (*Index, error) {
	if nVectors == 0 {
		return nil, errNoVectors
	}
	log.Printf("build: %d vectors", nVectors)
	return &Index{}, fmt.Errorf("Build not implemented")
}

func Save(idx *Index, path string) error {
	return fmt.Errorf("Save not implemented: %s", path)
}

func Load(path string) (*Index, error) {
	return nil, fmt.Errorf("Load not implemented: %s", path)
}
