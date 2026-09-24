package gen

import (
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/dashmage/namegen/internal/defaults"
)

type Options struct {
	MaxAttempts int
	Count       int
	Length      int
	Threshold   int
	TuneEnabled bool
}

var rhythmTemplates = []struct {
	Pattern string
	Weight  int
}{
	{Pattern: "CV", Weight: 5},
	{Pattern: "CVC", Weight: 6},
	{Pattern: "CVV", Weight: 2},
	{Pattern: "VC", Weight: 1},
}

var (
	rngMu sync.Mutex
	rng   = rand.New(rand.NewSource(time.Now().UnixNano()))
)

const initialNameCapacity = 64

// SetSeed sets the package-wide generator RNG to a deterministic seed. Concurrent
// calls are safe, though concurrent generation order is scheduling-dependent.
func SetSeed(seed int64) {
	rngMu.Lock()
	defer rngMu.Unlock()
	rng.Seed(seed)
}

// RandomName generates a random name of provided length. It is safe for concurrent use.
func RandomName(length int) string {
	if length <= 0 {
		return ""
	}

	rngMu.Lock()
	defer rngMu.Unlock()

	pattern := buildRhythmPattern(length)
	for i := range pattern {
		switch pattern[i] {
		case 'V':
			pattern[i] = randomVowel()
		default:
			pattern[i] = randomConsonant()
		}
	}
	return string(pattern)
}

// Generate creates pronounceable name candidates and populates results.
func Generate(opt Options) Result {
	result := Result{
		RequestedCount: opt.Count,
		Names:          make([]AcceptedName, 0),
		Threshold:      opt.Threshold,
		RuleHits:       NewRuleHits(),
	}
	if opt.TuneEnabled {
		result.AttemptLog = make([]Attempt, 0)
	}
	if opt.Count <= 0 || opt.MaxAttempts <= 0 || opt.Length <= 0 {
		return result
	}

	result.Names = make([]AcceptedName, 0, min(opt.Count, opt.MaxAttempts, initialNameCapacity))
	seenNames := make(map[string]struct{})
	for len(result.Names) < opt.Count {
		if result.Attempts >= opt.MaxAttempts {
			break
		}

		candidate := RandomName(opt.Length)
		evaluation := Evaluate(candidate, &result.RuleHits, opt.TuneEnabled)
		result.Attempts++

		entry := Attempt{
			Candidate:        candidate,
			Score:            evaluation.Score,
			Threshold:        opt.Threshold,
			HardRule:         evaluation.HardRule,
			SoftRules:        append([]Rule(nil), evaluation.SoftRules...),
			ProbabilityBand:  evaluation.ProbabilityBand,
			AvgLogProb:       evaluation.AvgLogProb,
			BigramAdjustment: evaluation.BigramAdjustment,
		}

		if evaluation.HardReject {
			result.HardRejects++
			entry.RejectReason = "hard_rule"
			if opt.TuneEnabled {
				result.AttemptLog = append(result.AttemptLog, entry)
			}
			continue
		}

		if evaluation.Score <= opt.Threshold {
			result.LowScoreRejects++
			entry.RejectReason = "low_score"
			if opt.TuneEnabled {
				result.AttemptLog = append(result.AttemptLog, entry)
			}
			continue
		}

		if _, duplicate := seenNames[candidate]; duplicate {
			result.DuplicateRejects++
			entry.RejectReason = "duplicate"
			if opt.TuneEnabled {
				result.AttemptLog = append(result.AttemptLog, entry)
			}
			continue
		}
		seenNames[candidate] = struct{}{}

		result.Names = append(result.Names, AcceptedName{
			Name:            candidate,
			Score:           evaluation.Score,
			ProbabilityBand: evaluation.ProbabilityBand,
		})
		entry.Accepted = true
		if opt.TuneEnabled {
			result.AttemptLog = append(result.AttemptLog, entry)
		}
	}

	return result
}

// isVowel reports whether ch exists in the configured vowel set.
func isVowel(ch byte) bool {
	return strings.ContainsRune(defaults.Vowels, rune(ch))
}

// buildRhythmPattern assembles a weighted CV pattern to the requested length.
func buildRhythmPattern(length int) []byte {
	pattern := make([]byte, 0, length)

	for len(pattern) < length {
		next := weightedTemplate()
		remaining := length - len(pattern)
		if len(next) > remaining {
			next = next[:remaining]
		}
		pattern = append(pattern, next...)
	}

	for i := 1; i < len(pattern)-1; i++ {
		if pattern[i-1] == 'V' && pattern[i] == 'V' && pattern[i+1] == 'V' {
			pattern[i] = 'C'
		}
	}

	if len(pattern) > 0 && pattern[len(pattern)-1] == 'V' && rng.Intn(100) < defaults.FinalConsonantBiasPercent {
		pattern[len(pattern)-1] = 'C'
	}

	return pattern
}

// weightedTemplate chooses a rhythm template using configured weights.
func weightedTemplate() string {
	total := 0
	for _, t := range rhythmTemplates {
		total += t.Weight
	}
	roll := rng.Intn(total)
	for _, t := range rhythmTemplates {
		if roll < t.Weight {
			return t.Pattern
		}
		roll -= t.Weight
	}
	return "CVC"
}

// randomConsonant returns a random consonant from the default pool.
func randomConsonant() byte {
	idx := rng.Intn(len(defaults.Consonants))
	return defaults.Consonants[idx]
}

// randomVowel returns a weighted random vowel from an internal pool.
func randomVowel() byte {
	// De-emphasize 'y' as a vowel.
	weightedVowelPool := "aaaaeeeiioouuy"
	idx := rng.Intn(len(weightedVowelPool))
	return weightedVowelPool[idx]
}
