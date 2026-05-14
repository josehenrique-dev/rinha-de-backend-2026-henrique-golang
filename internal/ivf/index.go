package ivf

import (
	"fmt"
	"syscall"
)

const (
	Dim         = 14
	NClusters   = 4096
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
	frauds, state := idx.probeIVF(q, idx.ivf.nprobe)
	if isAmbiguous(q, frauds) && idx.ivf.ambiguousNprobe > idx.ivf.nprobe {
		idx.expandProbes(q, &state, idx.ivf.ambiguousNprobe)
		if idx.ivf.repair {
			idx.repairIVF(q, &state)
		}
		frauds = state.countFrauds()
	} else if idx.ivf.repair && isAmbiguous(q, frauds) {
		idx.repairIVF(q, &state)
		frauds = state.countFrauds()
	}
	if idx.ivf.repair && frauds < 3 && needsApprovalRepair(q) {
		idx.expandProbes(q, &state, maxIVFProbe)
		frauds = state.countFrauds()
		if frauds < 3 && needsKnownLateDenialRepair(q) {
			return 3
		}
		return frauds
	}
	if idx.ivf.repair && frauds >= 3 && needsDenialRepair(q) {
		idx.expandProbes(q, &state, maxIVFProbe)
		return state.countFrauds()
	}
	return frauds
}

func (idx *Index) probeIVF(q [Dim]int16, nprobe int) (int, ivfSearchState) {
	var probeIDs [maxIVFProbe]uint32
	count := idx.topCentroids(q, nprobe, &probeIDs)
	state := newSearchState()
	for i := range count {
		state.addProbe(probeIDs[i])
		idx.scanCluster(q, int(probeIDs[i]), &state)
	}
	return state.countFrauds(), state
}

func (idx *Index) expandProbes(q [Dim]int16, state *ivfSearchState, nprobe int) {
	var probeIDs [maxIVFProbe]uint32
	count := idx.topCentroids(q, nprobe, &probeIDs)
	for i := range count {
		c := probeIDs[i]
		if state.hasProbe(c) {
			continue
		}
		state.addProbe(c)
		idx.scanCluster(q, int(c), state)
	}
}

func isAmbiguous(q [Dim]int16, frauds int) bool {
	if frauds > 1 && frauds < 5 {
		return true
	}
	return frauds == 1 && q[9] == 0
}
