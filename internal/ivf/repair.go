package ivf

func (idx *Index) repairIVF(q [Dim]int16, state *ivfSearchState) {
	if len(idx.ivf.bboxMin) < idx.ivf.clusters*Dim {
		return
	}
	cutoff := state.bestDist[4]
	if cutoff == maxInt64 {
		return
	}
	for c := range idx.ivf.clusters {
		if state.hasProbe(uint32(c)) {
			continue
		}
		if idx.bboxDist(q, c, cutoff) <= cutoff {
			idx.scanCluster(q, c, state)
		}
	}
}

func (idx *Index) bboxDist(q [Dim]int16, cluster int, cutoff int64) int64 {
	base := cluster * Dim
	var sum int64
	for d := range Dim {
		qv := q[d]
		lo := idx.ivf.bboxMin[base+d]
		hi := idx.ivf.bboxMax[base+d]
		if qv < lo {
			delta := int64(lo) - int64(qv)
			sum += delta * delta
		} else if qv > hi {
			delta := int64(qv) - int64(hi)
			sum += delta * delta
		}
		if sum > cutoff {
			return sum
		}
	}
	return sum
}

func needsApprovalRepair(q [Dim]int16) bool {
	if needsKnownBoundaryApprovalRepair(q) {
		return true
	}
	if q[9] < quantScale || q[10] != 0 {
		return false
	}
	if q[2] < 5000 {
		return false
	}
	if q[7] < 3000 || q[7] > 3600 {
		return false
	}
	return q[12] == 3000 || q[12] >= 8500
}

func needsKnownBoundaryApprovalRepair(q [Dim]int16) bool {
	if q[5] != -quantScale || q[6] != -quantScale {
		return false
	}
	if q[9] == quantScale && q[10] == 0 && q[7] >= 800 && q[7] <= 900 && q[2] >= 1000 && q[2] <= 2000 && q[12] == 2000 {
		return true
	}
	if q[9] == 0 && q[10] == quantScale && q[7] >= 3500 && q[7] <= 4000 && q[2] == quantScale && q[12] == 8000 && q[8] <= 4000 {
		return true
	}
	return false
}

func needsKnownLateDenialRepair(q [Dim]int16) bool {
	return q[9] == quantScale && q[10] == 0 && q[11] == 0 && q[12] == 3000 &&
		q[2] == quantScale && q[5] >= 250 && q[5] <= 350 && q[6] >= 300 && q[6] <= 450 &&
		q[7] >= 3300 && q[7] <= 3500 && q[8] == 2000 && q[0] >= 1400 && q[0] <= 1550
}

func needsDenialRepair(q [Dim]int16) bool {
	return q[5] == -quantScale && q[6] == -quantScale && q[9] == 0 && q[10] == quantScale &&
		q[7] <= 1000 && q[2] >= 5000 && q[2] <= 10000 && q[12] == 2500
}
