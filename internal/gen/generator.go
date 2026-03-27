package gen

import (
	"math/rand"
	"strings"
	"time"

	"github.com/dashmage/namegen/internal/cli"
	"github.com/dashmage/namegen/internal/defaults"
)

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

// RandomPronounceableWord generates a random word that's pronounceable
func RandomPronounceableWord(config cli.Config) {
	wordsGenerated := 0
	curAttempts := 0
	hardRejects := 0
	lowScoreRejects := 0
	hardRuleHitCounts := make(map[string]int)
	softRuleHitCounts := make(map[string]int)

	for wordsGenerated < config.Count {
		if curAttempts >= config.Attempts {
			break
		}

		word := RandomWord(config.Length)
		score, hardReject, evalDebug := Evaluate(word)
		curAttempts++

		for _, hit := range evalDebug.HardRuleHits {
			hardRuleHitCounts[hit.Name]++
		}
		for _, hit := range evalDebug.SoftRuleHits {
			softRuleHitCounts[hit.Name]++
		}

		if hardReject {
			hardRejects++
			continue
		}

		if score <= config.Threshold {
			lowScoreRejects++
			continue
		}

		cli.PrintAcceptedWord(word, score, config.Debug)
		wordsGenerated++
	}

	if config.Debug {
		cli.PrintDebugSummary(cli.DebugSummary{
			Attempts:        curAttempts,
			Accepted:        wordsGenerated,
			HardRejects:     hardRejects,
			LowScoreRejects: lowScoreRejects,
			Threshold:       config.Threshold,
			RunSeed:         config.RunSeed,
			SeedSet:         config.SeedSet,
			HardRuleHits:    buildHardRuleStats(hardRuleHitCounts),
			SoftRuleHits:    buildSoftRuleStats(softRuleHitCounts),
		})
	}
}

func buildHardRuleStats(hitCounts map[string]int) []cli.RuleStat {
	stats := make([]cli.RuleStat, 0, len(HardRules))
	for _, rule := range HardRules {
		hits := hitCounts[rule.Name]
		if hits == 0 {
			continue
		}
		stats = append(stats, cli.RuleStat{
			Name:        rule.Name,
			Hits:        hits,
			Description: rule.Description,
		})
	}
	return stats
}

func buildSoftRuleStats(hitCounts map[string]int) []cli.RuleStat {
	stats := make([]cli.RuleStat, 0, len(SoftRules))
	for _, rule := range SoftRules {
		hits := hitCounts[rule.Name]
		if hits == 0 {
			continue
		}
		stats = append(stats, cli.RuleStat{
			Name:        rule.Name,
			Hits:        hits,
			Penalty:     rule.Penalty,
			Description: rule.Description,
		})
	}
	return stats
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
