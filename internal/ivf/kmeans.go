package ivf

import "math/rand"

func trainCentroids(vectors []float32, nVectors int, rng *rand.Rand) [NClusters][Dim]float32 {
	sampleSize := nVectors
	if sampleSize > 200000 {
		sampleSize = 200000
	}

	perm := make([]int, sampleSize)
	for i := range perm {
		perm[i] = rng.Intn(nVectors)
	}

	var centroids [NClusters][Dim]float32
	for c := 0; c < NClusters; c++ {
		vi := perm[c%sampleSize]
		copy(centroids[c][:], vectors[vi*Dim:(vi+1)*Dim])
	}

	assignments := make([]int16, sampleSize)
	var sums [NClusters][Dim]float64
	var counts [NClusters]int32

	for iter := 0; iter < 20; iter++ {
		for si, vi := range perm {
			vec := vectors[vi*Dim : (vi+1)*Dim]
			best, bestD := 0, float32(1e38)
			for c := 0; c < NClusters; c++ {
				d := distSliceCent(vec, &centroids[c])
				if d < bestD {
					bestD = d
					best = c
				}
			}
			assignments[si] = int16(best)
		}

		for c := range sums {
			sums[c] = [Dim]float64{}
			counts[c] = 0
		}
		for si, vi := range perm {
			c := assignments[si]
			vec := vectors[vi*Dim : (vi+1)*Dim]
			for d := 0; d < Dim; d++ {
				sums[c][d] += float64(vec[d])
			}
			counts[c]++
		}
		for c := 0; c < NClusters; c++ {
			if counts[c] > 0 {
				for d := 0; d < Dim; d++ {
					centroids[c][d] = float32(sums[c][d] / float64(counts[c]))
				}
			} else {
				vi := perm[rng.Intn(sampleSize)]
				copy(centroids[c][:], vectors[vi*Dim:(vi+1)*Dim])
			}
		}
	}

	return centroids
}

func distSliceCent(vec []float32, c *[Dim]float32) float32 {
	_ = vec[Dim-1]
	var s float32
	for i := 0; i < Dim; i++ {
		d := vec[i] - c[i]
		s += d * d
	}
	return s
}
