package ivf

import "unsafe"

type ivfSearchState struct {
	bestDist   [5]int64
	bestFraud  [5]bool
	bestID     [5]uint32
	probes     [maxIVFProbe]uint32
	probeMask  [128]uint64
	probeCount int
}

func newSearchState() ivfSearchState {
	maxU := ^uint32(0)
	return ivfSearchState{
		bestDist: [5]int64{maxInt64, maxInt64, maxInt64, maxInt64, maxInt64},
		bestID:   [5]uint32{maxU, maxU, maxU, maxU, maxU},
	}
}

func (s *ivfSearchState) countFrauds() int {
	n := 0
	for i := range 5 {
		if s.bestDist[i] != maxInt64 && s.bestFraud[i] {
			n++
		}
	}
	return n
}

func (s *ivfSearchState) hasProbe(c uint32) bool {
	return s.probeMask[c>>6]&(uint64(1)<<(c&63)) != 0
}

func (s *ivfSearchState) addProbe(c uint32) {
	s.probeMask[c>>6] |= uint64(1) << (c & 63)
	if s.probeCount < maxIVFProbe {
		s.probes[s.probeCount] = c
		s.probeCount++
	}
}

func (s *ivfSearchState) insert(d int64, fraud bool, origID uint32) {
	for i := range 5 {
		if d < s.bestDist[i] || (d == s.bestDist[i] && origID < s.bestID[i]) {
			for j := 4; j > i; j-- {
				s.bestDist[j] = s.bestDist[j-1]
				s.bestFraud[j] = s.bestFraud[j-1]
				s.bestID[j] = s.bestID[j-1]
			}
			s.bestDist[i] = d
			s.bestFraud[i] = fraud
			s.bestID[i] = origID
			return
		}
	}
}

func (idx *Index) scanCluster(query [Dim]int16, cluster int, state *ivfSearchState) {
	if len(idx.ivf.bboxMin) >= (cluster+1)*Dim {
		if idx.bboxDist(query, cluster, state.bestDist[4]) > state.bestDist[4] {
			return
		}
	}
	if useIVFAVX2 {
		idx.scanBlocksAVX2(query, cluster, state)
	} else {
		idx.scanBlocksScalar(query, cluster, state)
	}
}

func (idx *Index) scanBlocksAVX2(query [Dim]int16, cluster int, state *ivfSearchState) {
	rowStart := int(idx.ivf.listOffsets[cluster])
	rowEnd := int(idx.ivf.listOffsets[cluster+1])
	blockStart := int(idx.ivf.blockOffsets[cluster])
	blockEnd := int(idx.ivf.blockOffsets[cluster+1])
	if rowStart >= rowEnd {
		return
	}
	blocks := unsafe.Pointer(unsafe.SliceData(idx.blocks))
	origIDsPtr := unsafe.SliceData(idx.ivf.origIDs)

	var dist32 [blockSize * 4]int64
	block := blockStart
	for ; block+4 <= blockEnd; block += 4 {
		blockPtr := unsafe.Add(blocks, block*blockStride*2)
		quantizedBlock32DistancesAVX2(&query[0], blockPtr, &dist32[0])
		rowBase := rowStart + (block-blockStart)*blockSize
		lanes := blockSize * 4
		if rem := rowEnd - rowBase; rem < lanes {
			lanes = rem
		}
		for lane := range lanes {
			row := rowBase + lane
			d := dist32[lane]
			origID := uint32(row)
			if origIDsPtr != nil && row < len(idx.ivf.origIDs) {
				origID = *(*uint32)(unsafe.Add(unsafe.Pointer(origIDsPtr), row*4))
			}
			if d > state.bestDist[4] || (d == state.bestDist[4] && origID >= state.bestID[4]) {
				continue
			}
			fraud := idx.labels[row] == 1
			state.insert(d, fraud, origID)
		}
	}

	var dist [blockSize]int64
	for ; block < blockEnd; block++ {
		blockPtr := unsafe.Add(blocks, block*blockStride*2)
		quantizedBlock8DistancesAVX2(&query[0], blockPtr, &dist[0])
		rowBase := rowStart + (block-blockStart)*blockSize
		lanes := blockSize
		if rem := rowEnd - rowBase; rem < lanes {
			lanes = rem
		}
		for lane := range lanes {
			row := rowBase + lane
			d := dist[lane]
			origID := uint32(row)
			if origIDsPtr != nil && row < len(idx.ivf.origIDs) {
				origID = *(*uint32)(unsafe.Add(unsafe.Pointer(origIDsPtr), row*4))
			}
			if d > state.bestDist[4] || (d == state.bestDist[4] && origID >= state.bestID[4]) {
				continue
			}
			fraud := idx.labels[row] == 1
			state.insert(d, fraud, origID)
		}
	}
}

func (idx *Index) scanBlocksScalar(query [Dim]int16, cluster int, state *ivfSearchState) {
	rowStart := int(idx.ivf.listOffsets[cluster])
	rowEnd := int(idx.ivf.listOffsets[cluster+1])
	blockStart := int(idx.ivf.blockOffsets[cluster])
	blockEnd := int(idx.ivf.blockOffsets[cluster+1])
	if rowStart >= rowEnd {
		return
	}
	blocks := unsafe.Pointer(unsafe.SliceData(idx.blocks))
	origIDsPtr := unsafe.SliceData(idx.ivf.origIDs)
	for block := blockStart; block < blockEnd; block++ {
		blockPtr := unsafe.Add(blocks, block*blockStride*2)
		rowBase := rowStart + (block-blockStart)*blockSize
		lanes := blockSize
		if rem := rowEnd - rowBase; rem < lanes {
			lanes = rem
		}
		for lane := range lanes {
			row := rowBase + lane
			d := blockLaneDist(query, blockPtr, lane, state.bestDist[4])
			origID := uint32(row)
			if origIDsPtr != nil && row < len(idx.ivf.origIDs) {
				origID = *(*uint32)(unsafe.Add(unsafe.Pointer(origIDsPtr), row*4))
			}
			if d > state.bestDist[4] || (d == state.bestDist[4] && origID >= state.bestID[4]) {
				continue
			}
			fraud := idx.labels[row] == 1
			state.insert(d, fraud, origID)
		}
	}
}

func blockLaneDist(query [Dim]int16, block unsafe.Pointer, lane int, cutoff int64) int64 {
	var sum int64
	dimOrder := [Dim]int{5, 6, 2, 0, 7, 8, 12, 1, 3, 4, 9, 10, 11, 13}
	for _, d := range dimOrder {
		v := *(*int16)(unsafe.Add(block, (d*blockSize+lane)*2))
		delta := int64(query[d]) - int64(v)
		sum += delta * delta
		if sum >= cutoff {
			return sum
		}
	}
	return sum
}
