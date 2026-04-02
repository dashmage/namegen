package gen

import (
	"math/rand"
	"strings"
	"time"

	"github.com/dashmage/namegen/internal/defaults"
)

type GenConfig struct {
	Attempts  int
	Count     int
	Length    int
	Threshold int
	Tune      bool
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

// RandomWord generates a random word of provided length
func RandomWord(length int) string {
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

// Generate creates pronounceable candidate words and populates stats.
func Generate(config GenConfig) RunResult {
	result := RunResult{
		Words: make([]ScoredWord, 0, config.Count),
		Stats: GenStats{
			Threshold: config.Threshold,
			RuleHits:  NewRuleHits(),
		},
	}
	if config.Tune {
		result.GenAttempts = make([]GenAttempt, 0, config.Attempts)
	}
	if config.Count <= 0 || config.Attempts <= 0 || config.Length <= 0 {
		return result
	}

	for result.Stats.Accepted < config.Count {
		if result.Stats.Attempts >= config.Attempts {
			break
		}

		word := RandomWord(config.Length)
		evaluation := Evaluate(word, &result.Stats.RuleHits, config.Tune)
		result.Stats.Attempts++

		entry := GenAttempt{
			Word:             word,
			Score:            evaluation.Score,
			Threshold:        config.Threshold,
			HardRule:         evaluation.HardRule,
			SoftRules:        append([]RulePenalty(nil), evaluation.SoftRules...),
			ProbabilityBand:  evaluation.ProbabilityBand,
			AvgLogProb:       evaluation.AvgLogProb,
			BigramAdjustment: evaluation.BigramAdjustment,
		}

		if evaluation.HardReject {
			result.Stats.HardRejects++
			entry.RejectReason = "hard_rule"
			if config.Tune {
				result.GenAttempts = append(result.GenAttempts, entry)
			}
			continue
		}

		if evaluation.Score <= config.Threshold {
			result.Stats.LowScoreRejects++
			entry.RejectReason = "low_score"
			if config.Tune {
				result.GenAttempts = append(result.GenAttempts, entry)
			}
			continue
		}

		result.Words = append(result.Words, ScoredWord{
			Word:            word,
			Score:           evaluation.Score,
			ProbabilityBand: evaluation.ProbabilityBand,
		})
		result.Stats.Accepted++
		entry.Accepted = true
		entry.RejectReason = "accepted"
		if config.Tune {
			result.GenAttempts = append(result.GenAttempts, entry)
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
