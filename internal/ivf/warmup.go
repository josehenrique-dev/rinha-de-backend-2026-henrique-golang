package ivf

import "time"

func Warmup(idx *Index, iters int) time.Duration {
	if idx == nil || iters <= 0 {
		return 0
	}
	t0 := time.Now()
	rng := newPCG(0xC0FFEE, 0xCAFE)
	var query [Dim]float32
	for range iters {
		for d := range Dim {
			v := int(rng.intn(20000)) - 10000
			query[d] = float32(v)
		}
		idx.SearchCount(query, 5)
	}
	return time.Since(t0)
}
