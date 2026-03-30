package gen

import (
	"math"
	"sync"

	"github.com/dashmage/namegen/internal/data"
	"github.com/dashmage/namegen/internal/defaults"
)

var (
	defaultModelOnce sync.Once
	defaultModel     *BigramModel
	defaultModelErr  error
)

type RuleCounters struct {
	Hard map[string]int
	Soft map[string]int
}

func NewRuleCounters() RuleCounters {
	return RuleCounters{
		Hard: make(map[string]int),
		Soft: make(map[string]int),
	}
}

type Evaluation struct {
	Score            int
	HardReject       bool
	HardRule         string
	SoftRules        []RulePenalty
	ProbBand         probabilityBand
	BigramAdjustment int
	AvgLogProb       float64
}

// loadDefaultModel trains a reusable bigram model from the embedded corpus.
func loadDefaultModel() (*BigramModel, error) {
	defaultModelOnce.Do(func() {
		words, err := data.LoadWords()
		if err != nil {
			defaultModelErr = err
			return
		}

		m := NewBigramModel(defaults.BaseAlpha)
		m.Train(words)
		defaultModel = m
	})

	return defaultModel, defaultModelErr
}

// Evaluate scores a candidate word and records any rule hits.
func Evaluate(word string, counters *RuleCounters, captureDetails bool) Evaluation {
	evaluation := Evaluation{
		Score:      defaults.BaseScore,
		ProbBand:   probBandUnknown,
		AvgLogProb: math.NaN(),
	}

	for _, r := range HardRules {
		if r.Check(word) {
			if counters != nil {
				counters.Hard[r.Name]++
			}
			evaluation.Score = 0
			evaluation.HardReject = true
			evaluation.HardRule = r.Name
			return evaluation
		}
	}
	for _, r := range SoftRules {
		if r.Check(word) {
			evaluation.Score -= r.Penalty
			if counters != nil {
				counters.Soft[r.Name]++
			}
			if captureDetails {
				evaluation.SoftRules = append(evaluation.SoftRules, RulePenalty{
					Name:        r.Name,
					Penalty:     r.Penalty,
					Description: r.Description,
				})
			}
		}
	}

	model, err := loadDefaultModel()
	var adjustment int
	if err == nil && model != nil {
		adjustment, evaluation.ProbBand, evaluation.AvgLogProb = model.ScoreAdjustment(word)
		evaluation.BigramAdjustment = adjustment
		evaluation.Score += adjustment
	}

	return evaluation
}
