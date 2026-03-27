package gen

import (
	"math/rand"
	"strings"
	"time"

	"github.com/dashmage/namegen/internal/cli"
	"github.com/dashmage/namegen/internal/defaults"
)

type generationStats struct {
	Attempts        int
	Accepted        int
	HardRejects     int
	LowScoreRejects int
	RuleHits        RuleCounters
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

// RandomPronounceableWord generates a random word that's pronounceable
func RandomPronounceableWord(config cli.Config) {
	stats := generationStats{
		RuleHits: NewRuleCounters(),
	}

	for stats.Accepted < config.Count {
		if stats.Attempts >= config.Attempts {
			break
		}

		word := RandomWord(config.Length)
		score, hardReject := Evaluate(word, &stats.RuleHits)
		stats.Attempts++

		if hardReject {
			stats.HardRejects++
			continue
		}

		if score <= config.Threshold {
			stats.LowScoreRejects++
			continue
		}

		cli.PrintAcceptedWord(word, score, config.Debug)
		stats.Accepted++
	}

	if config.Debug {
		cli.PrintDebugSummary(cli.DebugSummary{
			Attempts:        stats.Attempts,
			Accepted:        stats.Accepted,
			HardRejects:     stats.HardRejects,
			LowScoreRejects: stats.LowScoreRejects,
			Threshold:       config.Threshold,
			RunSeed:         config.RunSeed,
			SeedSet:         config.SeedSet,
			HardRuleHits:    buildHardRuleStats(stats.RuleHits),
			SoftRuleHits:    buildSoftRuleStats(stats.RuleHits),
		})
	}
}

func buildHardRuleStats(hitCounts RuleCounters) []cli.RuleStat {
	stats := make([]cli.RuleStat, 0, len(HardRules))
	for _, rule := range HardRules {
		hits := hitCounts.Hard[rule.Name]
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

func buildSoftRuleStats(hitCounts RuleCounters) []cli.RuleStat {
	stats := make([]cli.RuleStat, 0, len(SoftRules))
	for _, rule := range SoftRules {
		hits := hitCounts.Soft[rule.Name]
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
