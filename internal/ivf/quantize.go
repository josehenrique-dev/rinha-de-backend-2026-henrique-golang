package ivf

const quantScale = int16(10000)
const maxInt64 = int64(^uint64(0) >> 1)

func quantizeVector(v [Dim]float32) [Dim]int16 {
	var out [Dim]int16
	for i, val := range v {
		if val <= -1 {
			out[i] = -quantScale
			continue
		}
		if val < 0 {
			val = 0
		}
		if val > 1 {
			val = 1
		}
		out[i] = int16(val*float32(quantScale) + 0.5)
	}
	return out
}

func quantizedDistance(q [Dim]int16, ref []int16, cutoff int64) int64 {
	var sum int64
	var d int64

	d = int64(q[5]) - int64(ref[5])
	if sum+d*d >= cutoff {
		return sum
	}
	sum += d * d

	d = int64(q[6]) - int64(ref[6])
	if sum+d*d >= cutoff {
		return sum
	}
	sum += d * d

	d = int64(q[2]) - int64(ref[2])
	if sum+d*d >= cutoff {
		return sum
	}
	sum += d * d

	d = int64(q[0]) - int64(ref[0])
	if sum+d*d >= cutoff {
		return sum
	}
	sum += d * d

	d = int64(q[7]) - int64(ref[7])
	if sum+d*d >= cutoff {
		return sum
	}
	sum += d * d

	d = int64(q[8]) - int64(ref[8])
	if sum+d*d >= cutoff {
		return sum
	}
	sum += d * d

	d = int64(q[12]) - int64(ref[12])
	if sum+d*d >= cutoff {
		return sum
	}
	sum += d * d

	d = int64(q[1]) - int64(ref[1])
	if sum+d*d >= cutoff {
		return sum
	}
	sum += d * d

	d = int64(q[3]) - int64(ref[3])
	if sum+d*d >= cutoff {
		return sum
	}
	sum += d * d

	d = int64(q[4]) - int64(ref[4])
	if sum+d*d >= cutoff {
		return sum
	}
	sum += d * d

	d = int64(q[9]) - int64(ref[9])
	if sum+d*d >= cutoff {
		return sum
	}
	sum += d * d

	d = int64(q[10]) - int64(ref[10])
	if sum+d*d >= cutoff {
		return sum
	}
	sum += d * d

	d = int64(q[11]) - int64(ref[11])
	if sum+d*d >= cutoff {
		return sum
	}
	sum += d * d

	d = int64(q[13]) - int64(ref[13])
	sum += d * d

	return sum
}
