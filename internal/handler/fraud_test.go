package handler_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/josehenrique-dev/rinha-2026/internal/handler"
	"github.com/josehenrique-dev/rinha-2026/internal/service"
	"github.com/josehenrique-dev/rinha-2026/internal/vectorize"
)

type fakeSearcher struct {
	count int
}

func (f *fakeSearcher) SearchCount(_ [14]float32, _ int) int {
	return f.count
}

func buildHandler(t *testing.T) *handler.Handler {
	t.Helper()
	norm := vectorize.Normalization{
		MaxAmount: 10000, MaxInstallments: 12, AmountVsAvgRatio: 10,
		MaxMinutes: 1440, MaxKm: 1000, MaxTxCount24h: 20, MaxMerchantAvgAmount: 10000,
	}
	svc := service.New(&fakeSearcher{count: 0}, map[string]float32{}, norm)
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
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.FraudScore < 0 || resp.FraudScore > 1 {
		t.Errorf("fraud_score out of range: %f", resp.FraudScore)
	}
}

func TestFraudScore_LargeBody(t *testing.T) {
	h := buildHandler(t)

	merchants := make([]string, 100)
	for i := range merchants {
		merchants[i] = "MERCHANT-KNOWN-STORE-NUMBER-" + fmt.Sprintf("%04d", i)
	}

	type txBody struct {
		ID          string `json:"id"`
		Transaction struct {
			Amount       float32 `json:"amount"`
			Installments int     `json:"installments"`
			RequestedAt  string  `json:"requested_at"`
		} `json:"transaction"`
		Customer struct {
			AvgAmount      float32  `json:"avg_amount"`
			TxCount24h     int      `json:"tx_count_24h"`
			KnownMerchants []string `json:"known_merchants"`
		} `json:"customer"`
		Merchant struct {
			ID        string  `json:"id"`
			MCC       string  `json:"mcc"`
			AvgAmount float32 `json:"avg_amount"`
		} `json:"merchant"`
		Terminal struct {
			IsOnline    bool    `json:"is_online"`
			CardPresent bool    `json:"card_present"`
			KmFromHome  float32 `json:"km_from_home"`
		} `json:"terminal"`
		LastTransaction any `json:"last_transaction"`
	}

	var payload txBody
	payload.ID = "tx-large"
	payload.Transaction.Amount = 100
	payload.Transaction.Installments = 1
	payload.Transaction.RequestedAt = "2026-03-11T18:45:53Z"
	payload.Customer.AvgAmount = 100
	payload.Customer.TxCount24h = 1
	payload.Customer.KnownMerchants = merchants
	payload.Merchant.ID = "MERC-AA"
	payload.Merchant.MCC = "5411"
	payload.Merchant.AvgAmount = 100
	payload.Terminal.IsOnline = false
	payload.Terminal.CardPresent = true
	payload.Terminal.KmFromHome = 5.0
	payload.LastTransaction = nil

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if len(bodyBytes) <= 2048 {
		t.Fatalf("test body must exceed 2048 bytes, got %d", len(bodyBytes))
	}

	req := httptest.NewRequest(http.MethodPost, "/fraud-score", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.FraudScore(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestFraudScoreBadJSON(t *testing.T) {
	h := buildHandler(t)
	req := httptest.NewRequest(http.MethodPost, "/fraud-score", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.FraudScore(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 (fallback response), got %d", w.Code)
	}
}
