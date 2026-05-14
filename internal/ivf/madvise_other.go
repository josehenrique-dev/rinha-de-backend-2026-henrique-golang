//go:build !linux

package ivf

func postMmap(mem []byte) {
	for i := 0; i < len(mem); i += 4096 {
		_ = mem[i]
	}
}
