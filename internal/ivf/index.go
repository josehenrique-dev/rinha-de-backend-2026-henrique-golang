package ivf

import (
	"fmt"
	"log"
	"syscall"
)

const (
	Dim       = 14
	NClusters = 8192
	scaleF    = float32(32767)
)

type Index struct {
	centroids [NClusters][Dim]float32
	sizes     [NClusters]uint32
	offsets   [NClusters]uint32
	vecs      []int16
	labs      []uint8
	mem       []byte
}

func (idx *Index) Close() {
	if idx.mem != nil {
		syscall.Munmap(idx.mem)
		idx.mem = nil
	}
}

type knnHeap struct {
	buf    [5]struct{ d int32; i uint32 }
	n      int
	worstD int32
	worstP int
}

func newKnnHeap() knnHeap {
	return knnHeap{worstD: 1<<31 - 1}
}

func (h *knnHeap) push(d int32, i uint32) {
	if h.n < len(h.buf) {
		h.buf[h.n] = struct{ d int32; i uint32 }{d, i}
		h.n++
		if h.n == len(h.buf) {
			h.worstP = 0
			for j := 1; j < h.n; j++ {
				if h.buf[j].d > h.buf[h.worstP].d {
					h.worstP = j
				}
			}
			h.worstD = h.buf[h.worstP].d
		}
	} else if d < h.worstD {
		h.buf[h.worstP] = struct{ d int32; i uint32 }{d, i}
		h.worstP = 0
		for j := 1; j < h.n; j++ {
			if h.buf[j].d > h.buf[h.worstP].d {
				h.worstP = j
			}
		}
		h.worstD = h.buf[h.worstP].d
	}
}

func (h *knnHeap) fraudCount(labs []uint8) int {
	fc := 0
	for i := range h.n {
		if labs[h.buf[i].i] == 1 {
			fc++
		}
	}
	return fc
}

func (idx *Index) scanInto(qi [Dim]int16, clusters []int, h *knnHeap) {
	for _, c := range clusters {
		start := idx.offsets[c]
		sz := idx.sizes[c]
		base := idx.vecs[start*Dim:]
		for vi := range sz {
			d := distInt16(qi, base[vi*Dim:])
			h.push(d, start+vi)
		}
	}
}

func (idx *Index) SearchCount(query [Dim]float32, k int) int {
	var qi [Dim]int16
	for i, x := range query {
		v := x * scaleF
		if v > scaleF {
			v = scaleF
		} else if v < -scaleF {
			v = -scaleF
		}
		qi[i] = int16(v)
	}

	var topC [80]int
	idx.topCentroids(query, &topC)

	h := newKnnHeap()
	idx.scanInto(qi, topC[:20], &h)
	fc := h.fraudCount(idx.labs)

	if fc >= 1 && fc <= 4 {
		idx.scanInto(qi, topC[20:80], &h)
		fc = h.fraudCount(idx.labs)
	}
	return fc
}

func (idx *Index) topCentroids(query [Dim]float32, out *[80]int) {
	type entry struct {
		d float32
		c int
	}
	var best [80]entry
	n := len(out)
	filled := 0
	worstD := float32(1e38)
	worstP := 0

	for c := range NClusters {
		d := distArrayCent(query, &idx.centroids[c])
		if filled < n {
			best[filled] = entry{d, c}
			filled++
			if filled == n {
				worstP = 0
				for i := 1; i < n; i++ {
					if best[i].d > best[worstP].d {
						worstP = i
					}
				}
				worstD = best[worstP].d
			}
		} else if d < worstD {
			best[worstP] = entry{d, c}
			worstP = 0
			for i := 1; i < n; i++ {
				if best[i].d > best[worstP].d {
					worstP = i
				}
			}
			worstD = best[worstP].d
		}
	}

	for i := 1; i < filled; i++ {
		key := best[i]
		j := i - 1
		for j >= 0 && best[j].d > key.d {
			best[j+1] = best[j]
			j--
		}
		best[j+1] = key
	}
	for i := range filled {
		out[i] = best[i].c
	}
}


func distArrayCent(q [Dim]float32, c *[Dim]float32) float32 {
	d0 := q[0] - c[0]
	d1 := q[1] - c[1]
	d2 := q[2] - c[2]
	d3 := q[3] - c[3]
	d4 := q[4] - c[4]
	d5 := q[5] - c[5]
	d6 := q[6] - c[6]
	d7 := q[7] - c[7]
	d8 := q[8] - c[8]
	d9 := q[9] - c[9]
	d10 := q[10] - c[10]
	d11 := q[11] - c[11]
	d12 := q[12] - c[12]
	d13 := q[13] - c[13]
	return d0*d0 + d1*d1 + d2*d2 + d3*d3 +
		d4*d4 + d5*d5 + d6*d6 + d7*d7 +
		d8*d8 + d9*d9 + d10*d10 + d11*d11 +
		d12*d12 + d13*d13
}


func Build(vectors []float32, labels []uint8, nVectors int) (*Index, error) {
	if nVectors == 0 {
		return nil, fmt.Errorf("no vectors")
	}

	log.Printf("training k-means: %d clusters on all %d vectors (parallel)...", NClusters, nVectors)
	centroids := trainCentroids(vectors, nVectors)
	log.Println("k-means done")

	log.Printf("assigning %d vectors to clusters...", nVectors)
	assignments := make([]uint16, nVectors)
	assignParallel(vectors, nVectors, &centroids, assignments)

	var sizes [NClusters]uint32
	for _, c := range assignments {
		sizes[c]++
	}

	var offsets [NClusters]uint32
	var off uint32
	for c := range NClusters {
		offsets[c] = off
		off += sizes[c]
	}

	vecs := make([]int16, nVectors*Dim)
	labs := make([]uint8, nVectors)
	cursor := make([]uint32, NClusters)
	copy(cursor[:], offsets[:])

	for i := range nVectors {
		c := int(assignments[i])
		slot := cursor[c]
		src := vectors[i*Dim : (i+1)*Dim]
		dst := vecs[slot*Dim : (slot+1)*Dim]
		for d := range Dim {
			v := src[d] * scaleF
			if v > scaleF {
				v = scaleF
			} else if v < -scaleF {
				v = -scaleF
			}
			dst[d] = int16(v)
		}
		labs[slot] = labels[i]
		cursor[c]++
	}

	idx := &Index{
		centroids: centroids,
		sizes:     sizes,
		offsets:   offsets,
		vecs:      vecs,
		labs:      labs,
	}

	log.Println("ivf build done")
	return idx, nil
}
