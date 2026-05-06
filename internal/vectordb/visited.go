package vectordb

type visitedTracker struct {
	bits    []uint64
	touched []uint32
}

func newVisitedTracker(size int) *visitedTracker {
	words := (size + 63) / 64
	return &visitedTracker{
		bits:    make([]uint64, words),
		touched: make([]uint32, 0, 256),
	}
}

func (v *visitedTracker) isVisited(id uint32) bool {
	return v.bits[id>>6]&(1<<(id&63)) != 0
}

func (v *visitedTracker) visit(id uint32) {
	v.bits[id>>6] |= 1 << (id & 63)
	v.touched = append(v.touched, id)
}

func (v *visitedTracker) reset() {
	for _, id := range v.touched {
		v.bits[id>>6] &^= 1 << (id & 63)
	}
	v.touched = v.touched[:0]
}
