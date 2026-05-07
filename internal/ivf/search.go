package ivf

import "unsafe"

func (idx *Index) topCentroids(query [Dim]int16, nprobe int, out *[maxIVFProbe]uint32) int {
	if nprobe <= 0 {
		return 0
	}
	if nprobe > idx.ivf.clusters {
		nprobe = idx.ivf.clusters
	}
	if nprobe > maxIVFProbe {
		nprobe = maxIVFProbe
	}
	if useIVFAVX2 && len(idx.ivf.centroidBlocks) >= blocksForRows(idx.ivf.clusters)*blockStride {
		return idx.topCentroidsAVX2(query, nprobe, out)
	}
	return idx.topCentroidsScalar(query, nprobe, out)
}

func (idx *Index) topCentroidsScalar(query [Dim]int16, nprobe int, out *[maxIVFProbe]uint32) int {
	var bestDist [maxIVFProbe]int64
	for i := range nprobe {
		bestDist[i] = maxInt64
	}
	count := 0
	for c := range idx.ivf.clusters {
		base := c * Dim
		d := quantizedDistance(query, idx.ivf.centroids[base:base+Dim], bestDist[nprobe-1])
		if count < nprobe {
			insertCentroid(c, d, &bestDist, out, count)
			count++
		} else if d < bestDist[nprobe-1] {
			insertCentroid(c, d, &bestDist, out, nprobe-1)
		}
	}
	return count
}

func (idx *Index) topCentroidsAVX2(query [Dim]int16, nprobe int, out *[maxIVFProbe]uint32) int {
	var bestDist [maxIVFProbe]int64
	for i := range nprobe {
		bestDist[i] = maxInt64
	}
	centBlocks := unsafe.Pointer(unsafe.SliceData(idx.ivf.centroidBlocks))
	var dist [blockSize]int64
	count, c := 0, 0
	for ; c+blockSize <= idx.ivf.clusters; c += blockSize {
		quantizedBlock8DistancesAVX2(&query[0], unsafe.Add(centBlocks, (c/blockSize)*blockStride*2), &dist[0])
		for lane := range blockSize {
			d := dist[lane]
			cluster := c + lane
			if count < nprobe {
				insertCentroid(cluster, d, &bestDist, out, count)
				count++
			} else if d < bestDist[nprobe-1] {
				insertCentroid(cluster, d, &bestDist, out, nprobe-1)
			}
		}
	}
	for ; c < idx.ivf.clusters; c++ {
		base := c * Dim
		d := quantizedDistance(query, idx.ivf.centroids[base:base+Dim], bestDist[nprobe-1])
		if count < nprobe {
			insertCentroid(c, d, &bestDist, out, count)
			count++
		} else if d < bestDist[nprobe-1] {
			insertCentroid(c, d, &bestDist, out, nprobe-1)
		}
	}
	return count
}

func insertCentroid(cluster int, dist int64, bestDist *[maxIVFProbe]int64, bestID *[maxIVFProbe]uint32, last int) {
	i := last
	for i > 0 && dist < bestDist[i-1] {
		bestDist[i] = bestDist[i-1]
		bestID[i] = bestID[i-1]
		i--
	}
	bestDist[i] = dist
	bestID[i] = uint32(cluster)
}
