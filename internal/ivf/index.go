package ivf

import (
	"fmt"
	"log"
	"math/rand"
	"syscall"
)

const (
	Dim       = 14
	NClusters = 512
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

	var topC [24]int
	idx.topCentroids(query, &topC)

	fc := idx.scan(qi, k, topC[:5])
	if fc == 2 || fc == 3 {
		fc = idx.scan(qi, k, topC[:24])
	}
	return fc
}

func (idx *Index) topCentroids(query [Dim]float32, out *[24]int) {
	type entry struct {
		d float32
		c int
	}
	var best [24]entry
	n := len(out)
	filled := 0
	worstD := float32(1e38)
	worstP := 0

	for c := 0; c < NClusters; c++ {
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
	for i := 0; i < filled; i++ {
		out[i] = best[i].c
	}
}

func (idx *Index) scan(qi [Dim]int16, k int, clusters []int) int {
	type cand struct {
		d int32
		i uint32
	}
	var buf [5]cand
	for i := range buf {
		buf[i].d = 1<<31 - 1
	}
	n, worstD, worstP := 0, int32(1<<31-1), 0

	for _, c := range clusters {
		start := idx.offsets[c]
		sz := idx.sizes[c]
		base := idx.vecs[start*Dim:]
		for vi := uint32(0); vi < sz; vi++ {
			d := distInt16(qi, base[vi*Dim:])
			if n < k {
				buf[n] = cand{d, start + vi}
				n++
				if n == k {
					worstP = 0
					for i := 1; i < k; i++ {
						if buf[i].d > buf[worstP].d {
							worstP = i
						}
					}
					worstD = buf[worstP].d
				}
			} else if d < worstD {
				buf[worstP] = cand{d, start + vi}
				worstP = 0
				for i := 1; i < k; i++ {
					if buf[i].d > buf[worstP].d {
						worstP = i
					}
				}
				worstD = buf[worstP].d
			}
		}
	}

	fc := 0
	for i := 0; i < n; i++ {
		if idx.labs[buf[i].i] == 1 {
			fc++
		}
	}
	return fc
}

func distArrayCent(q [Dim]float32, c *[Dim]float32) float32 {
	var s float32
	for i := 0; i < Dim; i++ {
		d := q[i] - c[i]
		s += d * d
	}
	return s
}

func distInt16(q [Dim]int16, v []int16) int32 {
	_ = v[Dim-1]
	var s int32
	for i := 0; i < Dim; i++ {
		d := int32(q[i]) - int32(v[i])
		s += d * d
	}
	return s
}

func Build(vectors []float32, labels []uint8, nVectors int) (*Index, error) {
	if nVectors == 0 {
		return nil, fmt.Errorf("no vectors")
	}

	rng := rand.New(rand.NewSource(42))

	log.Printf("training k-means: %d clusters on sample of %d vectors...", NClusters, min(nVectors, 200000))
	centroids := trainCentroids(vectors, nVectors, rng)
	log.Println("k-means done")

	log.Printf("assigning %d vectors to clusters...", nVectors)
	assignments := make([]uint16, nVectors)
	var sizes [NClusters]uint32

	for i := 0; i < nVectors; i++ {
		vec := vectors[i*Dim : (i+1)*Dim]
		best, bestD := 0, float32(1e38)
		for c := 0; c < NClusters; c++ {
			d := distSliceCent(vec, &centroids[c])
			if d < bestD {
				bestD = d
				best = c
			}
		}
		assignments[i] = uint16(best)
		sizes[best]++
	}

	var offsets [NClusters]uint32
	var off uint32
	for c := 0; c < NClusters; c++ {
		offsets[c] = off
		off += sizes[c]
	}

	vecs := make([]int16, nVectors*Dim)
	labs := make([]uint8, nVectors)
	cursor := make([]uint32, NClusters)
	copy(cursor[:], offsets[:])

	for i := 0; i < nVectors; i++ {
		c := int(assignments[i])
		slot := cursor[c]
		src := vectors[i*Dim : (i+1)*Dim]
		dst := vecs[slot*Dim : (slot+1)*Dim]
		for d := 0; d < Dim; d++ {
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
