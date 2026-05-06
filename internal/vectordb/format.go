package vectordb

import (
	"encoding/binary"
	"fmt"
	"os"
	"sync"
	"syscall"
	"unsafe"
)

var magic = [8]byte{'H', 'N', 'S', 'W', '2', '0', '2', '6'}

const headerSize = 8 + 4 + 4 + 4 + 4 + 4 + 4 + 8

func writeHeader(f *os.File, nodeCount, M, M0, maxLayer int, entryPoint uint32, upperLayerSize uint64) error {
	buf := make([]byte, headerSize)
	copy(buf[0:8], magic[:])
	binary.LittleEndian.PutUint32(buf[8:], uint32(nodeCount))
	binary.LittleEndian.PutUint32(buf[12:], uint32(M))
	binary.LittleEndian.PutUint32(buf[16:], uint32(M0))
	binary.LittleEndian.PutUint32(buf[20:], uint32(maxLayer))
	binary.LittleEndian.PutUint32(buf[24:], entryPoint)
	binary.LittleEndian.PutUint32(buf[28:], 0)
	binary.LittleEndian.PutUint64(buf[32:], upperLayerSize)
	_, err := f.Write(buf)
	return err
}

func saveGraph(g *graph, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create: %w", err)
	}
	defer f.Close()

	upperData := serializeUpperLayers(g)

	if err := writeHeader(f, g.nodeCount, g.M, g.M0, g.maxLayer, g.entryPoint, uint64(len(upperData))); err != nil {
		return fmt.Errorf("write header: %w", err)
	}

	if len(g.layer0) > 0 {
		layer0Bytes := unsafe.Slice((*byte)(unsafe.Pointer(&g.layer0[0])), len(g.layer0)*4)
		if _, err := f.Write(layer0Bytes); err != nil {
			return fmt.Errorf("write layer0: %w", err)
		}
	}

	if len(upperData) > 0 {
		if _, err := f.Write(upperData); err != nil {
			return fmt.Errorf("write upper layers: %w", err)
		}
	}

	return nil
}

func serializeUpperLayers(g *graph) []byte {
	if len(g.upperNodes) == 0 {
		return nil
	}
	var buf []byte
	buf = binary.LittleEndian.AppendUint32(buf, uint32(len(g.upperNodes)))
	for _, un := range g.upperNodes {
		buf = binary.LittleEndian.AppendUint32(buf, un.nodeID)
		buf = append(buf, un.maxLayer)
		for _, layerNeighbors := range un.neighbors {
			for _, n := range layerNeighbors {
				buf = binary.LittleEndian.AppendUint32(buf, n)
			}
		}
	}
	return buf
}

func loadGraph(path string, vectors []float32, labels []uint8, dim int) (*graph, []byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, fmt.Errorf("open: %w", err)
	}
	defer f.Close()

	hdrBuf := make([]byte, headerSize)
	if _, err := f.Read(hdrBuf); err != nil {
		return nil, nil, fmt.Errorf("read header: %w", err)
	}

	var mag [8]byte
	copy(mag[:], hdrBuf[0:8])
	if mag != magic {
		return nil, nil, fmt.Errorf("invalid magic: %v", mag)
	}

	nodeCount := int(binary.LittleEndian.Uint32(hdrBuf[8:]))
	M := int(binary.LittleEndian.Uint32(hdrBuf[12:]))
	M0 := int(binary.LittleEndian.Uint32(hdrBuf[16:]))
	maxLayer := int(binary.LittleEndian.Uint32(hdrBuf[20:]))
	entryPoint := binary.LittleEndian.Uint32(hdrBuf[24:])

	stat, err := f.Stat()
	if err != nil {
		return nil, nil, err
	}
	fileSize := int(stat.Size())

	mem, err := syscall.Mmap(int(f.Fd()), 0, fileSize, syscall.PROT_READ, syscall.MAP_SHARED)
	if err != nil {
		return nil, nil, fmt.Errorf("mmap: %w", err)
	}

	layer0Size := nodeCount * M0 * 4

	layer0Bytes := mem[headerSize : headerSize+layer0Size]
	layer0 := unsafe.Slice((*uint32)(unsafe.Pointer(&layer0Bytes[0])), nodeCount*M0)

	upperData := mem[headerSize+layer0Size:]
	upperNodes, upperIndex := deserializeUpperLayers(upperData, M)

	g := &graph{
		layer0:     layer0,
		upperNodes: upperNodes,
		upperIndex: upperIndex,
		vectors:    vectors,
		labels:     labels,
		M:          M,
		M0:         M0,
		nodeCount:  nodeCount,
		dim:        dim,
		entryPoint: entryPoint,
		maxLayer:   maxLayer,
	}
	g.pool = sync.Pool{
		New: func() any { return newVisitedTracker(nodeCount) },
	}
	g.candPool = sync.Pool{
		New: func() any { return newCandidateHeap(defaultEfSearch * 2) },
	}
	g.resPool = sync.Pool{
		New: func() any { return newResultHeap(defaultEfSearch) },
	}

	return g, mem, nil
}

func deserializeUpperLayers(data []byte, M int) ([]upperNode, map[uint32]int) {
	if len(data) < 4 {
		return nil, make(map[uint32]int)
	}
	count := int(binary.LittleEndian.Uint32(data[:4]))
	offset := 4
	nodes := make([]upperNode, 0, count)
	index := make(map[uint32]int, count)
	for i := 0; i < count; i++ {
		nodeID := binary.LittleEndian.Uint32(data[offset:])
		offset += 4
		maxLayerVal := int(data[offset])
		offset++
		neighbors := make([][]uint32, maxLayerVal)
		for l := 0; l < maxLayerVal; l++ {
			neighbors[l] = make([]uint32, M)
			for j := 0; j < M; j++ {
				neighbors[l][j] = binary.LittleEndian.Uint32(data[offset:])
				offset += 4
			}
		}
		index[nodeID] = len(nodes)
		nodes = append(nodes, upperNode{
			nodeID:    nodeID,
			maxLayer:  uint8(maxLayerVal),
			neighbors: neighbors,
		})
	}
	return nodes, index
}

func freeMmap(mem []byte) {
	if mem != nil {
		syscall.Munmap(mem)
	}
}
