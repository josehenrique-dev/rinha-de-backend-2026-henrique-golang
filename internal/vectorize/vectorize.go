package vectorize

import (
	"fmt"
	"slices"
)

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
	RequestedAt  string
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
	Timestamp     string
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

func parseHourWeekday(s string) (hour, weekday int, err error) {
	if len(s) < 19 {
		return 0, 0, fmt.Errorf("rfc3339 too short")
	}
	h := int(s[11]-'0')*10 + int(s[12]-'0')
	if h < 0 || h > 23 {
		return 0, 0, fmt.Errorf("invalid hour")
	}
	year := int(s[0]-'0')*1000 + int(s[1]-'0')*100 + int(s[2]-'0')*10 + int(s[3]-'0')
	month := int(s[5]-'0')*10 + int(s[6]-'0')
	day := int(s[8]-'0')*10 + int(s[9]-'0')
	t := [12]int{0, 3, 2, 5, 0, 3, 5, 1, 4, 6, 2, 4}
	if month < 3 {
		year--
	}
	dow := (year + year/4 - year/100 + year/400 + t[month-1] + day) % 7
	wd := (dow + 6) % 7
	return h, wd, nil
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
	return slices.Contains(known, id)
}

func parseMinutesBetween(later, earlier string) float32 {
	if len(later) < 19 || len(earlier) < 19 {
		return 0
	}
	ly := int(later[0]-'0')*1000 + int(later[1]-'0')*100 + int(later[2]-'0')*10 + int(later[3]-'0')
	lm := int(later[5]-'0')*10 + int(later[6]-'0')
	ld := int(later[8]-'0')*10 + int(later[9]-'0')
	lh := int(later[11]-'0')*10 + int(later[12]-'0')
	lmin := int(later[14]-'0')*10 + int(later[15]-'0')
	ls := int(later[17]-'0')*10 + int(later[18]-'0')

	ey := int(earlier[0]-'0')*1000 + int(earlier[1]-'0')*100 + int(earlier[2]-'0')*10 + int(earlier[3]-'0')
	em := int(earlier[5]-'0')*10 + int(earlier[6]-'0')
	ed := int(earlier[8]-'0')*10 + int(earlier[9]-'0')
	eh := int(earlier[11]-'0')*10 + int(earlier[12]-'0')
	emin := int(earlier[14]-'0')*10 + int(earlier[15]-'0')
	es := int(earlier[17]-'0')*10 + int(earlier[18]-'0')

	lTotal := toUnixDays(ly, lm, ld)*1440 + lh*60 + lmin + ls/60
	eTotal := toUnixDays(ey, em, ed)*1440 + eh*60 + emin + es/60

	diff := lTotal - eTotal
	if diff < 0 {
		diff = 0
	}
	return float32(diff)
}

func toUnixDays(y, m, d int) int {
	if m <= 2 {
		y--
		m += 12
	}
	return 365*y + y/4 - y/100 + y/400 + (153*(m-3)+2)/5 + d - 719469
}

func Vectorize(p Payload, mccRisk map[string]float32, norm Normalization) [14]float32 {
	var v [14]float32

	v[0] = clamp(p.Transaction.Amount / norm.MaxAmount)
	v[1] = clamp(float32(p.Transaction.Installments) / norm.MaxInstallments)
	v[2] = clamp((p.Transaction.Amount / p.Customer.AvgAmount) / norm.AmountVsAvgRatio)

	hour, weekday, _ := parseHourWeekday(p.Transaction.RequestedAt)
	v[3] = float32(hour) / 23.0
	v[4] = float32(weekday) / 6.0

	if p.LastTransaction == nil {
		v[5] = -1
		v[6] = -1
	} else {
		minutes := parseMinutesBetween(p.Transaction.RequestedAt, p.LastTransaction.Timestamp)
		v[5] = clamp(minutes / norm.MaxMinutes)
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
