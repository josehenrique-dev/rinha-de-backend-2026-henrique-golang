package service

import (
	"github.com/josehenrique-dev/rinha-2026/internal/vectordb"
	"github.com/josehenrique-dev/rinha-2026/internal/vectorize"
)

type FraudService struct {
	index   *vectordb.Index
	mccRisk map[string]float32
	norm    vectorize.Normalization
}

func New(index *vectordb.Index, mccRisk map[string]float32, norm vectorize.Normalization) *FraudService {
	return &FraudService{index: index, mccRisk: mccRisk, norm: norm}
}

func (s *FraudService) Score(p vectorize.Payload) (bool, float32) {
	vec := vectorize.Vectorize(p, s.mccRisk, s.norm)
	score := s.index.Search(vec, 5)
	return score < 0.6, score
}
