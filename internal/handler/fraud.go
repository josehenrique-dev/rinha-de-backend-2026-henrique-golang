package handler

import (
	"bytes"
	"net/http"
	"sync"

	"github.com/josehenrique-dev/rinha-2026/internal/service"
	"github.com/josehenrique-dev/rinha-2026/internal/vectorize"
)

type Handler struct {
	svc *service.FraudService
}

func New(svc *service.FraudService) *Handler {
	return &Handler{svc: svc}
}

var precomputed [6][]byte

func init() {
	bodies := [6]string{
		`{"approved":true,"fraud_score":0.0}`,
		`{"approved":true,"fraud_score":0.2}`,
		`{"approved":true,"fraud_score":0.4}`,
		`{"approved":false,"fraud_score":0.6}`,
		`{"approved":false,"fraud_score":0.8}`,
		`{"approved":false,"fraud_score":1.0}`,
	}
	for i, b := range bodies {
		precomputed[i] = []byte(b + "\n")
	}
}

var bufPool = sync.Pool{
	New: func() any { return bytes.NewBuffer(make([]byte, 0, 4096)) },
}

func (h *Handler) Ready(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) FraudScore(w http.ResponseWriter, r *http.Request) {
	buf := bufPool.Get().(*bytes.Buffer)
	buf.Reset()
	buf.ReadFrom(r.Body)

	var p vectorize.Payload
	if err := ParsePayload(buf.Bytes(), &p); err != nil {
		bufPool.Put(buf)
		w.Header().Set("Content-Type", "application/json")
		w.Write(precomputed[0])
		return
	}
	bufPool.Put(buf)

	fraudCount := h.svc.FraudCount(p)
	w.Header().Set("Content-Type", "application/json")
	w.Write(precomputed[fraudCount])
}
