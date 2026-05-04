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

func Preprocess(srcGzPath, vectorsDst, labelsDst string) error {
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
	for _, e := range entries {
		for _, v := range e.Vector {
			binary.LittleEndian.PutUint32(buf, math.Float32bits(v))
			if _, err := vf.Write(buf); err != nil {
				return err
			}
		}
		label := uint8(0)
		if e.Label == "fraud" {
			label = 1
		}
		if _, err := lf.Write([]byte{label}); err != nil {
			return err
		}
	}
	return nil
}

func BinaryExists(vectorsPath, labelsPath string) bool {
	_, ve := os.Stat(vectorsPath)
	_, le := os.Stat(labelsPath)
	return ve == nil && le == nil
}
