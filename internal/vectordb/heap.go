package vectordb

import "sort"

type heapItem struct {
	id   uint32
	dist float32
}

// candidateHeap is a min-heap ordered by dist (closest first).
type candidateHeap struct {
	items []heapItem
}

func newCandidateHeap(cap int) *candidateHeap {
	return &candidateHeap{items: make([]heapItem, 0, cap)}
}

func (h *candidateHeap) reset() { h.items = h.items[:0] }

func (h *candidateHeap) push(id uint32, dist float32) {
	h.items = append(h.items, heapItem{id, dist})
	h.siftUp(len(h.items) - 1)
}

func (h *candidateHeap) popMin() heapItem {
	top := h.items[0]
	n := len(h.items) - 1
	h.items[0] = h.items[n]
	h.items = h.items[:n]
	if n > 0 {
		h.siftDown(0)
	}
	return top
}

func (h *candidateHeap) empty() bool { return len(h.items) == 0 }

func (h *candidateHeap) siftUp(i int) {
	for i > 0 {
		parent := (i - 1) >> 1
		if h.items[i].dist >= h.items[parent].dist {
			break
		}
		h.items[i], h.items[parent] = h.items[parent], h.items[i]
		i = parent
	}
}

func (h *candidateHeap) siftDown(i int) {
	n := len(h.items)
	for {
		left := (i << 1) + 1
		if left >= n {
			break
		}
		j := left
		if right := left + 1; right < n && h.items[right].dist < h.items[left].dist {
			j = right
		}
		if h.items[i].dist <= h.items[j].dist {
			break
		}
		h.items[i], h.items[j] = h.items[j], h.items[i]
		i = j
	}
}

// resultHeap is a max-heap ordered by dist (farthest at root), capped at cap entries.
type resultHeap struct {
	items []heapItem
	cap   int
}

func newResultHeap(k int) *resultHeap {
	return &resultHeap{items: make([]heapItem, 0, k+1), cap: k}
}

func (h *resultHeap) reset() { h.items = h.items[:0] }

func (h *resultHeap) push(id uint32, dist float32) {
	h.items = append(h.items, heapItem{id, dist})
	h.siftUp(len(h.items) - 1)
	if len(h.items) > h.cap {
		h.popMax()
	}
}

func (h *resultHeap) popMax() heapItem {
	top := h.items[0]
	n := len(h.items) - 1
	h.items[0] = h.items[n]
	h.items = h.items[:n]
	if n > 0 {
		h.siftDown(0)
	}
	return top
}

func (h *resultHeap) worst() float32 {
	if len(h.items) < h.cap {
		return 1e38
	}
	return h.items[0].dist
}

func (h *resultHeap) len() int { return len(h.items) }

func (h *resultHeap) siftUp(i int) {
	for i > 0 {
		parent := (i - 1) >> 1
		if h.items[i].dist <= h.items[parent].dist {
			break
		}
		h.items[i], h.items[parent] = h.items[parent], h.items[i]
		i = parent
	}
}

func (h *resultHeap) siftDown(i int) {
	n := len(h.items)
	for {
		left := (i << 1) + 1
		if left >= n {
			break
		}
		j := left
		if right := left + 1; right < n && h.items[right].dist > h.items[left].dist {
			j = right
		}
		if h.items[i].dist >= h.items[j].dist {
			break
		}
		h.items[i], h.items[j] = h.items[j], h.items[i]
		i = j
	}
}

func (h *resultHeap) ids() []uint32 {
	out := make([]uint32, len(h.items))
	for i, it := range h.items {
		out[i] = it.id
	}
	return out
}

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
