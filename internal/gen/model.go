package gen

import (
	"math"

	"github.com/dashmage/namegen/internal/defaults"
)

type ProbabilityBand struct {
	Name  string
	Value int
}

var (
	probBandUnknown = ProbabilityBand{Name: "unknown", Value: 0}
	probBandVeryLow = ProbabilityBand{Name: "vlow", Value: -defaults.VeryLowProbPenalty}
	probBandLow     = ProbabilityBand{Name: "low", Value: -defaults.LowProbPenalty}
	probBandMid     = ProbabilityBand{Name: "mid", Value: -defaults.MidProbPenalty}
	probBandGood    = ProbabilityBand{Name: "good", Value: defaults.GoodProbBonus}
)

// BigramModel stores transition counts and smoothing configuration.
type BigramModel struct {
	BigramCounts map[[2]byte]int // bigram counts
	RowTotals    map[byte]int    // outgoing totals per first char
	Alpha        float64         // laplace smoothing factor
}

// NewBigramModel creates a model with Laplace smoothing parameter alpha.
// If alpha is <= 0, it defaults to 0.5.
func NewBigramModel(alpha float64) *BigramModel {
	if alpha <= 0 {
		alpha = defaults.BaseAlpha
	}

	return &BigramModel{
		BigramCounts: make(map[[2]byte]int),
		RowTotals:    make(map[byte]int),
		Alpha:        alpha,
	}
}

// Train updates bigram and row counts using normalized corpus words.
func (m *BigramModel) Train(words []string) {
	for _, word := range words {
		clean := normalizeWord(word)
		if clean == "" {
			continue
		}

		buf := make([]byte, 0, len(clean)+2)
		buf = append(buf, defaults.StartToken)
		buf = append(buf, clean...)
		buf = append(buf, defaults.EndToken)

		for i := 0; i < len(buf)-1; i++ {
			a := buf[i]
			b := buf[i+1]
			key := [2]byte{a, b}
			m.BigramCounts[key]++
			m.RowTotals[a]++
		}
	}
}

// LogProb returns log P(b|a) with Laplace smoothing.
func (m *BigramModel) LogProb(a, b byte) float64 {
	key := [2]byte{a, b}
	numerator := float64(m.BigramCounts[key]) + m.Alpha
	denominator := float64(m.RowTotals[a]) + m.Alpha*float64(defaults.VocabSize)
	return math.Log(numerator / denominator)
}

// AvgLogProb returns the mean log-probability of transitions in a word.
// It includes start and end boundary transitions.
func (m *BigramModel) AvgLogProb(word string) float64 {
	clean := normalizeWord(word)
	if clean == "" {
		return math.Inf(-1)
	}

	buf := make([]byte, 0, len(clean)+2)
	buf = append(buf, defaults.StartToken)
	buf = append(buf, clean...)
	buf = append(buf, defaults.EndToken)

	sum := 0.0
	steps := 0
	for i := 0; i < len(buf)-1; i++ {
		sum += m.LogProb(buf[i], buf[i+1])
		steps++
	}

	if steps == 0 {
		return math.Inf(-1)
	}

	return sum / float64(steps)
}

// probabilityBandFor returns the probability band name for a particular cutoff value
func probabilityBandFor(avgLogProb float64) ProbabilityBand {
	switch {
	case avgLogProb < defaults.VeryLowProbCutoff:
		return probBandVeryLow
	case avgLogProb < defaults.LowProbCutoff:
		return probBandLow
	case avgLogProb < defaults.MidProbCutoff:
		return probBandMid
	default:
		return probBandGood
	}
}

// ScoreAdjustment maps average bigram log-probability into a score adjustment.
// Low-probability transitions apply penalties; strong transitions can add a small bonus.
func (m *BigramModel) ScoreAdjustment(word string) (band ProbabilityBand, avgLogProb float64) {
	avgLogProb = m.AvgLogProb(word)
	band = probabilityBandFor(avgLogProb)
	return band, avgLogProb
}

// normalizeWord lowercases ASCII letters and removes non a-z bytes.
func normalizeWord(s string) string {
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		b := s[i]
		if b >= 'A' && b <= 'Z' {
			b = b + ('a' - 'A')
		}
		if b >= 'a' && b <= 'z' {
			out = append(out, b)
		}
	}
	return string(out)
}
