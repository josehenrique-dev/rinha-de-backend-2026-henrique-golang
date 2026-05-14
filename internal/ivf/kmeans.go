package ivf

import "math"

const kMeansSampleSize = 65536
const kMeansIters = 6

var kMeansRNGSeed = [2]uint64{0xCAFEBABEDEADBEEF, 0xC0FFEE0123456789}

type pcgRng struct {
	state, inc uint64
}

func newPCG(s, i uint64) *pcgRng {
	r := &pcgRng{0, i | 1}
	r.next()
	r.state += s
	r.next()
	return r
}

func (r *pcgRng) next() uint32 {
	old := r.state
	r.state = old*6364136223846793005 + r.inc
	xorshifted := uint32(((old >> 18) ^ old) >> 27)
	rot := uint32(old >> 59)
	return (xorshifted >> rot) | (xorshifted << ((-rot) & 31))
}

func (r *pcgRng) intn(n int) int {
	return int(r.next() % uint32(n))
}

func (r *pcgRng) float64() float64 {
	return float64(r.next()) / float64(1<<32)
}

func trainKMeans(qvecs []int16, nVectors, clusters int) []int {
	rng := newPCG(kMeansRNGSeed[0], kMeansRNGSeed[1])

	sampleSize := kMeansSampleSize
	if sampleSize > nVectors {
		sampleSize = nVectors
	}

	sampleIDs := drawSample(nVectors, sampleSize, rng)

	sampleVecs := make([]float32, sampleSize*Dim)
	for i, id := range sampleIDs {
		for d := range Dim {
			sampleVecs[i*Dim+d] = float32(qvecs[id*Dim+d])
		}
	}

	centroids := kMeansPlusPlus(sampleVecs, sampleSize, clusters, rng)

	assign := make([]int, sampleSize)
	for iter := 0; iter < kMeansIters; iter++ {
		for i := range sampleSize {
			bestClust, bestDist := 0, math.MaxFloat64
			for c := range clusters {
				var dist float64
				for d := range Dim {
					delta := float64(sampleVecs[i*Dim+d]) - float64(centroids[c*Dim+d])
					dist += delta * delta
				}
				if dist < bestDist {
					bestDist = dist
					bestClust = c
				}
			}
			assign[i] = bestClust
		}

		for c := range clusters {
			var sums [Dim]float64
			count := 0
			for i := range sampleSize {
				if assign[i] == c {
					for d := range Dim {
						sums[d] += float64(sampleVecs[i*Dim+d])
					}
					count++
				}
			}
			if count > 0 {
				for d := range Dim {
					centroids[c*Dim+d] = float32(sums[d] / float64(count))
				}
			}
		}
	}

	fullAssign := make([]int, nVectors)
	for i := range nVectors {
		bestClust, bestDist := 0, math.MaxFloat64
		for c := range clusters {
			var dist float64
			for d := range Dim {
				delta := float32(qvecs[i*Dim+d]) - centroids[c*Dim+d]
				dist += float64(delta * delta)
			}
			if dist < bestDist {
				bestDist = dist
				bestClust = c
			}
		}
		fullAssign[i] = bestClust
	}

	return fullAssign
}

func drawSample(nVectors, sampleSize int, rng *pcgRng) []int {
	picked := make(map[int]bool)
	result := make([]int, sampleSize)

	for i := 0; i < sampleSize; i++ {
		var id int
		for {
			id = rng.intn(nVectors)
			if !picked[id] {
				break
			}
		}
		picked[id] = true
		result[i] = id
	}
	return result
}

func kMeansPlusPlus(sampleVecs []float32, sampleSize, clusters int, rng *pcgRng) []float32 {
	centroids := make([]float32, clusters*Dim)

	chosenID := rng.intn(sampleSize)
	for d := range Dim {
		centroids[0*Dim+d] = sampleVecs[chosenID*Dim+d]
	}

	dists := make([]float64, sampleSize)

	for c := 1; c < clusters; c++ {
		for i := range sampleSize {
			bestDist := math.MaxFloat64
			for prevC := 0; prevC < c; prevC++ {
				var dist float64
				for d := range Dim {
					delta := sampleVecs[i*Dim+d] - centroids[prevC*Dim+d]
					dist += float64(delta * delta)
				}
				if dist < bestDist {
					bestDist = dist
				}
			}
			dists[i] = bestDist
		}

		var totalDist float64
		for i := range sampleSize {
			totalDist += dists[i]
		}

		target := rng.float64() * totalDist
		var cumsum float64
		chosenID = 0
		for i := range sampleSize {
			cumsum += dists[i]
			if cumsum >= target {
				chosenID = i
				break
			}
		}

		for d := range Dim {
			centroids[c*Dim+d] = sampleVecs[chosenID*Dim+d]
		}
	}

	return centroids
}
