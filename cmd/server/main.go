package main

import (
	"encoding/json"
	"log"
	"math"
	"os"
	"runtime"
	"runtime/debug"
	"strconv"
	"time"

	"github.com/josehenrique-dev/rinha-2026/internal/fdpass"
	"github.com/josehenrique-dev/rinha-2026/internal/handler"
	"github.com/josehenrique-dev/rinha-2026/internal/ivf"
	"github.com/josehenrique-dev/rinha-2026/internal/server"
	"github.com/josehenrique-dev/rinha-2026/internal/service"
	"github.com/josehenrique-dev/rinha-2026/internal/vectorize"
)

func main() {
	runtime.GOMAXPROCS(1)

	indexPath := env("INDEX_PATH", "/data/ivf.bin")
	mccRiskPath := env("MCC_RISK_PATH", "/data/mcc_risk.json")
	normPath := env("NORMALIZATION_PATH", "/data/normalization.json")
	socketPath := env("SOCKET_PATH", "")
	port := env("PORT", "8080")

	log.Printf("loading ivf index from %s...", indexPath)
	idx, err := ivf.Load(indexPath)
	if err != nil {
		log.Fatalf("load ivf index: %v", err)
	}
	defer idx.Close()
	log.Println("ivf index ready")

	d := ivf.Warmup(idx, 500)
	log.Printf("warmup: 500 iters in %s", d)

	runtime.GC()
	debug.SetGCPercent(-1)
	debug.SetMemoryLimit(math.MaxInt64)

	mccRisk, err := loadMccRisk(mccRiskPath)
	if err != nil {
		log.Fatalf("load mcc_risk: %v", err)
	}

	norm, err := loadNormalization(normPath)
	if err != nil {
		log.Fatalf("load normalization: %v", err)
	}

	svc := service.New(idx, mccRisk, norm)

	shedSlots := envInt("SHED_SLOTS", 4)
	shedTimeoutMS := envInt("SHED_TIMEOUT_MS", 3)
	shedSem := make(chan struct{}, shedSlots)

	h := func(path []byte, body []byte) []byte {
		if len(path) >= 6 && path[1] == 'r' {
			return server.Responses[6]
		}

		select {
		case shedSem <- struct{}{}:
			defer func() { <-shedSem }()
		case <-time.After(time.Duration(shedTimeoutMS) * time.Millisecond):
			return server.Responses[0]
		}

		var p vectorize.Payload
		if err := handler.ParsePayload(body, &p); err != nil {
			return server.Responses[0]
		}
		fraudCount := svc.FraudCount(p)
		return server.Responses[fraudCount]
	}

	var srv *server.Server
	if socketPath != "" {
		srv, err = server.Listen(socketPath, h)
		if err != nil {
			log.Fatalf("listen unix %s: %v", socketPath, err)
		}
		log.Printf("listening on unix:%s", socketPath)
	} else {
		srv, err = server.ListenTCP(":"+port, h)
		if err != nil {
			log.Fatalf("listen tcp :%s: %v", port, err)
		}
		log.Printf("listening on :%s", port)
	}

	if ctrlPath := os.Getenv("API_CTRL_SOCKET"); ctrlPath != "" {
		fdCh, _, err := fdpass.Listen(ctrlPath)
		if err != nil {
			log.Printf("WARN: fdpass listen on %q: %v", ctrlPath, err)
		} else {
			log.Printf("fdpass listening on %s", ctrlPath)
			go srv.ServeFDChannel(fdCh)
		}
	}

	select {}
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envInt(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 1 {
		return def
	}
	return n
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
