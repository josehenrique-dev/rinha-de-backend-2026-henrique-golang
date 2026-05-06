package service

import (
	"github.com/josehenrique-dev/rinha-2026/internal/vectorize"
)

type Searcher interface {
	SearchCount(query [14]float32, k int) int
}

type FraudService struct {
	index   Searcher
	mccRisk map[string]float32
	norm    vectorize.Normalization
}

func New(index Searcher, mccRisk map[string]float32, norm vectorize.Normalization) *FraudService {
	return &FraudService{index: index, mccRisk: mccRisk, norm: norm}
}

func (s *FraudService) Score(p vectorize.Payload) (bool, float32) {
	vec := vectorize.Vectorize(p, s.mccRisk, s.norm)
	score := float32(s.index.SearchCount(vec, 5)) / 5.0
	return score < 0.6, score
}

func (s *FraudService) FraudCount(p vectorize.Payload) int {
	vec := vectorize.Vectorize(p, s.mccRisk, s.norm)
	return s.index.SearchCount(vec, 5)
}
