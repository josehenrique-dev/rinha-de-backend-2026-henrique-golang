package ivf

import (
	"fmt"
	"syscall"
)

const (
	Dim         = 14
	NClusters   = 8192
	blockSize   = 8
	blockStride = Dim * blockSize
	maxIVFProbe = 128
)

type ivfMeta struct {
	clusters        int
	nprobe          int
	ambiguousNprobe int
	repair          bool
	centroids       []int16
	centroidBlocks  []int16
	listOffsets     []uint32
	blockOffsets    []uint32
	bboxMin         []int16
	bboxMax         []int16
	origIDs         []uint32
}

type Index struct {
	blocks []int16
	labels []uint8
	ivf    ivfMeta
	mem    []byte
}

func (idx *Index) Close() {
	if idx.mem != nil {
		syscall.Munmap(idx.mem)
		idx.mem = nil
	}
}

func (idx *Index) SearchCount(query [Dim]float32, _ int) int {
	q := quantizeVector(query)
	return idx.searchIVF(q)
}

func (idx *Index) hasBlocks() bool {
	return len(idx.ivf.blockOffsets) == idx.ivf.clusters+1 && len(idx.blocks) > 0
}

func blocksForRows(rows int) int {
	if rows <= 0 {
		return 0
	}
	return (rows + blockSize - 1) / blockSize
}

var errNoVectors = fmt.Errorf("no vectors")

func (idx *Index) searchIVF(q [Dim]int16) int {
	return 0
}
