package ivf

import (
	"runtime"
	"sync"
)

// assignParallel assigns every vector to its nearest centroid in parallel.
func assignParallel(vectors []float32, nVectors int, centroids *[NClusters][Dim]float32, assignments []uint16) {
	nWorkers := runtime.NumCPU()
	if nWorkers < 1 {
		nWorkers = 1
	}
	chunkSize := (nVectors + nWorkers - 1) / nWorkers
	var wg sync.WaitGroup
	for w := 0; w < nWorkers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			start := w * chunkSize
			end := start + chunkSize
			if end > nVectors {
				end = nVectors
			}
			for i := start; i < end; i++ {
				vec := vectors[i*Dim : (i+1)*Dim]
				best, bestD := 0, float32(1e38)
				for c := 0; c < NClusters; c++ {
					d := distSliceCent(vec, &centroids[c])
					if d < bestD {
						bestD = d
						best = c
					}
				}
				assignments[i] = uint16(best)
			}
		}(w)
	}
	wg.Wait()
}

func trainCentroids(vectors []float32, nVectors int) [NClusters][Dim]float32 {
	centroids := initCentroids(vectors, nVectors)
	assignments := make([]uint16, nVectors)

	nWorkers := runtime.NumCPU()
	if nWorkers < 1 {
		nWorkers = 1
	}

	type localAcc struct {
		sums   [NClusters][Dim]float64
		counts [NClusters]int32
	}
	accs := make([]*localAcc, nWorkers)
	for i := range accs {
		accs[i] = new(localAcc)
	}

	chunkSize := (nVectors + nWorkers - 1) / nWorkers

	for iter := 0; iter < 15; iter++ {
		var wg sync.WaitGroup
		for w := 0; w < nWorkers; w++ {
			wg.Add(1)
			go func(w int) {
				defer wg.Done()
				start := w * chunkSize
				end := start + chunkSize
				if end > nVectors {
					end = nVectors
				}
				acc := accs[w]
				for c := range acc.sums {
					acc.sums[c] = [Dim]float64{}
					acc.counts[c] = 0
				}
				for i := start; i < end; i++ {
					vec := vectors[i*Dim : (i+1)*Dim]
					best, bestD := 0, float32(1e38)
					for c := 0; c < NClusters; c++ {
						d := distSliceCent(vec, &centroids[c])
						if d < bestD {
							bestD = d
							best = c
						}
					}
					assignments[i] = uint16(best)
					for d := 0; d < Dim; d++ {
						acc.sums[best][d] += float64(vec[d])
					}
					acc.counts[best]++
				}
			}(w)
		}
		wg.Wait()

		for c := 0; c < NClusters; c++ {
			var sumD [Dim]float64
			var count int64
			for _, acc := range accs {
				count += int64(acc.counts[c])
				for d := 0; d < Dim; d++ {
					sumD[d] += acc.sums[c][d]
				}
			}
			if count > 0 {
				for d := 0; d < Dim; d++ {
					centroids[c][d] = float32(sumD[d] / float64(count))
				}
			}
		}
	}

	return centroids
}

func initCentroids(vectors []float32, nVectors int) [NClusters][Dim]float32 {
	var centroids [NClusters][Dim]float32
	step := nVectors / NClusters
	if step < 1 {
		step = 1
	}
	for c := 0; c < NClusters; c++ {
		vi := (c * step) % nVectors
		copy(centroids[c][:], vectors[vi*Dim:(vi+1)*Dim])
	}
	return centroids
}

func distSliceCent(vec []float32, c *[Dim]float32) float32 {
	_ = vec[Dim-1]
	d0 := vec[0] - c[0]
	d1 := vec[1] - c[1]
	d2 := vec[2] - c[2]
	d3 := vec[3] - c[3]
	d4 := vec[4] - c[4]
	d5 := vec[5] - c[5]
	d6 := vec[6] - c[6]
	d7 := vec[7] - c[7]
	d8 := vec[8] - c[8]
	d9 := vec[9] - c[9]
	d10 := vec[10] - c[10]
	d11 := vec[11] - c[11]
	d12 := vec[12] - c[12]
	d13 := vec[13] - c[13]
	return d0*d0 + d1*d1 + d2*d2 + d3*d3 +
		d4*d4 + d5*d5 + d6*d6 + d7*d7 +
		d8*d8 + d9*d9 + d10*d10 + d11*d11 +
		d12*d12 + d13*d13
}
