package loader

import (
	"compress/gzip"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"os"
)

type refEntry struct {
	Vector []float32 `json:"vector"`
	Label  string    `json:"label"`
}

// fraudRatio is the fraction of the budget allocated to fraud vectors.
// Increase fraud coverage to reduce FN (weight=3) at modest cost in FP (weight=1).
const fraudRatio = 0.55

func Preprocess(srcGzPath, vectorsDst, labelsDst string, maxVectors int) error {
	f, err := os.Open(srcGzPath)
	if err != nil {
		return fmt.Errorf("open gz: %w", err)
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("gzip reader: %w", err)
	}
	defer gr.Close()

	var entries []refEntry
	if err := json.NewDecoder(gr).Decode(&entries); err != nil {
		return fmt.Errorf("decode json: %w", err)
	}

	var fraudEntries, legitEntries []refEntry
	for _, e := range entries {
		if e.Label == "fraud" {
			fraudEntries = append(fraudEntries, e)
		} else {
			legitEntries = append(legitEntries, e)
		}
	}

	fraudBudget := int(float64(maxVectors) * fraudRatio)
	if fraudBudget > len(fraudEntries) {
		fraudBudget = len(fraudEntries)
	}
	legitBudget := maxVectors - fraudBudget
	if legitBudget > len(legitEntries) {
		legitBudget = len(legitEntries)
	}

	fraudStep := 1
	if len(fraudEntries) > fraudBudget {
		fraudStep = len(fraudEntries) / fraudBudget
	}
	legitStep := 1
	if len(legitEntries) > legitBudget {
		legitStep = len(legitEntries) / legitBudget
	}

	vf, err := os.Create(vectorsDst)
	if err != nil {
		return err
	}
	defer vf.Close()

	lf, err := os.Create(labelsDst)
	if err != nil {
		return err
	}
	defer lf.Close()

	buf := make([]byte, 4)
	writeEntry := func(e refEntry, label uint8) error {
		for _, v := range e.Vector {
			binary.LittleEndian.PutUint32(buf, math.Float32bits(v))
			if _, err := vf.Write(buf); err != nil {
				return err
			}
		}
		_, err := lf.Write([]byte{label})
		return err
	}

	// Interleave fraud and legit for balanced HNSW graph connectivity.
	fi, li := 0, 0
	fraudWritten, legitWritten := 0, 0
	for fi < len(fraudEntries) || li < len(legitEntries) {
		if fi < len(fraudEntries) && (fraudStep == 1 || fi%fraudStep == 0) && fraudWritten < fraudBudget {
			if err := writeEntry(fraudEntries[fi], 1); err != nil {
				return err
			}
			fraudWritten++
		}
		fi++

		if li < len(legitEntries) && (legitStep == 1 || li%legitStep == 0) && legitWritten < legitBudget {
			if err := writeEntry(legitEntries[li], 0); err != nil {
				return err
			}
			legitWritten++
		}
		li++

		if fraudWritten >= fraudBudget && legitWritten >= legitBudget {
			break
		}
	}
	return nil
}

func BinaryExists(vectorsPath, labelsPath string) bool {
	_, ve := os.Stat(vectorsPath)
	_, le := os.Stat(labelsPath)
	return ve == nil && le == nil
}

// ReadAll decodes the gzipped JSON and returns all vectors (flat float32) and labels.
func ReadAll(srcGzPath string) (vectors []float32, labels []uint8, err error) {
	f, err := os.Open(srcGzPath)
	if err != nil {
		return nil, nil, fmt.Errorf("open gz: %w", err)
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		return nil, nil, fmt.Errorf("gzip reader: %w", err)
	}
	defer gr.Close()

	var entries []refEntry
	if err := json.NewDecoder(gr).Decode(&entries); err != nil {
		return nil, nil, fmt.Errorf("decode json: %w", err)
	}

	n := len(entries)
	vectors = make([]float32, n*14)
	labels = make([]uint8, n)
	for i, e := range entries {
		copy(vectors[i*14:], e.Vector)
		if e.Label == "fraud" {
			labels[i] = 1
		}
	}
	return vectors, labels, nil
}
