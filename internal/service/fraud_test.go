package service_test

import (
	"testing"

	"github.com/josehenrique-dev/rinha-2026/internal/service"
	"github.com/josehenrique-dev/rinha-2026/internal/vectorize"
)

type fakeSearcher struct {
	count int
}

func (f *fakeSearcher) SearchCount(_ [14]float32, _ int) int {
	return f.count
}

func TestScoreReturnsValidRange(t *testing.T) {
	svc := service.New(&fakeSearcher{count: 2}, map[string]float32{}, vectorize.Normalization{
		MaxAmount: 10000, MaxInstallments: 12, AmountVsAvgRatio: 10,
		MaxMinutes: 1440, MaxKm: 1000, MaxTxCount24h: 20, MaxMerchantAvgAmount: 10000,
	})
	p := vectorize.Payload{
		Transaction: vectorize.Transaction{Amount: 100, Installments: 1, RequestedAt: "2026-03-11T18:45:53Z"},
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
	svc := service.New(&fakeSearcher{count: 0}, map[string]float32{}, vectorize.Normalization{
		MaxAmount: 10000, MaxInstallments: 12, AmountVsAvgRatio: 10,
		MaxMinutes: 1440, MaxKm: 1000, MaxTxCount24h: 20, MaxMerchantAvgAmount: 10000,
	})
	p := vectorize.Payload{
		Transaction: vectorize.Transaction{Amount: 100, Installments: 1, RequestedAt: "2026-03-11T18:45:53Z"},
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
