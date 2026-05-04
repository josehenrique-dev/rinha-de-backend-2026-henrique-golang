package loader

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

type Dataset struct {
	Vectors    []float32
	Labels     []uint8
	Dim        int
	Count      int
	vectorsMem []byte
	labelsMem  []byte
}

func Load(vectorsPath, labelsPath string, dim int) (*Dataset, error) {
	vf, err := os.Open(vectorsPath)
	if err != nil {
		return nil, fmt.Errorf("open vectors: %w", err)
	}
	defer vf.Close()

	vStat, err := vf.Stat()
	if err != nil {
		return nil, err
	}
	vSize := int(vStat.Size())

	vMem, err := syscall.Mmap(int(vf.Fd()), 0, vSize, syscall.PROT_READ, syscall.MAP_SHARED)
	if err != nil {
		return nil, fmt.Errorf("mmap vectors: %w", err)
	}

	lf, err := os.Open(labelsPath)
	if err != nil {
		syscall.Munmap(vMem)
		return nil, fmt.Errorf("open labels: %w", err)
	}
	defer lf.Close()

	lStat, err := lf.Stat()
	if err != nil {
		syscall.Munmap(vMem)
		return nil, err
	}
	lSize := int(lStat.Size())

	lMem, err := syscall.Mmap(int(lf.Fd()), 0, lSize, syscall.PROT_READ, syscall.MAP_SHARED)
	if err != nil {
		syscall.Munmap(vMem)
		return nil, fmt.Errorf("mmap labels: %w", err)
	}

	count := lSize
	vectors := unsafe.Slice((*float32)(unsafe.Pointer(&vMem[0])), count*dim)

	return &Dataset{
		Vectors:    vectors,
		Labels:     lMem,
		Dim:        dim,
		Count:      count,
		vectorsMem: vMem,
		labelsMem:  lMem,
	}, nil
}

func (d *Dataset) Close() error {
	if err := syscall.Munmap(d.vectorsMem); err != nil {
		return err
	}
	return syscall.Munmap(d.labelsMem)
}
