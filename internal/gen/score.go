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
func Evaluate(word string, counters *RuleCounters) (score int, hardReject bool) {
	score = defaults.BaseScore

	for _, r := range HardRules {
		if r.Check(word) {
			if counters != nil {
				counters.Hard[r.Name]++
			}
			return 0, true
		}
	}
	for _, r := range SoftRules {
		if r.Check(word) {
			score -= r.Penalty
			if counters != nil {
				counters.Soft[r.Name]++
			}
		}
	}

	model, err := loadDefaultModel()
	if err == nil && model != nil {
		score += model.ScoreAdjustment(word)
	}

	return score, false
}
