package gen

import (
	"math/rand"
	"strings"
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

var rng = rand.New(rand.NewSource(time.Now().UnixNano()))

// SetSeed sets the generator RNG to a deterministic seed.
func SetSeed(seed int64) {
	rng.Seed(seed)
}

// RandomName generates a random name of provided length.
func RandomName(length int) string {
	if length <= 0 {
		return ""
	}

	pattern := buildRhythmPattern(length)
	var res strings.Builder
	res.Grow(len(pattern))
	for i := 0; i < len(pattern); i++ {
		switch pattern[i] {
		case 'V':
			res.WriteByte(randomVowel())
		default:
			res.WriteByte(randomConsonant())
		}
	}
	return res.String()
}

// Generate creates pronounceable name candidates and populates results.
func Generate(opt Options) Result {
	result := Result{
		Names:     make([]AcceptedName, 0, opt.Count),
		Threshold: opt.Threshold,
		RuleHits:  NewRuleHits(),
	}
	if opt.TuneEnabled {
		result.AttemptLog = make([]Attempt, 0, opt.MaxAttempts)
	}
	if opt.Count <= 0 || opt.MaxAttempts <= 0 || opt.Length <= 0 {
		return result
	}

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
func buildRhythmPattern(length int) string {
	var pattern strings.Builder
	pattern.Grow(length)

	for pattern.Len() < length {
		next := weightedTemplate()
		remaining := length - pattern.Len()
		if len(next) > remaining {
			next = next[:remaining]
		}
		pattern.WriteString(next)
	}

	out := []byte(pattern.String())
	for i := 1; i < len(out)-1; i++ {
		if out[i-1] == 'V' && out[i] == 'V' && out[i+1] == 'V' {
			out[i] = 'C'
		}
	}

	if len(out) > 0 && out[len(out)-1] == 'V' && rng.Intn(100) < defaults.FinalConsonantBiasPercent {
		out[len(out)-1] = 'C'
	}

	return string(out)
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
