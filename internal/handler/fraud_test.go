package handler_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	gojson "github.com/goccy/go-json"
	"github.com/josehenrique-dev/rinha-2026/internal/handler"
	"github.com/josehenrique-dev/rinha-2026/internal/loader"
	"github.com/josehenrique-dev/rinha-2026/internal/service"
	"github.com/josehenrique-dev/rinha-2026/internal/vectordb"
	"github.com/josehenrique-dev/rinha-2026/internal/vectorize"
)

func buildHandler(t *testing.T) *handler.Handler {
	t.Helper()
	const dim = 14
	const n = 20
	vectors := make([]float32, n*dim)
	labels := make([]uint8, n)
	ds := &loader.Dataset{Vectors: vectors, Labels: labels, Dim: dim, Count: n}
	idx, err := vectordb.Build(ds)
	if err != nil {
		t.Fatal(err)
	}
	norm := vectorize.Normalization{
		MaxAmount:            10000,
		MaxInstallments:      12,
		AmountVsAvgRatio:     10,
		MaxMinutes:           1440,
		MaxKm:                1000,
		MaxTxCount24h:        20,
		MaxMerchantAvgAmount: 10000,
	}
	svc := service.New(idx, map[string]float32{}, norm)
	return handler.New(svc)
}

func TestReady(t *testing.T) {
	h := buildHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	w := httptest.NewRecorder()
	h.Ready(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestFraudScore(t *testing.T) {
	h := buildHandler(t)
	body := `{
		"id":"tx-1",
		"transaction":{"amount":100,"installments":1,"requested_at":"2026-03-11T18:45:53Z"},
		"customer":{"avg_amount":100,"tx_count_24h":1,"known_merchants":["MERC-001"]},
		"merchant":{"id":"MERC-001","mcc":"5411","avg_amount":100},
		"terminal":{"is_online":false,"card_present":true,"km_from_home":5.0},
		"last_transaction":null
	}`
	req := httptest.NewRequest(http.MethodPost, "/fraud-score", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.FraudScore(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp struct {
		Approved   bool    `json:"approved"`
		FraudScore float32 `json:"fraud_score"`
	}
	if err := gojson.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.FraudScore < 0 || resp.FraudScore > 1 {
		t.Errorf("fraud_score out of range: %f", resp.FraudScore)
	}
}

func TestFraudScoreBadJSON(t *testing.T) {
	h := buildHandler(t)
	req := httptest.NewRequest(http.MethodPost, "/fraud-score", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.FraudScore(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}
