//go:build amd64

package vectordb

// squaredDist14 uses AVX to compute squared Euclidean distance for dim=14.
// Implemented in dist_amd64.s using 256-bit YMM registers (floats 0-7),
// 128-bit XMM (floats 8-11), and a 64-bit VMOVSD load (floats 12-13).
func squaredDist14(a, b []float32) float32
