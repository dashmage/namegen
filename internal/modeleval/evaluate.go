package modeleval

import (
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"math"
	"sort"
	"strings"

	"github.com/dashmage/namegen/internal/defaults"
	"github.com/dashmage/namegen/internal/gen"
)

const (
	trainBuckets      = 14
	validationBuckets = 3
	totalBuckets      = 20
	negativeTryLimit  = 1000
)

type Split struct {
	Train      []string
	Validation []string
	Test       []string
}

type ValidationScore struct {
	BackoffStrength float64
	MeanLogProb     float64
	CrossEntropy    float64
}

type Metrics struct {
	HeldoutNames           int
	GeneratedNames         int
	MeanHeldoutLogProb     float64
	MeanGeneratedLogProb   float64
	HeldoutCrossEntropy    float64
	GeneratedCrossEntropy  float64
	MeanPairwiseLogProbGap float64
	PairwiseAccuracy       float64
}

type Rating struct {
	Name  string
	Value float64
}

type RatingMetrics struct {
	Names           int
	BigramSpearman  float64
	TrigramSpearman float64
}

type Report struct {
	CorpusNames             int
	TrainNames              int
	ValidationNames         int
	TestNames               int
	ValidationBigramLogProb float64
	TrigramValidationScores []ValidationScore
	SelectedBackoffStrength float64
	BigramTest              Metrics
	TrigramTest             Metrics
	HumanRatings            *RatingMetrics
}

type scoredPair struct {
	positive string
	negative string
}

// NormalizeName lowercases ASCII letters and discards all other bytes, matching
// the character vocabulary used by the name models.
func NormalizeName(value string) string {
	var out strings.Builder
	out.Grow(len(value))
	for i := 0; i < len(value); i++ {
		ch := value[i]
		if ch >= 'A' && ch <= 'Z' {
			ch += 'a' - 'A'
		}
		if ch >= 'a' && ch <= 'z' {
			out.WriteByte(ch)
		}
	}
	return out.String()
}

// SplitCorpus creates deterministic, content-based 70/15/15 train, validation,
// and test splits. Normalized duplicates always stay in the same partition.
func SplitCorpus(words []string, seed uint32) (Split, error) {
	unique := make(map[string]struct{}, len(words))
	for _, word := range words {
		clean := NormalizeName(word)
		if len(clean) < 2 {
			continue
		}
		unique[clean] = struct{}{}
	}

	names := make([]string, 0, len(unique))
	for name := range unique {
		names = append(names, name)
	}
	sort.Strings(names)
	if len(names) < 3 {
		return Split{}, fmt.Errorf("need at least 3 unique normalized names, got %d", len(names))
	}

	var split Split
	for _, name := range names {
		bucket := splitBucket(name, seed)
		switch {
		case bucket < trainBuckets:
			split.Train = append(split.Train, name)
		case bucket < trainBuckets+validationBuckets:
			split.Validation = append(split.Validation, name)
		default:
			split.Test = append(split.Test, name)
		}
	}
	if len(split.Train) == 0 || len(split.Validation) == 0 || len(split.Test) == 0 {
		return Split{}, fmt.Errorf("split produced train=%d, validation=%d, test=%d names", len(split.Train), len(split.Validation), len(split.Test))
	}
	return split, nil
}

func splitBucket(name string, seed uint32) uint32 {
	var seedBytes [4]byte
	binary.LittleEndian.PutUint32(seedBytes[:], seed)
	hasher := crc32.NewIEEE()
	_, _ = hasher.Write(seedBytes[:])
	_, _ = hasher.Write([]byte(name))
	return hasher.Sum32() % totalBuckets
}

// Compare trains on the training partition, chooses trigram backoff using
// validation likelihood, then reports both models on an untouched test split.
func Compare(words []string, seed uint32, negativesPerName int, backoffStrengths []float64) (Report, error) {
	return CompareWithRatings(words, nil, seed, negativesPerName, backoffStrengths)
}

// CompareWithRatings runs the corpus comparison and optionally correlates model
// likelihoods with human ratings. Rated names are excluded from corpus splits
// to prevent exact-name leakage into training or validation.
func CompareWithRatings(words []string, ratings []Rating, seed uint32, negativesPerName int, backoffStrengths []float64) (Report, error) {
	if negativesPerName <= 0 {
		return Report{}, fmt.Errorf("negatives per name must be greater than 0")
	}
	if len(backoffStrengths) == 0 {
		return Report{}, fmt.Errorf("at least one trigram backoff strength is required")
	}
	for _, strength := range backoffStrengths {
		if strength < 0 || math.IsNaN(strength) || math.IsInf(strength, 0) {
			return Report{}, fmt.Errorf("backoff strength must be finite and non-negative, got %v", strength)
		}
	}

	cleanRatings, err := normalizeRatings(ratings)
	if err != nil {
		return Report{}, err
	}
	excludedRatings := make(map[string]struct{}, len(cleanRatings))
	for _, rating := range cleanRatings {
		excludedRatings[rating.Name] = struct{}{}
	}
	corpus := make([]string, 0, len(words))
	for _, word := range words {
		if _, rated := excludedRatings[NormalizeName(word)]; !rated {
			corpus = append(corpus, word)
		}
	}

	split, err := SplitCorpus(corpus, seed)
	if err != nil {
		return Report{}, err
	}

	bigram := gen.NewBigramModel(defaults.BaseAlpha)
	bigram.Train(split.Train)
	validationBigramLogProb := meanTransitionLogProb(split.Validation, bigram.AvgLogProb)

	validationScores := make([]ValidationScore, 0, len(backoffStrengths))
	selectedStrength := backoffStrengths[0]
	bestValidationLogProb := math.Inf(-1)
	for _, strength := range backoffStrengths {
		model := gen.NewInterpolatedTrigramModelWithBackoff(defaults.BaseAlpha, strength)
		model.Train(split.Train)
		logProb := meanTransitionLogProb(split.Validation, model.AvgLogProb)
		validationScores = append(validationScores, ValidationScore{
			BackoffStrength: strength,
			MeanLogProb:     logProb,
			CrossEntropy:    -logProb,
		})
		if logProb > bestValidationLogProb {
			bestValidationLogProb = logProb
			selectedStrength = strength
		}
	}

	trainingNames := append(append([]string(nil), split.Train...), split.Validation...)
	finalBigram := gen.NewBigramModel(defaults.BaseAlpha)
	finalBigram.Train(trainingNames)
	finalTrigram := gen.NewInterpolatedTrigramModelWithBackoff(defaults.BaseAlpha, selectedStrength)
	finalTrigram.Train(trainingNames)

	knownNames := make(map[string]struct{}, len(split.Train)+len(split.Validation)+len(split.Test)+len(cleanRatings))
	for _, name := range split.Train {
		knownNames[name] = struct{}{}
	}
	for _, name := range split.Validation {
		knownNames[name] = struct{}{}
	}
	for _, name := range split.Test {
		knownNames[name] = struct{}{}
	}
	for _, rating := range cleanRatings {
		knownNames[rating.Name] = struct{}{}
	}
	pairs, err := generatePairs(split.Test, knownNames, negativesPerName, seed)
	if err != nil {
		return Report{}, err
	}

	report := Report{
		CorpusNames:             len(split.Train) + len(split.Validation) + len(split.Test),
		TrainNames:              len(split.Train),
		ValidationNames:         len(split.Validation),
		TestNames:               len(split.Test),
		ValidationBigramLogProb: validationBigramLogProb,
		TrigramValidationScores: validationScores,
		SelectedBackoffStrength: selectedStrength,
		BigramTest:              evaluatePairs(pairs, finalBigram.AvgLogProb),
		TrigramTest:             evaluatePairs(pairs, finalTrigram.AvgLogProb),
	}
	if len(cleanRatings) > 0 {
		report.HumanRatings = &RatingMetrics{
			Names:           len(cleanRatings),
			BigramSpearman:  ratingSpearman(cleanRatings, finalBigram.AvgLogProb),
			TrigramSpearman: ratingSpearman(cleanRatings, finalTrigram.AvgLogProb),
		}
	}
	return report, nil
}

func normalizeRatings(ratings []Rating) ([]Rating, error) {
	totals := make(map[string]float64, len(ratings))
	counts := make(map[string]int, len(ratings))
	for _, rating := range ratings {
		name := NormalizeName(rating.Name)
		if len(name) < 2 {
			return nil, fmt.Errorf("rated name %q normalizes to fewer than 2 letters", rating.Name)
		}
		if rating.Value < 1 || rating.Value > 5 || math.IsNaN(rating.Value) || math.IsInf(rating.Value, 0) {
			return nil, fmt.Errorf("rating for %q must be between 1 and 5", rating.Name)
		}
		totals[name] += rating.Value
		counts[name]++
	}

	names := make([]string, 0, len(totals))
	for name := range totals {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]Rating, 0, len(names))
	for _, name := range names {
		out = append(out, Rating{Name: name, Value: totals[name] / float64(counts[name])})
	}
	return out, nil
}

func ratingSpearman(ratings []Rating, score func(string) float64) float64 {
	observed := make([]float64, len(ratings))
	predicted := make([]float64, len(ratings))
	for i, rating := range ratings {
		observed[i] = rating.Value
		predicted[i] = score(rating.Name)
	}
	return rankCorrelation(observed, predicted)
}

func rankCorrelation(left, right []float64) float64 {
	if len(left) != len(right) || len(left) < 2 {
		return 0
	}
	leftRanks := averageRanks(left)
	rightRanks := averageRanks(right)
	leftMean := mean(leftRanks)
	rightMean := mean(rightRanks)
	numerator := 0.0
	leftVariance := 0.0
	rightVariance := 0.0
	for i := range leftRanks {
		leftDelta := leftRanks[i] - leftMean
		rightDelta := rightRanks[i] - rightMean
		numerator += leftDelta * rightDelta
		leftVariance += leftDelta * leftDelta
		rightVariance += rightDelta * rightDelta
	}
	if leftVariance == 0 || rightVariance == 0 {
		return 0
	}
	return numerator / math.Sqrt(leftVariance*rightVariance)
}

func averageRanks(values []float64) []float64 {
	indices := make([]int, len(values))
	for i := range indices {
		indices[i] = i
	}
	sort.Slice(indices, func(i, j int) bool {
		return values[indices[i]] < values[indices[j]]
	})

	ranks := make([]float64, len(values))
	for start := 0; start < len(indices); {
		end := start + 1
		for end < len(indices) && values[indices[end]] == values[indices[start]] {
			end++
		}
		averageRank := float64(start+1+end) / 2
		for i := start; i < end; i++ {
			ranks[indices[i]] = averageRank
		}
		start = end
	}
	return ranks
}

func mean(values []float64) float64 {
	total := 0.0
	for _, value := range values {
		total += value
	}
	return total / float64(len(values))
}

func meanTransitionLogProb(words []string, score func(string) float64) float64 {
	totalLogProb := 0.0
	totalTransitions := 0
	for _, word := range words {
		transitions := len(word) + 1 // start and end boundaries
		totalLogProb += score(word) * float64(transitions)
		totalTransitions += transitions
	}
	return totalLogProb / float64(totalTransitions)
}

func generatePairs(testNames []string, knownNames map[string]struct{}, negativesPerName int, seed uint32) ([]scoredPair, error) {
	gen.SetSeed(int64(seed))
	pairs := make([]scoredPair, 0, len(testNames)*negativesPerName)
	for _, positive := range testNames {
		for i := 0; i < negativesPerName; i++ {
			found := false
			for attempt := 0; attempt < negativeTryLimit; attempt++ {
				negative := gen.RandomName(len(positive))
				if _, known := knownNames[negative]; known {
					continue
				}
				if hasHardRuleFailure(negative) {
					continue
				}
				pairs = append(pairs, scoredPair{positive: positive, negative: negative})
				found = true
				break
			}
			if !found {
				return nil, fmt.Errorf("could not generate a hard-rule-valid negative for %q", positive)
			}
		}
	}
	return pairs, nil
}

func hasHardRuleFailure(name string) bool {
	for _, rule := range gen.HardRules {
		if rule.Check(name) {
			return true
		}
	}
	return false
}

func evaluatePairs(pairs []scoredPair, score func(string) float64) Metrics {
	positiveLogProb := 0.0
	positiveTransitions := 0
	heldoutNames := make(map[string]struct{})
	negativeLogProb := 0.0
	negativeTransitions := 0
	gapTotal := 0.0
	correctPairs := 0.0

	for _, pair := range pairs {
		heldoutNames[pair.positive] = struct{}{}
		positive := score(pair.positive)
		negative := score(pair.negative)
		positiveLogProb += positive * float64(len(pair.positive)+1)
		positiveTransitions += len(pair.positive) + 1
		negativeLogProb += negative * float64(len(pair.negative)+1)
		negativeTransitions += len(pair.negative) + 1
		gapTotal += positive - negative
		switch {
		case positive > negative:
			correctPairs++
		case positive == negative:
			correctPairs += 0.5
		}
	}

	meanPositive := positiveLogProb / float64(positiveTransitions)
	meanNegative := negativeLogProb / float64(negativeTransitions)
	return Metrics{
		HeldoutNames:           len(heldoutNames),
		GeneratedNames:         len(pairs),
		MeanHeldoutLogProb:     meanPositive,
		MeanGeneratedLogProb:   meanNegative,
		HeldoutCrossEntropy:    -meanPositive,
		GeneratedCrossEntropy:  -meanNegative,
		MeanPairwiseLogProbGap: gapTotal / float64(len(pairs)),
		PairwiseAccuracy:       correctPairs / float64(len(pairs)),
	}
}
