package vectordb

type visitedTracker struct {
	visited []bool
	touched []uint32
}

func newVisitedTracker(size int) *visitedTracker {
	return &visitedTracker{
		visited: make([]bool, size),
		touched: make([]uint32, 0, 64),
	}
}

func (v *visitedTracker) isVisited(id uint32) bool {
	return v.visited[id]
}

func (v *visitedTracker) visit(id uint32) {
	v.visited[id] = true
	v.touched = append(v.touched, id)
}

func (v *visitedTracker) reset() {
	for _, id := range v.touched {
		v.visited[id] = false
	}
	v.touched = v.touched[:0]
}
