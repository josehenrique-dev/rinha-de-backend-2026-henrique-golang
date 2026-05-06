//go:build amd64

package ivf

//go:noescape
func distInt16(q [Dim]int16, v []int16) int32
