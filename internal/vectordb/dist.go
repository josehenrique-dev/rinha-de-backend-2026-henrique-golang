package vectordb

// squaredDist computes squared Euclidean distance for slices of any length.
// For dim=14 it delegates to squaredDist14 (AVX on amd64, unrolled otherwise).
func squaredDist(a, b []float32) float32 {
	if len(a) == 14 {
		return squaredDist14(a, b)
	}
	var s float32
	for i := range a {
		d := a[i] - b[i]
		s += d * d
	}
	return s
}
