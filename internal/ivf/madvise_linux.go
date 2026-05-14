//go:build linux

package ivf

import "golang.org/x/sys/unix"

func postMmap(mem []byte) {
	unix.Madvise(mem, unix.MADV_RANDOM)
	unix.Madvise(mem, unix.MADV_POPULATE_READ)
	unix.Madvise(mem, unix.MADV_HUGEPAGE)
}
