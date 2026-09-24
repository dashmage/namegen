package gen

import (
	"math"
	"sync"

	"github.com/dashmage/namegen/internal/data"
	"github.com/dashmage/namegen/internal/defaults"
)

var (
	defaultModelOnce sync.Once
	defaultModel     *TrigramModel
	defaultModelErr  error
)

type RuleHits struct {
	Hard map[string]int
	Soft map[string]int
}

func NewRuleHits() RuleHits {
	return RuleHits{
		Hard: make(map[string]int),
		Soft: make(map[string]int),
	}
}

type Evaluation struct {
	Score           int
	HardReject      bool
	HardRule        string
	SoftRules       []Rule
	ProbabilityBand ProbabilityBand
	ModelAdjustment int
	AvgLogProb      float64
}

// loadDefaultModel trains a reusable trigram model from the combined embedded
// human-name and company/brand corpora.
func loadDefaultModel() (*TrigramModel, error) {
	defaultModelOnce.Do(func() {
		words, err := data.LoadTrainingWords()
		if err != nil {
			defaultModelErr = err
			return
		}

		m := NewTrigramModel(defaults.BaseAlpha)
		m.Train(words)
		defaultModel = m
	})

	return defaultModel, defaultModelErr
}

// Evaluate scores a candidate name and records any rule hits.
func Evaluate(name string, hits *RuleHits, captureAttemptDetails bool) Evaluation {
	evaluation := Evaluation{
		Score:           defaults.BaseScore,
		ProbabilityBand: probBandUnknown,
		AvgLogProb:      math.NaN(),
	}

	for _, r := range HardRules {
		if r.Check(name) {
			if hits != nil {
				hits.Hard[r.Name]++
			}
			evaluation.Score = 0
			evaluation.HardReject = true
			evaluation.HardRule = r.Name
			return evaluation
		}
	}
	for _, r := range SoftRules {
		if r.Check(name) {
			evaluation.Score -= r.Penalty
			if hits != nil {
				hits.Soft[r.Name]++
			}
			if captureAttemptDetails {
				evaluation.SoftRules = append(evaluation.SoftRules, Rule{
					Name:        r.Name,
					Penalty:     r.Penalty,
					Description: r.Description,
				})
			}
		}
	}

	model, err := loadDefaultModel()
	if err == nil && model != nil {
		evaluation.ProbabilityBand, evaluation.AvgLogProb = model.ScoreAdjustment(name)
		evaluation.ModelAdjustment = evaluation.ProbabilityBand.Value
		evaluation.Score += evaluation.ModelAdjustment
	}

	return evaluation
}
