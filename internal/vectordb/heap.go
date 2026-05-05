package vectordb

import (
	"container/heap"
	"math"
	"sort"
)

type heapItem struct {
	id   uint32
	dist float32
}

type candidateHeap struct {
	items []heapItem
}

func newCandidateHeap(cap int) *candidateHeap {
	return &candidateHeap{items: make([]heapItem, 0, cap)}
}

func (h *candidateHeap) Len() int           { return len(h.items) }
func (h *candidateHeap) Less(i, j int) bool { return h.items[i].dist < h.items[j].dist }
func (h *candidateHeap) Swap(i, j int)      { h.items[i], h.items[j] = h.items[j], h.items[i] }
func (h *candidateHeap) Push(x any)         { h.items = append(h.items, x.(heapItem)) }
func (h *candidateHeap) Pop() any {
	n := len(h.items)
	x := h.items[n-1]
	h.items = h.items[:n-1]
	return x
}
func (h *candidateHeap) push(id uint32, dist float32) {
	heap.Push(h, heapItem{id, dist})
}
func (h *candidateHeap) popMin() heapItem {
	return heap.Pop(h).(heapItem)
}
func (h *candidateHeap) peekMin() float32 {
	if len(h.items) == 0 {
		return math.MaxFloat32
	}
	return h.items[0].dist
}
func (h *candidateHeap) empty() bool { return len(h.items) == 0 }

type resultHeap struct {
	items []heapItem
	cap   int
}

func newResultHeap(k int) *resultHeap {
	return &resultHeap{items: make([]heapItem, 0, k+1), cap: k}
}

func (h *resultHeap) Len() int           { return len(h.items) }
func (h *resultHeap) Less(i, j int) bool { return h.items[i].dist > h.items[j].dist }
func (h *resultHeap) Swap(i, j int)      { h.items[i], h.items[j] = h.items[j], h.items[i] }
func (h *resultHeap) Push(x any)         { h.items = append(h.items, x.(heapItem)) }
func (h *resultHeap) Pop() any {
	n := len(h.items)
	x := h.items[n-1]
	h.items = h.items[:n-1]
	return x
}
func (h *resultHeap) push(id uint32, dist float32) {
	heap.Push(h, heapItem{id, dist})
	if len(h.items) > h.cap {
		heap.Pop(h)
	}
}
func (h *resultHeap) worst() float32 {
	if len(h.items) == 0 {
		return 1e38
	}
	return h.items[0].dist
}
func (h *resultHeap) len() int { return len(h.items) }
func (h *resultHeap) ids() []uint32 {
	out := make([]uint32, len(h.items))
	for i, it := range h.items {
		out[i] = it.id
	}
	return out
}

// sortedIDs returns node IDs sorted by distance ascending (closest first).
// Used during build so that candidates[0] is the closest entry point
// and selectNeighbors picks the M truly nearest nodes.
func (h *resultHeap) sortedIDs() []uint32 {
	items := make([]heapItem, len(h.items))
	copy(items, h.items)
	sort.Slice(items, func(i, j int) bool { return items[i].dist < items[j].dist })
	out := make([]uint32, len(items))
	for i, it := range items {
		out[i] = it.id
	}
	return out
}
