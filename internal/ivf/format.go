package ivf

import (
	"encoding/binary"
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

var magic = [4]byte{'I', 'V', 'F', '1'}

func Save(idx *Index, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create: %w", err)
	}
	defer f.Close()

	nVectors := uint64(len(idx.labs))

	hdr := make([]byte, 20)
	copy(hdr[0:4], magic[:])
	binary.LittleEndian.PutUint32(hdr[4:8], NClusters)
	binary.LittleEndian.PutUint32(hdr[8:12], Dim)
	binary.LittleEndian.PutUint64(hdr[12:20], nVectors)
	if _, err := f.Write(hdr); err != nil {
		return err
	}

	centBytes := unsafe.Slice((*byte)(unsafe.Pointer(&idx.centroids[0][0])), NClusters*Dim*4)
	if _, err := f.Write(centBytes); err != nil {
		return err
	}

	sizeBytes := unsafe.Slice((*byte)(unsafe.Pointer(&idx.sizes[0])), NClusters*4)
	if _, err := f.Write(sizeBytes); err != nil {
		return err
	}

	if nVectors > 0 {
		vecBytes := unsafe.Slice((*byte)(unsafe.Pointer(&idx.vecs[0])), int(nVectors)*Dim*2)
		if _, err := f.Write(vecBytes); err != nil {
			return err
		}
		if _, err := f.Write(idx.labs); err != nil {
			return err
		}
	}

	return nil
}

func Load(path string) (*Index, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open: %w", err)
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return nil, err
	}
	fileSize := int(stat.Size())

	mem, err := syscall.Mmap(int(f.Fd()), 0, fileSize, syscall.PROT_READ, syscall.MAP_SHARED)
	if err != nil {
		return nil, fmt.Errorf("mmap: %w", err)
	}

	for i := 0; i < len(mem); i += 4096 {
		_ = mem[i]
	}

	var mag [4]byte
	copy(mag[:], mem[0:4])
	if mag != magic {
		syscall.Munmap(mem)
		return nil, fmt.Errorf("invalid magic")
	}
	nc := int(binary.LittleEndian.Uint32(mem[4:8]))
	d := int(binary.LittleEndian.Uint32(mem[8:12]))
	nVectors := int(binary.LittleEndian.Uint64(mem[12:20]))

	if nc != NClusters || d != Dim {
		syscall.Munmap(mem)
		return nil, fmt.Errorf("incompatible index: nClusters=%d dim=%d", nc, d)
	}

	var idx Index
	idx.mem = mem

	off := 20

	centRaw := unsafe.Slice((*float32)(unsafe.Pointer(&mem[off])), NClusters*Dim)
	for c := 0; c < NClusters; c++ {
		copy(idx.centroids[c][:], centRaw[c*Dim:(c+1)*Dim])
	}
	off += NClusters * Dim * 4

	for c := 0; c < NClusters; c++ {
		idx.sizes[c] = binary.LittleEndian.Uint32(mem[off+c*4:])
	}
	off += NClusters * 4

	var cursor uint32
	for c := 0; c < NClusters; c++ {
		idx.offsets[c] = cursor
		cursor += idx.sizes[c]
	}

	if nVectors > 0 {
		idx.vecs = unsafe.Slice((*int16)(unsafe.Pointer(&mem[off])), nVectors*Dim)
		off += nVectors * Dim * 2
		idx.labs = mem[off : off+nVectors]
	}

	return &idx, nil
}
