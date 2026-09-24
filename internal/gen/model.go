package gen

import (
	"math"

	"github.com/dashmage/namegen/internal/defaults"
)

// ProbabilityBand labels an average bigram log-probability range. Value is the
// rounded score adjustment computed from the continuous probability score.
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

// BigramModel stores transition counts and smoothing configuration.
type BigramModel struct {
	BigramCounts map[[2]byte]int // bigram counts
	RowTotals    map[byte]int    // outgoing totals per first char
	Alpha        float64         // laplace smoothing factor
}

// InterpolatedTrigramModel combines smoothed trigram probabilities with bigram
// backoff for sparse two-character contexts.
type InterpolatedTrigramModel struct {
	bigram           *BigramModel
	trigramCounts    map[[3]byte]int
	trigramRowTotals map[[2]byte]int
	backoffStrength  float64
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

// NewInterpolatedTrigramModel creates a trigram model with the default bigram
// backoff strength.
func NewInterpolatedTrigramModel(alpha float64) *InterpolatedTrigramModel {
	return NewInterpolatedTrigramModelWithBackoff(alpha, defaults.TrigramBackoffStrength)
}

// NewInterpolatedTrigramModelWithBackoff creates a trigram model with a custom
// non-negative backoff strength. Zero uses trigram estimates whenever a context
// has been observed and still falls back to bigrams for unseen contexts.
func NewInterpolatedTrigramModelWithBackoff(alpha, backoffStrength float64) *InterpolatedTrigramModel {
	if backoffStrength < 0 {
		backoffStrength = defaults.TrigramBackoffStrength
	}
	bigram := NewBigramModel(alpha)
	return &InterpolatedTrigramModel{
		bigram:           bigram,
		trigramCounts:    make(map[[3]byte]int),
		trigramRowTotals: make(map[[2]byte]int),
		backoffStrength:  backoffStrength,
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

// Train updates trigram counts while training its bigram backoff model.
func (m *InterpolatedTrigramModel) Train(words []string) {
	m.bigram.Train(words)
	for _, word := range words {
		clean := normalizeWord(word)
		if clean == "" {
			continue
		}

		buf := make([]byte, 0, len(clean)+2)
		buf = append(buf, defaults.StartToken)
		buf = append(buf, clean...)
		buf = append(buf, defaults.EndToken)

		previousPrevious := defaults.StartToken
		for i := 0; i < len(buf)-1; i++ {
			previous := buf[i]
			next := buf[i+1]
			context := [2]byte{previousPrevious, previous}
			trigram := [3]byte{previousPrevious, previous, next}
			m.trigramCounts[trigram]++
			m.trigramRowTotals[context]++
			previousPrevious = previous
		}
	}
}

// InterpolatedLogProb returns a smoothed trigram probability backed off to the
// bigram model when the two-character context is sparse.
func (m *InterpolatedTrigramModel) InterpolatedLogProb(a, b, c byte) float64 {
	context := [2]byte{a, b}
	key := [3]byte{a, b, c}
	contextCount := m.trigramRowTotals[context]

	trigramNumerator := float64(m.trigramCounts[key]) + m.bigram.Alpha
	trigramDenominator := float64(contextCount) + m.bigram.Alpha*float64(defaults.VocabSize)
	trigramProbability := trigramNumerator / trigramDenominator
	bigramProbability := math.Exp(m.bigram.LogProb(b, c))

	trigramWeight := 0.0
	if contextCount > 0 {
		if m.backoffStrength == 0 {
			trigramWeight = 1
		} else {
			trigramWeight = float64(contextCount) / (float64(contextCount) + m.backoffStrength)
		}
	}
	probability := trigramWeight*trigramProbability + (1-trigramWeight)*bigramProbability
	return math.Log(probability)
}

// AvgLogProb returns mean interpolated trigram log-probability, including
// start and end boundary transitions.
func (m *InterpolatedTrigramModel) AvgLogProb(word string) float64 {
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
		sum += m.InterpolatedLogProb(previousPrevious, previous, next)
		previousPrevious = previous
		previous = next
		steps++
	}
	sum += m.InterpolatedLogProb(previousPrevious, previous, defaults.EndToken)
	steps++

	return sum / float64(steps)
}

// AvgLogProb returns the mean bigram log-probability of transitions in a word.
// It includes start and end boundary transitions and is retained as the baseline.
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

// ScoreAdjustment maps average bigram log-probability into a continuous score
// adjustment. ProbabilityBand.Name remains a coarse diagnostic label, while
// ProbabilityBand.Value contains the rounded adjustment used for scoring.
func (m *BigramModel) ScoreAdjustment(word string) (band ProbabilityBand, avgLogProb float64) {
	avgLogProb = m.AvgLogProb(word)
	band = probabilityBandFor(avgLogProb)
	band.Value = scoreAdjustmentFor(avgLogProb)
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
