package vectordb

import (
	"math"
	"testing"
)

func TestSquaredDist_Equal(t *testing.T) {
	a := []float32{1, 2, 3}
	if squaredDist(a, a) != 0 {
		t.Fatal("dist to self must be 0")
	}
}

func TestSquaredDist_Known(t *testing.T) {
	a := []float32{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	b := []float32{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	got := squaredDist(a, b)
	if got != 1.0 {
		t.Fatalf("expected 1.0, got %f", got)
	}
}

func TestSquaredDist_Sentinel(t *testing.T) {
	a := []float32{0, 0, 0, 0, 0, -1, -1, 0, 0, 0, 0, 0, 0, 0}
	b := []float32{0, 0, 0, 0, 0, -1, -1, 0, 0, 0, 0, 0, 0, 0}
	if squaredDist(a, b) != 0 {
		t.Fatal("equal sentinel vectors must have dist 0")
	}
}

func TestSquaredDist_Pythagorean(t *testing.T) {
	a := make([]float32, 14)
	b := make([]float32, 14)
	b[0] = 3
	b[1] = 4
	got := squaredDist(a, b)
	want := float32(25)
	if math.Abs(float64(got-want)) > 1e-6 {
		t.Fatalf("expected %f, got %f", want, got)
	}
}
