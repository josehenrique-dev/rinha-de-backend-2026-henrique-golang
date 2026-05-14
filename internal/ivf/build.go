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

func Build(vectors []float32, labels []uint8, nVectors int) (*Index, error) {
	if nVectors == 0 {
		return nil, errNoVectors
	}

	clusters := NClusters
	if clusters > nVectors {
		clusters = nVectors
	}

	qvecs := make([]int16, nVectors*Dim)
	for i := range nVectors {
		src := vectors[i*Dim : (i+1)*Dim]
		var v [Dim]float32
		copy(v[:], src)
		q := quantizeVector(v)
		copy(qvecs[i*Dim:(i+1)*Dim], q[:])
	}

	assign := trainKMeans(qvecs, nVectors, clusters)

	ids := make([]uint32, nVectors)
	for i := range nVectors {
		ids[i] = uint32(i)
	}

	ranges := make([]ivfBuildRange, clusters)
	clusterCounts := make([]int, clusters)
	for _, c := range assign {
		clusterCounts[c]++
	}

	pos := 0
	for c := range clusters {
		ranges[c].start = pos
		pos += clusterCounts[c]
		ranges[c].end = pos
	}

	clusterPos := make([]int, clusters)
	for i, c := range assign {
		idx := ranges[c].start + clusterPos[c]
		ids[idx] = uint32(i)
		clusterPos[c]++
	}

	return materializeIVF(qvecs, labels, ids, ranges, clusters), nil
}

func materializeIVF(vectors []int16, labels []uint8, ids []uint32, ranges []ivfBuildRange, clusters int) *Index {
	orderedVecs := make([]int16, len(vectors))
	orderedLabs := make([]uint8, len(labels))
	centroids := make([]int16, clusters*Dim)
	listOffsets := make([]uint32, clusters+1)
	blockOffsets := make([]uint32, clusters+1)
	bboxMin := make([]int16, clusters*Dim)
	bboxMax := make([]int16, clusters*Dim)
	origIDs := make([]uint32, len(labels))

	pos, blockPos := 0, 0
	for c, r := range ranges {
		listOffsets[c] = uint32(pos)
		blockOffsets[c] = uint32(blockPos)
		computeClusterStats(vectors, ids, r,
			centroids[c*Dim:(c+1)*Dim], bboxMin[c*Dim:(c+1)*Dim], bboxMax[c*Dim:(c+1)*Dim])

		clusterIDs := sortClusterByCentroidDist(vectors, ids[r.start:r.end], centroids[c*Dim:(c+1)*Dim])
		for _, origID := range clusterIDs {
			copy(orderedVecs[pos*Dim:(pos+1)*Dim], vectors[int(origID)*Dim:(int(origID)+1)*Dim])
			orderedLabs[pos] = labels[origID]
			origIDs[pos] = origID
			pos++
		}
		listOffsets[c+1] = uint32(pos)
		blockPos += blocksForRows(len(clusterIDs))
		blockOffsets[c+1] = uint32(blockPos)
	}

	blocks := buildIVFBlocks(orderedVecs, listOffsets, blockOffsets)
	centroidBlocks := buildCentroidBlocks(centroids, clusters)

	return &Index{
		blocks: blocks,
		labels: orderedLabs,
		ivf: ivfMeta{
			clusters:        clusters,
			nprobe:          8,
			ambiguousNprobe: 24,
			repair:          true,
			centroids:       centroids,
			centroidBlocks:  centroidBlocks,
			listOffsets:     listOffsets,
			blockOffsets:    blockOffsets,
			bboxMin:         bboxMin,
			bboxMax:         bboxMax,
			origIDs:         origIDs,
		},
	}
}

func highestPowerOfTwoLE(v int) int {
	p := 1
	for p*2 <= v {
		p *= 2
	}
	return p
}
