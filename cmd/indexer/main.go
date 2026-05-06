package main

import (
	"log"
	"os"

	"github.com/josehenrique-dev/rinha-2026/internal/ivf"
	"github.com/josehenrique-dev/rinha-2026/internal/loader"
)

func main() {
	srcGz := env("SRC_GZ_PATH", "/data/references.json.gz")
	indexPath := env("INDEX_PATH", "/data/ivf.bin")

	log.Printf("loading all vectors from %s...", srcGz)
	vectors, labels, err := loader.ReadAll(srcGz)
	if err != nil {
		log.Fatalf("read all: %v", err)
	}
	nVectors := len(labels)
	log.Printf("loaded %d vectors", nVectors)

	idx, err := ivf.Build(vectors, labels, nVectors)
	if err != nil {
		log.Fatalf("build: %v", err)
	}

	log.Printf("saving index to %s...", indexPath)
	if err := ivf.Save(idx, indexPath); err != nil {
		log.Fatalf("save: %v", err)
	}
	log.Println("index saved successfully")
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
