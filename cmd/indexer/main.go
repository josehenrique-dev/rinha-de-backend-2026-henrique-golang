package main

import (
	"log"
	"os"
	"strconv"

	"github.com/josehenrique-dev/rinha-2026/internal/loader"
	"github.com/josehenrique-dev/rinha-2026/internal/vectordb"
)

const dim = 14

func main() {
	srcGz := env("SRC_GZ_PATH", "/data/references.json.gz")
	vectorsPath := env("VECTORS_PATH", "/data/vectors.bin")
	labelsPath := env("LABELS_PATH", "/data/labels.bin")
	indexPath := env("INDEX_PATH", "/data/index.bin")
	maxVectors := envInt("MAX_VECTORS", 1300000)

	if !loader.BinaryExists(vectorsPath, labelsPath) {
		log.Printf("preprocessing dataset (max %d vectors)...", maxVectors)
		if err := loader.Preprocess(srcGz, vectorsPath, labelsPath, maxVectors); err != nil {
			log.Fatalf("preprocess: %v", err)
		}
		log.Println("preprocessing done")
	}

	ds, err := loader.Load(vectorsPath, labelsPath, dim)
	if err != nil {
		log.Fatalf("load dataset: %v", err)
	}
	defer ds.Close()

	log.Printf("building hnsw index for %d vectors...", ds.Count)
	idx, err := vectordb.Build(ds)
	if err != nil {
		log.Fatalf("build: %v", err)
	}

	log.Printf("saving index to %s...", indexPath)
	if err := vectordb.Save(idx, indexPath); err != nil {
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

func envInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		n, err := strconv.Atoi(v)
		if err == nil {
			return n
		}
	}
	return fallback
}
