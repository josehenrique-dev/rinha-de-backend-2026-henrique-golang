package ivf

import "sort"

type ivfBuildRange struct {
	start int
	end   int
}

func balancedSplit(vectors []int16, ids []uint32, ranges []ivfBuildRange, start, end, clusterBase, clusterCount int) {
	if clusterCount == 1 {
		ranges[clusterBase] = ivfBuildRange{start: start, end: end}
		return
	}
	dim := maxVarianceDimension(vectors, ids, start, end)
	window := ids[start:end]
	sort.Slice(window, func(i, j int) bool {
		l := vectors[int(window[i])*Dim+dim]
		r := vectors[int(window[j])*Dim+dim]
		if l == r {
			return window[i] < window[j]
		}
		return l < r
	})
	mid := start + (end-start)/2
	half := clusterCount / 2
	balancedSplit(vectors, ids, ranges, start, mid, clusterBase, half)
	balancedSplit(vectors, ids, ranges, mid, end, clusterBase+half, half)
}

func maxVarianceDimension(vectors []int16, ids []uint32, start, end int) int {
	var sums [Dim]int64
	var sumSq [Dim]int64
	for i := start; i < end; i++ {
		base := int(ids[i]) * Dim
		for d := range Dim {
			v := int64(vectors[base+d])
			sums[d] += v
			sumSq[d] += v * v
		}
	}
	count := float64(end - start)
	best, bestVar := 0, -1.0
	for d := range Dim {
		mean := float64(sums[d]) / count
		variance := float64(sumSq[d])/count - mean*mean
		if variance > bestVar {
			bestVar = variance
			best = d
		}
	}
	return best
}

func computeClusterStats(vectors []int16, ids []uint32, r ivfBuildRange, centroid, bboxMin, bboxMax []int16) {
	for d := range Dim {
		bboxMin[d] = 32767
		bboxMax[d] = -32768
	}
	var sums [Dim]int64
	for i := r.start; i < r.end; i++ {
		base := int(ids[i]) * Dim
		for d := range Dim {
			v := vectors[base+d]
			sums[d] += int64(v)
			if v < bboxMin[d] {
				bboxMin[d] = v
			}
			if v > bboxMax[d] {
				bboxMax[d] = v
			}
		}
	}
	count := int64(r.end - r.start)
	for d := range Dim {
		if count > 0 {
			centroid[d] = int16((sums[d] + count/2) / count)
		}
	}
}

func sortClusterByCentroidDist(vectors []int16, ids []uint32, centroid []int16) []uint32 {
	type pair struct {
		id   uint32
		dist int64
	}
	items := make([]pair, len(ids))
	for i, id := range ids {
		base := int(id) * Dim
		var sum int64
		for d := range Dim {
			delta := int64(vectors[base+d]) - int64(centroid[d])
			sum += delta * delta
		}
		items[i] = pair{id, sum}
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].dist == items[j].dist {
			return items[i].id < items[j].id
		}
		return items[i].dist < items[j].dist
	})
	out := make([]uint32, len(ids))
	for i, it := range items {
		out[i] = it.id
	}
	return out
}

func buildIVFBlocks(vectors []int16, listOffsets, blockOffsets []uint32) []int16 {
	blockCount := int(blockOffsets[len(blockOffsets)-1])
	blocks := make([]int16, blockCount*blockStride)
	for c := 0; c+1 < len(listOffsets); c++ {
		rowStart := int(listOffsets[c])
		rowEnd := int(listOffsets[c+1])
		blockStart := int(blockOffsets[c])
		for row := rowStart; row < rowEnd; row++ {
			rel := row - rowStart
			block := blockStart + rel/blockSize
			lane := rel % blockSize
			blockBase := block * blockStride
			vectorBase := row * Dim
			for d := range Dim {
				blocks[blockBase+d*blockSize+lane] = vectors[vectorBase+d]
			}
		}
	}
	return blocks
}

func buildCentroidBlocks(centroids []int16, clusters int) []int16 {
	blockCount := blocksForRows(clusters)
	blocks := make([]int16, blockCount*blockStride)
	for c := range clusters {
		block := c / blockSize
		lane := c % blockSize
		blockBase := block * blockStride
		centBase := c * Dim
		for d := range Dim {
			blocks[blockBase+d*blockSize+lane] = centroids[centBase+d]
		}
	}
	return blocks
}
