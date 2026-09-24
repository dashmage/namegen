package gen

import (
	"fmt"
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
	Substring   string
	Threshold   int
	TuneEnabled bool
}

type rhythmTemplate struct {
	Pattern string
	Weight  int
}

var rhythmTemplates = []rhythmTemplate{
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

	return fillRhythmPattern(buildRhythmPattern(length), 0, "")
}

// RandomNameContaining generates a random name containing substring, with at
// least one generated character on each side. Matching is case-insensitive.
func RandomNameContaining(length int, substring string) string {
	if substring == "" {
		return RandomName(length)
	}
	if ValidateSubstring(length, substring) != nil {
		return ""
	}
	substring = strings.ToLower(substring)

	rngMu.Lock()
	defer rngMu.Unlock()

	pattern := buildRhythmPattern(length)
	start := 1 + rng.Intn(length-len(substring)-1)
	return fillRhythmPattern(pattern, start, substring)
}

func fillRhythmPattern(pattern []byte, fixedStart int, fixed string) string {
	fixedEnd := fixedStart + len(fixed)
	copy(pattern[fixedStart:fixedEnd], fixed)
	for i := range pattern {
		if i >= fixedStart && i < fixedEnd {
			continue
		}
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
	if opt.Count <= 0 || opt.MaxAttempts <= 0 || opt.Length <= 0 || ValidateSubstring(opt.Length, opt.Substring) != nil {
		return result
	}

	result.Names = make([]AcceptedName, 0, min(opt.Count, opt.MaxAttempts, initialNameCapacity))
	seenNames := make(map[string]struct{})
	for len(result.Names) < opt.Count {
		if result.Attempts >= opt.MaxAttempts {
			break
		}

		candidate := ""
		if opt.Substring == "" {
			candidate = RandomName(opt.Length)
		} else {
			candidate = RandomNameContaining(opt.Length, opt.Substring)
		}
		evaluation := Evaluate(candidate, &result.RuleHits, opt.TuneEnabled)
		result.Attempts++

		entry := Attempt{
			Candidate:       candidate,
			Score:           evaluation.Score,
			Threshold:       opt.Threshold,
			HardRule:        evaluation.HardRule,
			SoftRules:       append([]Rule(nil), evaluation.SoftRules...),
			ProbabilityBand: evaluation.ProbabilityBand,
			AvgLogProb:      evaluation.AvgLogProb,
			ModelAdjustment: evaluation.ModelAdjustment,
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

// ValidateSubstring checks that a required substring can fit within the given
// name length and does not itself violate hard pronunciation rules.
func ValidateSubstring(length int, substring string) error {
	if substring == "" {
		return nil
	}
	if length < len(substring)+2 {
		return fmt.Errorf("length must be at least %d when substring length is %d", len(substring)+2, len(substring))
	}
	for i := 0; i < len(substring); i++ {
		ch := substring[i]
		if (ch < 'a' || ch > 'z') && (ch < 'A' || ch > 'Z') {
			return fmt.Errorf("substring must contain only ASCII letters")
		}
	}
	substring = strings.ToLower(substring)
	if ThreeConsecutiveConsonants(substring) {
		return fmt.Errorf("substring contains three consecutive consonants")
	}
	if TripleSameLetter(substring) {
		return fmt.Errorf("substring contains three repeated letters")
	}
	if IllegalConsonantAdjacency(substring) {
		return fmt.Errorf("substring contains a disallowed consonant sequence")
	}
	return nil
}

// isVowel reports whether ch exists in the configured vowel set.
func isVowel(ch byte) bool {
	return strings.ContainsRune(defaults.Vowels, rune(ch))
}

// buildRhythmPattern assembles complete weighted syllable shapes to the requested length.
func buildRhythmPattern(length int) []byte {
	if length <= 0 {
		return nil
	}

	pattern := make([]byte, 0, length)
	remaining := length
	lastTemplate := ""
	for remaining > 0 {
		lastTemplate = weightedTemplate(remaining)
		pattern = append(pattern, lastTemplate...)
		remaining -= len(lastTemplate)
	}

	for i := 1; i < len(pattern)-1; i++ {
		if pattern[i-1] == 'V' && pattern[i] == 'V' && pattern[i+1] == 'V' {
			pattern[i] = 'C'
		}
	}

	// Add a final consonant only when the last syllable retains a vowel nucleus.
	if len(pattern) > 0 && pattern[len(pattern)-1] == 'V' && strings.Count(lastTemplate, "V") > 1 && rng.Intn(100) < defaults.FinalConsonantBiasPercent {
		pattern[len(pattern)-1] = 'C'
	}

	return pattern
}

// weightedTemplate selects a template that leaves either no remainder or a
// remainder large enough to form another complete template.
func weightedTemplate(remaining int) string {
	if remaining == 1 {
		return selectWeightedTemplate(rhythmTemplates)[:1]
	}

	eligible := make([]rhythmTemplate, 0, len(rhythmTemplates))
	for _, template := range rhythmTemplates {
		remainder := remaining - len(template.Pattern)
		if remainder < 0 || remainder == 1 {
			continue
		}
		eligible = append(eligible, template)
	}

	return selectWeightedTemplate(eligible)
}

func selectWeightedTemplate(templates []rhythmTemplate) string {
	totalWeight := 0
	for _, template := range templates {
		totalWeight += template.Weight
	}
	roll := rng.Intn(totalWeight)
	for _, template := range templates {
		if roll < template.Weight {
			return template.Pattern
		}
		roll -= template.Weight
	}
	return templates[len(templates)-1].Pattern
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
