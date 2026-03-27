package gen

import (
	"sync"

	"github.com/dashmage/namegen/internal/data"
	"github.com/dashmage/namegen/internal/defaults"
)

var (
	defaultModelOnce sync.Once
	defaultModel     *BigramModel
	defaultModelErr  error
)

type RuleHit struct {
	Name        string
	Description string
	Penalty     int
}

type EvalDebug struct {
	HardRuleHits []RuleHit
	SoftRuleHits []RuleHit
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

func ngramDelta(avgLogProb float64) int {
	switch {
	case avgLogProb < defaults.NGramVeryLowCutoff:
		return defaults.NGramVeryLowDelta
	case avgLogProb < defaults.NGramLowCutoff:
		return defaults.NGramLowDelta
	case avgLogProb < defaults.NGramMidCutoff:
		return defaults.NGramMidDelta
	default:
		return defaults.NGramGoodDelta
	}
}

// Evaluate includes per-rule hits for debugging and tuning.
func Evaluate(word string) (score int, hardReject bool, debug EvalDebug) {
	score = defaults.BaseScore

	for _, r := range HardRules {
		if r.Check(word) {
			debug.HardRuleHits = append(debug.HardRuleHits, RuleHit{
				Name:        r.Name,
				Description: r.Description,
			})
			return 0, true, debug
		}
	}
	for _, r := range SoftRules {
		if r.Check(word) {
			score -= r.Penalty
			debug.SoftRuleHits = append(debug.SoftRuleHits, RuleHit{
				Name:        r.Name,
				Description: r.Description,
				Penalty:     r.Penalty,
			})
		}
	}

	model, err := loadDefaultModel()
	if err == nil && model != nil {
		score += ngramDelta(model.AvgLogProb(word))
	}

	return score, false, debug
}
