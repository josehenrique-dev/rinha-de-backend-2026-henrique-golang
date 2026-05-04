package vectorize_test

import (
	"testing"
	"time"

	"github.com/josehenrique-dev/rinha-2026/internal/vectorize"
)

func TestVectorizeKnownLegit(t *testing.T) {
	norm := vectorize.Normalization{
		MaxAmount:            10000,
		MaxInstallments:      12,
		AmountVsAvgRatio:     10,
		MaxMinutes:           1440,
		MaxKm:                1000,
		MaxTxCount24h:        20,
		MaxMerchantAvgAmount: 10000,
	}
	mccRisk := map[string]float32{"5411": 0.15}

	requestedAt, _ := time.Parse(time.RFC3339, "2026-03-11T18:45:53Z")
	p := vectorize.Payload{
		ID: "tx-1329056812",
		Transaction: vectorize.Transaction{
			Amount:       41.12,
			Installments: 2,
			RequestedAt:  requestedAt,
		},
		Customer: vectorize.Customer{
			AvgAmount:      82.24,
			TxCount24h:     3,
			KnownMerchants: []string{"MERC-003", "MERC-016"},
		},
		Merchant: vectorize.Merchant{
			ID:        "MERC-016",
			MCC:       "5411",
			AvgAmount: 60.25,
		},
		Terminal: vectorize.Terminal{
			IsOnline:    false,
			CardPresent: true,
			KmFromHome:  29.23,
		},
		LastTransaction: nil,
	}

	got := vectorize.Vectorize(p, mccRisk, norm)

	expected := [14]float32{
		0.004112, 0.1667, 0.05, 0.7826, 0.3333,
		-1, -1,
		0.02923, 0.15, 0, 1, 0, 0.15, 0.006025,
	}

	for i, v := range expected {
		diff := got[i] - v
		if diff < 0 {
			diff = -diff
		}
		if diff > 0.001 {
			t.Errorf("dim[%d]: expected %.6f got %.6f (diff %.6f)", i, v, got[i], diff)
		}
	}
}

func TestVectorizeKnownFraud(t *testing.T) {
	norm := vectorize.Normalization{
		MaxAmount:            10000,
		MaxInstallments:      12,
		AmountVsAvgRatio:     10,
		MaxMinutes:           1440,
		MaxKm:                1000,
		MaxTxCount24h:        20,
		MaxMerchantAvgAmount: 10000,
	}
	mccRisk := map[string]float32{"7802": 0.75}

	requestedAt, _ := time.Parse(time.RFC3339, "2026-03-14T05:15:12Z")
	p := vectorize.Payload{
		ID: "tx-3330991687",
		Transaction: vectorize.Transaction{
			Amount:       9505.97,
			Installments: 10,
			RequestedAt:  requestedAt,
		},
		Customer: vectorize.Customer{
			AvgAmount:      81.28,
			TxCount24h:     20,
			KnownMerchants: []string{"MERC-008", "MERC-007", "MERC-005"},
		},
		Merchant: vectorize.Merchant{
			ID:        "MERC-068",
			MCC:       "7802",
			AvgAmount: 54.86,
		},
		Terminal: vectorize.Terminal{
			IsOnline:    false,
			CardPresent: true,
			KmFromHome:  952.27,
		},
		LastTransaction: nil,
	}

	got := vectorize.Vectorize(p, mccRisk, norm)

	expected := [14]float32{
		0.9506, 0.8333, 1.0, 0.2174, 0.8333,
		-1, -1,
		0.9523, 1.0, 0, 1, 1, 0.75, 0.0055,
	}

	for i, v := range expected {
		diff := got[i] - v
		if diff < 0 {
			diff = -diff
		}
		if diff > 0.001 {
			t.Errorf("dim[%d]: expected %.4f got %.4f (diff %.4f)", i, v, got[i], diff)
		}
	}
}
