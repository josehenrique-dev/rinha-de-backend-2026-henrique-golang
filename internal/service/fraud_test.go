package service_test

import (
	"testing"

	"github.com/josehenrique-dev/rinha-2026/internal/loader"
	"github.com/josehenrique-dev/rinha-2026/internal/service"
	"github.com/josehenrique-dev/rinha-2026/internal/vectordb"
	"github.com/josehenrique-dev/rinha-2026/internal/vectorize"
)

func buildTestService(t *testing.T) *service.FraudService {
	t.Helper()
	const dim = 14
	const n = 20
	vectors := make([]float32, n*dim)
	labels := make([]uint8, n)
	for i := 0; i < n; i++ {
		labels[i] = uint8(i % 2)
	}
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
	return service.New(idx, map[string]float32{}, norm)
}

func TestScoreReturnsValidRange(t *testing.T) {
	svc := buildTestService(t)
	p := vectorize.Payload{
		Transaction: vectorize.Transaction{Amount: 100, Installments: 1},
		Customer:    vectorize.Customer{AvgAmount: 100, TxCount24h: 1},
		Merchant:    vectorize.Merchant{ID: "M1", MCC: "5411", AvgAmount: 100},
		Terminal:    vectorize.Terminal{CardPresent: true},
	}
	approved, score := svc.Score(p)
	if score < 0 || score > 1 {
		t.Errorf("score out of range: %f", score)
	}
	if approved != (score < 0.6) {
		t.Errorf("approved mismatch: approved=%v score=%f", approved, score)
	}
}

func TestScoreApprovedWhenLowFraud(t *testing.T) {
	const dim = 14
	const n = 20
	vectors := make([]float32, n*dim)
	labels := make([]uint8, n)
	ds := &loader.Dataset{Vectors: vectors, Labels: labels, Dim: dim, Count: n}
	idx, err := vectordb.Build(ds)
	if err != nil {
		t.Fatal(nil)
	}
	norm := vectorize.Normalization{
		MaxAmount: 10000, MaxInstallments: 12, AmountVsAvgRatio: 10,
		MaxMinutes: 1440, MaxKm: 1000, MaxTxCount24h: 20, MaxMerchantAvgAmount: 10000,
	}
	svc := service.New(idx, map[string]float32{}, norm)
	p := vectorize.Payload{
		Transaction: vectorize.Transaction{Amount: 100, Installments: 1},
		Customer:    vectorize.Customer{AvgAmount: 100, TxCount24h: 1},
		Merchant:    vectorize.Merchant{ID: "M1", MCC: "5411", AvgAmount: 100},
		Terminal:    vectorize.Terminal{CardPresent: true},
	}
	approved, score := svc.Score(p)
	if score != 0.0 {
		t.Errorf("expected score 0.0 (all legit), got %f", score)
	}
	if !approved {
		t.Errorf("expected approved=true for score 0.0")
	}
}
