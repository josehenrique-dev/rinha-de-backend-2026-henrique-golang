package ivf

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"syscall"
	"unsafe"
)

var indexMagic = [8]byte{'I', 'V', 'F', '2', '0', '2', '6', 'A'}

const indexVersion = uint32(1)

type indexHeader struct {
	Magic      [8]byte
	Version    uint32
	Dimensions uint32
	Scale      int32
	Count      uint32
}

type indexIVFHeader struct {
	Clusters        uint32
	NProbe          uint32
	AmbiguousNProbe uint32
	Flags           uint32
}

func Save(idx *Index, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create: %w", err)
	}
	defer f.Close()

	count := uint32(len(idx.labels))
	clusters := uint32(idx.ivf.clusters)
	flags := uint32(0)
	if idx.ivf.repair {
		flags = 1
	}

	hdr := indexHeader{Magic: indexMagic, Version: indexVersion, Dimensions: Dim, Scale: int32(quantScale), Count: count}
	if err := binary.Write(f, binary.LittleEndian, hdr); err != nil {
		return err
	}

	ivfHdr := indexIVFHeader{Clusters: clusters, NProbe: uint32(idx.ivf.nprobe), AmbiguousNProbe: uint32(idx.ivf.ambiguousNprobe), Flags: flags}
	if err := binary.Write(f, binary.LittleEndian, ivfHdr); err != nil {
		return err
	}

	if err := binary.Write(f, binary.LittleEndian, idx.ivf.centroids); err != nil {
		return err
	}
	centBlocks := idx.ivf.centroidBlocks
	if len(centBlocks) == 0 {
		centBlocks = buildCentroidBlocks(idx.ivf.centroids, idx.ivf.clusters)
	}
	if err := binary.Write(f, binary.LittleEndian, centBlocks); err != nil {
		return err
	}
	if err := binary.Write(f, binary.LittleEndian, idx.ivf.listOffsets); err != nil {
		return err
	}
	if err := binary.Write(f, binary.LittleEndian, idx.ivf.blockOffsets); err != nil {
		return err
	}
	if err := binary.Write(f, binary.LittleEndian, idx.ivf.bboxMin); err != nil {
		return err
	}
	if err := binary.Write(f, binary.LittleEndian, idx.ivf.bboxMax); err != nil {
		return err
	}
	if err := binary.Write(f, binary.LittleEndian, idx.ivf.origIDs); err != nil {
		return err
	}
	if _, err := f.Write(idx.labels); err != nil {
		return err
	}
	if len(idx.labels)%2 != 0 {
		if _, err := f.Write([]byte{0}); err != nil {
			return err
		}
	}
	if err := binary.Write(f, binary.LittleEndian, idx.blocks); err != nil {
		return err
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

	mem, err := syscall.Mmap(int(f.Fd()), 0, int(stat.Size()), syscall.PROT_READ, syscall.MAP_SHARED)
	if err != nil {
		return loadHeap(f)
	}

	for i := 0; i < len(mem); i += 4096 {
		_ = mem[i]
	}

	idx, err := parseIndexBytes(mem)
	if err != nil {
		syscall.Munmap(mem)
		return nil, err
	}
	idx.mem = mem
	return idx, nil
}

func loadHeap(f *os.File) (*Index, error) {
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	data, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}
	return parseIndexBytes(data)
}

func parseIndexBytes(data []byte) (*Index, error) {
	off := 0
	hdrSize := binary.Size(indexHeader{})
	ivfHdrSize := binary.Size(indexIVFHeader{})

	if len(data) < hdrSize+ivfHdrSize {
		return nil, fmt.Errorf("index too small")
	}

	var hdr indexHeader
	if err := binary.Read(asReader(data[off:]), binary.LittleEndian, &hdr); err != nil {
		return nil, err
	}
	if hdr.Magic != indexMagic {
		return nil, fmt.Errorf("invalid magic")
	}
	if hdr.Dimensions != Dim || hdr.Scale != int32(quantScale) {
		return nil, fmt.Errorf("incompatible index: dim=%d scale=%d", hdr.Dimensions, hdr.Scale)
	}
	off += hdrSize

	var ivfHdr indexIVFHeader
	if err := binary.Read(asReader(data[off:]), binary.LittleEndian, &ivfHdr); err != nil {
		return nil, err
	}
	off += ivfHdrSize

	clusters := int(ivfHdr.Clusters)
	count := int(hdr.Count)

	centroidLen := clusters * Dim
	centBlkLen := blocksForRows(clusters) * blockStride
	offsetsLen := clusters + 1
	bboxLen := clusters * Dim
	origIDsLen := count
	labelsLen := count
	labelsPad := 0
	if count%2 != 0 {
		labelsPad = 1
	}

	centroids := readInt16s(data, &off, centroidLen)
	centBlocks := readInt16s(data, &off, centBlkLen)
	listOffsets := readUint32s(data, &off, offsetsLen)
	blockOffsets := readUint32s(data, &off, offsetsLen)
	bboxMin := readInt16s(data, &off, bboxLen)
	bboxMax := readInt16s(data, &off, bboxLen)
	origIDs := readUint32s(data, &off, origIDsLen)

	labels := data[off : off+labelsLen]
	off += labelsLen + labelsPad

	blockCount := int(blockOffsets[len(blockOffsets)-1])
	blocks := readInt16s(data, &off, blockCount*blockStride)

	return &Index{
		blocks: blocks,
		labels: labels,
		ivf: ivfMeta{
			clusters:        clusters,
			nprobe:          int(ivfHdr.NProbe),
			ambiguousNprobe: int(ivfHdr.AmbiguousNProbe),
			repair:          ivfHdr.Flags&1 == 1,
			centroids:       centroids,
			centroidBlocks:  centBlocks,
			listOffsets:     listOffsets,
			blockOffsets:    blockOffsets,
			bboxMin:         bboxMin,
			bboxMax:         bboxMax,
			origIDs:         origIDs,
		},
	}, nil
}

func readInt16s(data []byte, off *int, n int) []int16 {
	size := n * 2
	s := unsafe.Slice((*int16)(unsafe.Pointer(&data[*off])), n)
	*off += size
	return s
}

func readUint32s(data []byte, off *int, n int) []uint32 {
	size := n * 4
	s := unsafe.Slice((*uint32)(unsafe.Pointer(&data[*off])), n)
	*off += size
	return s
}

type byteReader struct {
	data []byte
	pos  int
}

func asReader(data []byte) *byteReader { return &byteReader{data: data} }

func (r *byteReader) Read(p []byte) (int, error) {
	n := copy(p, r.data[r.pos:])
	r.pos += n
	if n < len(p) {
		return n, io.EOF
	}
	return n, nil
}
