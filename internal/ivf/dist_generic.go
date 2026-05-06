//go:build !amd64

package ivf

func distInt16(q [Dim]int16, v []int16) int32 {
	_ = v[13]
	d0 := int32(q[0]) - int32(v[0])
	d1 := int32(q[1]) - int32(v[1])
	d2 := int32(q[2]) - int32(v[2])
	d3 := int32(q[3]) - int32(v[3])
	d4 := int32(q[4]) - int32(v[4])
	d5 := int32(q[5]) - int32(v[5])
	d6 := int32(q[6]) - int32(v[6])
	d7 := int32(q[7]) - int32(v[7])
	d8 := int32(q[8]) - int32(v[8])
	d9 := int32(q[9]) - int32(v[9])
	d10 := int32(q[10]) - int32(v[10])
	d11 := int32(q[11]) - int32(v[11])
	d12 := int32(q[12]) - int32(v[12])
	d13 := int32(q[13]) - int32(v[13])
	return d0*d0 + d1*d1 + d2*d2 + d3*d3 +
		d4*d4 + d5*d5 + d6*d6 + d7*d7 +
		d8*d8 + d9*d9 + d10*d10 + d11*d11 +
		d12*d12 + d13*d13
}
