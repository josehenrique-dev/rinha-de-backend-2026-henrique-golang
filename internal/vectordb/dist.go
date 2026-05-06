package vectordb

// squaredDist computes squared Euclidean distance for slices of any length.
// Used by tests and the generic searchLayer path.
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

// squaredDist14 is an unrolled, bounds-check-free implementation for dim=14.
func squaredDist14(a, b []float32) float32 {
	_ = a[13]
	_ = b[13]
	d0 := a[0] - b[0]
	d1 := a[1] - b[1]
	d2 := a[2] - b[2]
	d3 := a[3] - b[3]
	d4 := a[4] - b[4]
	d5 := a[5] - b[5]
	d6 := a[6] - b[6]
	d7 := a[7] - b[7]
	d8 := a[8] - b[8]
	d9 := a[9] - b[9]
	d10 := a[10] - b[10]
	d11 := a[11] - b[11]
	d12 := a[12] - b[12]
	d13 := a[13] - b[13]
	return d0*d0 + d1*d1 + d2*d2 + d3*d3 +
		d4*d4 + d5*d5 + d6*d6 + d7*d7 +
		d8*d8 + d9*d9 + d10*d10 + d11*d11 +
		d12*d12 + d13*d13
}
