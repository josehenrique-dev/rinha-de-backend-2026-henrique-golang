package vectorize

import "time"

type Normalization struct {
	MaxAmount            float32
	MaxInstallments      float32
	AmountVsAvgRatio     float32
	MaxMinutes           float32
	MaxKm                float32
	MaxTxCount24h        float32
	MaxMerchantAvgAmount float32
}

type Transaction struct {
	Amount       float32
	Installments int
	RequestedAt  time.Time
}

type Customer struct {
	AvgAmount      float32
	TxCount24h     int
	KnownMerchants []string
}

type Merchant struct {
	ID        string
	MCC       string
	AvgAmount float32
}

type Terminal struct {
	IsOnline    bool
	CardPresent bool
	KmFromHome  float32
}

type LastTransaction struct {
	Timestamp     time.Time
	KmFromCurrent float32
}

type Payload struct {
	ID              string
	Transaction     Transaction
	Customer        Customer
	Merchant        Merchant
	Terminal        Terminal
	LastTransaction *LastTransaction
}

func clamp(v float32) float32 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func isKnownMerchant(id string, known []string) bool {
	for _, m := range known {
		if m == id {
			return true
		}
	}
	return false
}

func Vectorize(p Payload, mccRisk map[string]float32, norm Normalization) [14]float32 {
	var v [14]float32

	v[0] = clamp(p.Transaction.Amount / norm.MaxAmount)
	v[1] = clamp(float32(p.Transaction.Installments) / norm.MaxInstallments)
	v[2] = clamp((p.Transaction.Amount / p.Customer.AvgAmount) / norm.AmountVsAvgRatio)

	hour := float32(p.Transaction.RequestedAt.UTC().Hour())
	v[3] = hour / 23.0

	wd := p.Transaction.RequestedAt.UTC().Weekday()
	var dayIdx float32
	if wd == time.Sunday {
		dayIdx = 6
	} else {
		dayIdx = float32(wd) - 1
	}
	v[4] = dayIdx / 6.0

	if p.LastTransaction == nil {
		v[5] = -1
		v[6] = -1
	} else {
		minutes := p.Transaction.RequestedAt.Sub(p.LastTransaction.Timestamp).Minutes()
		v[5] = clamp(float32(minutes) / norm.MaxMinutes)
		v[6] = clamp(p.LastTransaction.KmFromCurrent / norm.MaxKm)
	}

	v[7] = clamp(p.Terminal.KmFromHome / norm.MaxKm)
	v[8] = clamp(float32(p.Customer.TxCount24h) / norm.MaxTxCount24h)

	if p.Terminal.IsOnline {
		v[9] = 1
	}
	if p.Terminal.CardPresent {
		v[10] = 1
	}
	if !isKnownMerchant(p.Merchant.ID, p.Customer.KnownMerchants) {
		v[11] = 1
	}

	mcc, ok := mccRisk[p.Merchant.MCC]
	if !ok {
		mcc = 0.5
	}
	v[12] = mcc

	v[13] = clamp(p.Merchant.AvgAmount / norm.MaxMerchantAvgAmount)

	return v
}
