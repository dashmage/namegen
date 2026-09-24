package gen

import (
	"math"

	"github.com/dashmage/namegen/internal/defaults"
)

// ProbabilityBand labels an average character trigram log-probability range.
// Value is the rounded score adjustment computed from the continuous score.
type ProbabilityBand struct {
	Name  string
	Value int
}

var (
	probBandUnknown = ProbabilityBand{Name: "unknown"}
	probBandVeryLow = ProbabilityBand{Name: "vlow"}
	probBandLow     = ProbabilityBand{Name: "low"}
	probBandMid     = ProbabilityBand{Name: "mid"}
	probBandGood    = ProbabilityBand{Name: "good"}
)

// TrigramModel stores smoothed character transitions conditioned on the two
// preceding characters.
type TrigramModel struct {
	TrigramCounts map[[3]byte]int
	RowTotals     map[[2]byte]int
	Alpha         float64
}

// NewTrigramModel creates a character trigram model with Laplace smoothing.
// If alpha is <= 0, it defaults to 0.5.
func NewTrigramModel(alpha float64) *TrigramModel {
	if alpha <= 0 {
		alpha = defaults.BaseAlpha
	}
	return &TrigramModel{
		TrigramCounts: make(map[[3]byte]int),
		RowTotals:     make(map[[2]byte]int),
		Alpha:         alpha,
	}
}

// Train updates trigram counts from normalized corpus words. Two start tokens
// allow the model to score the first letter with its beginning-of-word context.
func (m *TrigramModel) Train(words []string) {
	for _, word := range words {
		clean := normalizeWord(word)
		if clean == "" {
			continue
		}

		buf := make([]byte, 0, len(clean)+3)
		buf = append(buf, defaults.StartToken, defaults.StartToken)
		buf = append(buf, clean...)
		buf = append(buf, defaults.EndToken)

		for i := 0; i < len(buf)-2; i++ {
			context := [2]byte{buf[i], buf[i+1]}
			trigram := [3]byte{buf[i], buf[i+1], buf[i+2]}
			m.TrigramCounts[trigram]++
			m.RowTotals[context]++
		}
	}
}

// LogProb returns log P(c|ab) with Laplace smoothing.
func (m *TrigramModel) LogProb(a, b, c byte) float64 {
	context := [2]byte{a, b}
	key := [3]byte{a, b, c}
	numerator := float64(m.TrigramCounts[key]) + m.Alpha
	denominator := float64(m.RowTotals[context]) + m.Alpha*float64(defaults.VocabSize)
	return math.Log(numerator / denominator)
}

// AvgLogProb returns average trigram log-probability over a word, including
// beginning- and end-of-word transitions.
func (m *TrigramModel) AvgLogProb(word string) float64 {
	clean := normalizeWord(word)
	if clean == "" {
		return math.Inf(-1)
	}

	previousPrevious := defaults.StartToken
	previous := defaults.StartToken
	sum := 0.0
	steps := 0
	for i := 0; i < len(clean); i++ {
		next := clean[i]
		sum += m.LogProb(previousPrevious, previous, next)
		previousPrevious = previous
		previous = next
		steps++
	}
	sum += m.LogProb(previousPrevious, previous, defaults.EndToken)
	steps++
	return sum / float64(steps)
}

// ScoreAdjustment maps average trigram log-probability into a continuous score
// adjustment while retaining a coarse probability-band label.
func (m *TrigramModel) ScoreAdjustment(word string) (band ProbabilityBand, avgLogProb float64) {
	avgLogProb = m.AvgLogProb(word)
	band = probabilityBandFor(avgLogProb)
	band.Value = scoreAdjustmentFor(avgLogProb)
	return band, avgLogProb
}

// probabilityBandFor returns the descriptive probability band for an average log-probability.
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

// scoreAdjustmentFor continuously maps average log-probability to the configured
// penalty/bonus range, interpolating between the cutoff anchors.
func scoreAdjustmentFor(avgLogProb float64) int {
	switch {
	case math.IsNaN(avgLogProb):
		return 0
	case avgLogProb <= defaults.VeryLowProbCutoff:
		return -defaults.VeryLowProbPenalty
	case avgLogProb <= defaults.LowProbCutoff:
		return interpolateScore(avgLogProb, defaults.VeryLowProbCutoff, -defaults.VeryLowProbPenalty, defaults.LowProbCutoff, -defaults.LowProbPenalty)
	case avgLogProb <= defaults.MidProbCutoff:
		return interpolateScore(avgLogProb, defaults.LowProbCutoff, -defaults.LowProbPenalty, defaults.MidProbCutoff, -defaults.MidProbPenalty)
	case avgLogProb <= defaults.GoodProbBonusCutoff:
		return interpolateScore(avgLogProb, defaults.MidProbCutoff, -defaults.MidProbPenalty, defaults.GoodProbBonusCutoff, defaults.GoodProbBonus)
	default:
		return defaults.GoodProbBonus
	}
}

func interpolateScore(x, x1 float64, y1 int, x2 float64, y2 int) int {
	fraction := (x - x1) / (x2 - x1)
	return int(math.Round(float64(y1) + fraction*float64(y2-y1)))
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
