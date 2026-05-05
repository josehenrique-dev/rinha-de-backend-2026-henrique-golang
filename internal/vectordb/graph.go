package vectordb

import (
	"math"
	"math/rand"
	"sync"
)

const sentinel = math.MaxUint32

type graph struct {
	layer0     []uint32
	upperNodes []upperNode
	upperIndex map[uint32]int
	vectors    []float32
	labels     []uint8
	M          int
	M0         int
	nodeCount  int
	dim        int
	entryPoint uint32
	maxLayer   int
	pool       sync.Pool
}

type upperNode struct {
	nodeID    uint32
	maxLayer  uint8
	neighbors [][]uint32
}

func buildGraph(vectors []float32, labels []uint8, nodeCount, dim, M, efConstruction int) *graph {
	M0 := M * 2
	ml := 1.0 / math.Log(float64(M))

	g := &graph{
		layer0:     make([]uint32, nodeCount*M0),
		upperIndex: make(map[uint32]int),
		vectors:    vectors,
		labels:     labels,
		M:          M,
		M0:         M0,
		nodeCount:  nodeCount,
		dim:        dim,
	}

	for i := range g.layer0 {
		g.layer0[i] = sentinel
	}

	g.pool = sync.Pool{
		New: func() any { return newVisitedTracker(nodeCount) },
	}

	for i := 0; i < nodeCount; i++ {
		g.insertNode(uint32(i), ml, efConstruction)
	}
	return g
}

func (g *graph) insertNode(id uint32, ml float64, ef int) {
	layer := int(math.Floor(-math.Log(rand.Float64()) * ml))

	if id == 0 {
		if layer > 0 {
			g.assignUpperNode(id, layer)
		}
		g.maxLayer = layer
		g.entryPoint = id
		return
	}

	ep := g.entryPoint

	for l := g.maxLayer; l > layer; l-- {
		ep = g.greedyClosest(ep, g.vec(id), l)
	}

	for l := minInt(layer, g.maxLayer); l >= 0; l-- {
		Ml := g.M
		if l == 0 {
			Ml = g.M0
		}
		candidates := g.searchLayer(g.vec(id), ep, ef, l)
		neighbors := g.selectNeighbors(candidates, Ml)
		g.connectNeighbors(id, neighbors, l)
		for _, n := range neighbors {
			g.pruneNeighbors(n, Ml, l)
		}
		if len(candidates) > 0 {
			ep = candidates[0]
		}
	}

	if layer > g.maxLayer {
		g.assignUpperNode(id, layer)
		g.maxLayer = layer
		g.entryPoint = id
	} else if layer > 0 {
		g.assignUpperNode(id, layer)
	}
}

func (g *graph) vec(id uint32) []float32 {
	base := int(id) * g.dim
	return g.vectors[base : base+g.dim]
}

func (g *graph) layer0Neighbors(id uint32) []uint32 {
	base := int(id) * g.M0
	return g.layer0[base : base+g.M0]
}

func (g *graph) upperNeighbors(id uint32, layer int) []uint32 {
	idx, ok := g.upperIndex[id]
	if !ok || layer < 1 || layer > int(g.upperNodes[idx].maxLayer) {
		return nil
	}
	return g.upperNodes[idx].neighbors[layer-1]
}

func (g *graph) assignUpperNode(id uint32, maxLayer int) {
	neighbors := make([][]uint32, maxLayer)
	for l := range neighbors {
		neighbors[l] = make([]uint32, g.M)
		for i := range neighbors[l] {
			neighbors[l][i] = sentinel
		}
	}
	g.upperNodes = append(g.upperNodes, upperNode{
		nodeID:    id,
		maxLayer:  uint8(maxLayer),
		neighbors: neighbors,
	})
	g.upperIndex[id] = len(g.upperNodes) - 1
}

func (g *graph) greedyClosest(ep uint32, query []float32, layer int) uint32 {
	best := ep
	bestDist := squaredDist(query, g.vec(ep))
	for {
		improved := false
		var neighbors []uint32
		if layer == 0 {
			neighbors = g.layer0Neighbors(best)
		} else {
			neighbors = g.upperNeighbors(best, layer)
		}
		for _, n := range neighbors {
			if n == sentinel {
				break
			}
			d := squaredDist(query, g.vec(n))
			if d < bestDist {
				bestDist = d
				best = n
				improved = true
			}
		}
		if !improved {
			break
		}
	}
	return best
}

func (g *graph) searchLayer(query []float32, ep uint32, ef, layer int) []uint32 {
	vt := g.pool.Get().(*visitedTracker)
	defer func() { vt.reset(); g.pool.Put(vt) }()

	cands := newCandidateHeap(ef * 2)
	res := newResultHeap(ef)

	d := squaredDist(query, g.vec(ep))
	cands.push(ep, d)
	res.push(ep, d)
	vt.visit(ep)

	for !cands.empty() {
		c := cands.popMin()
		if c.dist > res.worst() {
			break
		}
		var neighbors []uint32
		if layer == 0 {
			neighbors = g.layer0Neighbors(c.id)
		} else {
			neighbors = g.upperNeighbors(c.id, layer)
		}
		for _, n := range neighbors {
			if n == sentinel {
				break
			}
			if vt.isVisited(n) {
				continue
			}
			vt.visit(n)
			nd := squaredDist(query, g.vec(n))
			if nd <= res.worst() {
				cands.push(n, nd)
				res.push(n, nd)
			}
		}
	}
	return res.ids()
}

func (g *graph) selectNeighbors(candidates []uint32, M int) []uint32 {
	if len(candidates) <= M {
		return candidates
	}
	return candidates[:M]
}

func (g *graph) connectNeighbors(id uint32, neighbors []uint32, layer int) {
	if layer == 0 {
		slots := g.layer0Neighbors(id)
		for i, n := range neighbors {
			if i >= len(slots) {
				break
			}
			slots[i] = n
		}
	} else {
		idx, ok := g.upperIndex[id]
		if !ok {
			return
		}
		if layer-1 >= len(g.upperNodes[idx].neighbors) {
			return
		}
		slots := g.upperNodes[idx].neighbors[layer-1]
		for i, n := range neighbors {
			if i >= len(slots) {
				break
			}
			slots[i] = n
		}
	}
	for _, n := range neighbors {
		if layer == 0 {
			g.addNeighborIfSlot(n, id, g.M0, 0)
		} else {
			g.addNeighborIfSlot(n, id, g.M, layer)
		}
	}
}

func (g *graph) addNeighborIfSlot(nodeID, newNeighbor uint32, M, layer int) {
	var slots []uint32
	if layer == 0 {
		slots = g.layer0Neighbors(nodeID)
	} else {
		slots = g.upperNeighbors(nodeID, layer)
	}
	for i, s := range slots {
		if s == sentinel {
			slots[i] = newNeighbor
			return
		}
	}
}

func (g *graph) pruneNeighbors(id uint32, M, layer int) {
	var slots []uint32
	if layer == 0 {
		slots = g.layer0Neighbors(id)
	} else {
		slots = g.upperNeighbors(id, layer)
	}
	count := 0
	for _, s := range slots {
		if s != sentinel {
			count++
		}
	}
	if count <= M {
		return
	}
	query := g.vec(id)
	worstIdx := -1
	worstDist := float32(-1)
	for i, s := range slots {
		if s == sentinel {
			continue
		}
		d := squaredDist(query, g.vec(s))
		if d > worstDist {
			worstDist = d
			worstIdx = i
		}
	}
	if worstIdx >= 0 {
		slots[worstIdx] = sentinel
	}
}

func (g *graph) search(query []float32, k, efSearch int) []uint32 {
	ep := g.entryPoint
	for l := g.maxLayer; l > 0; l-- {
		ep = g.greedyClosest(ep, query, l)
	}

	vt := g.pool.Get().(*visitedTracker)
	defer func() { vt.reset(); g.pool.Put(vt) }()

	cands := newCandidateHeap(efSearch * 2)
	res := newResultHeap(k)

	d := squaredDist(query, g.vec(ep))
	cands.push(ep, d)
	res.push(ep, d)
	vt.visit(ep)

	for !cands.empty() {
		c := cands.popMin()
		if c.dist > res.worst() {
			break
		}
		for _, n := range g.layer0Neighbors(c.id) {
			if n == sentinel {
				break
			}
			if vt.isVisited(n) {
				continue
			}
			vt.visit(n)
			nd := squaredDist(query, g.vec(n))
			if nd <= res.worst() {
				cands.push(n, nd)
				res.push(n, nd)
			}
		}
	}
	return res.ids()
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
