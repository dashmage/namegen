package gen

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/dashmage/namegen/internal/defaults"
)

type Config struct {
	Attempts  int
	Count     int
	Length    int
	Threshold int
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

func SetSeed(seed int64) {
	rng.Seed(seed)
}

func SeedWithTime() int64 {
	seed := time.Now().UnixNano()
	rng.Seed(seed)
	return seed
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
func Generate(config Config) RunResult {
	result := RunResult{
		Words: make([]ScoredWord, 0, config.Count),
		Stats: GenStats{
			Threshold: config.Threshold,
			RuleHits:  NewRuleCounters(),
		},
	}

	for result.Stats.Accepted < config.Count {
		if result.Stats.Attempts >= config.Attempts {
			break
		}

		word := RandomWord(config.Length)
		score, hardReject, probBand := Evaluate(word, &result.Stats.RuleHits)
		fmt.Printf("word=%s, score=%d, probBand=%s\n\n", word, score, probBand)
		result.Stats.Attempts++

		if hardReject {
			result.Stats.HardRejects++
			continue
		}

		if score <= config.Threshold {
			result.Stats.LowScoreRejects++
			continue
		}

		result.Words = append(result.Words, ScoredWord{
			Word:       word,
			Score:      score,
			BigramProb: string(probBand),
		})
		result.Stats.Accepted++
	}

	return result
}

func checkVowel(randChar string) bool {
	return strings.Contains(defaults.Vowels, randChar)
}

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

func randomConsonant() byte {
	idx := rng.Intn(len(defaults.Consonants))
	return defaults.Consonants[idx]
}

func randomVowel() byte {
	// De-emphasize 'y' as a vowel.
	WeightedVowelPool := "aaaaeeeiioouuy"
	idx := rng.Intn(len(WeightedVowelPool))
	return WeightedVowelPool[idx]
}
