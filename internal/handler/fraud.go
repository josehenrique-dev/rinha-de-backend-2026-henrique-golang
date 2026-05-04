package handler

import (
	"net/http"
	"time"

	gojson "github.com/goccy/go-json"
	"github.com/josehenrique-dev/rinha-2026/internal/service"
	"github.com/josehenrique-dev/rinha-2026/internal/vectorize"
)

type Handler struct {
	svc *service.FraudService
}

func New(svc *service.FraudService) *Handler {
	return &Handler{svc: svc}
}

type fraudRequest struct {
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
	LastTransaction *struct {
		Timestamp     string  `json:"timestamp"`
		KmFromCurrent float32 `json:"km_from_current"`
	} `json:"last_transaction"`
}

type fraudResponse struct {
	Approved   bool    `json:"approved"`
	FraudScore float32 `json:"fraud_score"`
}

func (h *Handler) Ready(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) FraudScore(w http.ResponseWriter, r *http.Request) {
	var req fraudRequest
	if err := gojson.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	requestedAt, err := time.Parse(time.RFC3339, req.Transaction.RequestedAt)
	if err != nil {
		http.Error(w, "invalid requested_at", http.StatusBadRequest)
		return
	}

	p := vectorize.Payload{
		ID: req.ID,
		Transaction: vectorize.Transaction{
			Amount:       req.Transaction.Amount,
			Installments: req.Transaction.Installments,
			RequestedAt:  requestedAt,
		},
		Customer: vectorize.Customer{
			AvgAmount:      req.Customer.AvgAmount,
			TxCount24h:     req.Customer.TxCount24h,
			KnownMerchants: req.Customer.KnownMerchants,
		},
		Merchant: vectorize.Merchant{
			ID:        req.Merchant.ID,
			MCC:       req.Merchant.MCC,
			AvgAmount: req.Merchant.AvgAmount,
		},
		Terminal: vectorize.Terminal{
			IsOnline:    req.Terminal.IsOnline,
			CardPresent: req.Terminal.CardPresent,
			KmFromHome:  req.Terminal.KmFromHome,
		},
	}

	if req.LastTransaction != nil {
		ts, err := time.Parse(time.RFC3339, req.LastTransaction.Timestamp)
		if err == nil {
			p.LastTransaction = &vectorize.LastTransaction{
				Timestamp:     ts,
				KmFromCurrent: req.LastTransaction.KmFromCurrent,
			}
		}
	}

	approved, score := h.svc.Score(p)

	w.Header().Set("Content-Type", "application/json")
	gojson.NewEncoder(w).Encode(fraudResponse{Approved: approved, FraudScore: score})
}
