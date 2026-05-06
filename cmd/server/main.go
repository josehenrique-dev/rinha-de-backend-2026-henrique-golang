package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

"github.com/josehenrique-dev/rinha-2026/internal/handler"
	"github.com/josehenrique-dev/rinha-2026/internal/loader"
	"github.com/josehenrique-dev/rinha-2026/internal/service"
	"github.com/josehenrique-dev/rinha-2026/internal/vectordb"
	"github.com/josehenrique-dev/rinha-2026/internal/vectorize"
)

const dim = 14

func main() {
	vectorsPath := env("VECTORS_PATH", "/data/vectors.bin")
	labelsPath := env("LABELS_PATH", "/data/labels.bin")
	indexPath := env("INDEX_PATH", "/data/index.bin")
	mccRiskPath := env("MCC_RISK_PATH", "/data/mcc_risk.json")
	normPath := env("NORMALIZATION_PATH", "/data/normalization.json")
	port := env("PORT", "8080")

	ds, err := loader.Load(vectorsPath, labelsPath, dim)
	if err != nil {
		log.Fatalf("load dataset: %v", err)
	}
	defer ds.Close()

	log.Printf("dataset loaded: %d vectors", ds.Count)

	idx, err := vectordb.Load(indexPath, ds)
	if err != nil {
		log.Fatalf("load hnsw index: %v", err)
	}

	log.Println("hnsw index ready")

	mccRisk, err := loadMccRisk(mccRiskPath)
	if err != nil {
		log.Fatalf("load mcc_risk: %v", err)
	}

	norm, err := loadNormalization(normPath)
	if err != nil {
		log.Fatalf("load normalization: %v", err)
	}

	svc := service.New(idx, mccRisk, norm)
	h := handler.New(svc)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /ready", h.Ready)
	mux.HandleFunc("POST /fraud-score", h.FraudScore)

	log.Printf("listening on :%s", port)
	if err := http.ListenAndServe(fmt.Sprintf(":%s", port), mux); err != nil {
		log.Fatalf("server: %v", err)
	}
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func loadMccRisk(path string) (map[string]float32, error) {
	f, err := os.Open(path)
	if err != nil {
		return map[string]float32{}, nil
	}
	defer f.Close()
	var m map[string]float32
	if err := json.NewDecoder(f).Decode(&m); err != nil {
		return nil, err
	}
	return m, nil
}

func loadNormalization(path string) (vectorize.Normalization, error) {
	f, err := os.Open(path)
	if err != nil {
		return vectorize.Normalization{}, err
	}
	defer f.Close()
	var raw struct {
		MaxAmount            float32 `json:"max_amount"`
		MaxInstallments      float32 `json:"max_installments"`
		AmountVsAvgRatio     float32 `json:"amount_vs_avg_ratio"`
		MaxMinutes           float32 `json:"max_minutes"`
		MaxKm                float32 `json:"max_km"`
		MaxTxCount24h        float32 `json:"max_tx_count_24h"`
		MaxMerchantAvgAmount float32 `json:"max_merchant_avg_amount"`
	}
	if err := json.NewDecoder(f).Decode(&raw); err != nil {
		return vectorize.Normalization{}, err
	}
	return vectorize.Normalization{
		MaxAmount:            raw.MaxAmount,
		MaxInstallments:      raw.MaxInstallments,
		AmountVsAvgRatio:     raw.AmountVsAvgRatio,
		MaxMinutes:           raw.MaxMinutes,
		MaxKm:                raw.MaxKm,
		MaxTxCount24h:        raw.MaxTxCount24h,
		MaxMerchantAvgAmount: raw.MaxMerchantAvgAmount,
	}, nil
}
