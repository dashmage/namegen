package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"strconv"
	"strings"

	"github.com/dashmage/namegen/internal/data"
	"github.com/dashmage/namegen/internal/modeleval"
)

func main() {
	corpusPath := flag.String("corpus", "", "cleaned one-name-per-line corpus to evaluate")
	ratingsPath := flag.String("ratings", "", "optional CSV with name,rating columns (ratings 1-5)")
	seed := flag.Uint("seed", 42, "deterministic split and generated-negative seed")
	negatives := flag.Int("negatives-per-name", 1, "hard-rule-valid generated comparisons per held-out name")
	backoffs := flag.String("backoffs", "0,1,5,10,20,50", "comma-separated trigram backoff strengths to compare on validation")
	flag.Parse()

	if *corpusPath == "" {
		fmt.Fprintln(os.Stderr, "--corpus is required")
		os.Exit(2)
	}
	words, err := data.LoadWordsFromFile(*corpusPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load corpus: %v\n", err)
		os.Exit(1)
	}
	strengths, err := parseBackoffStrengths(*backoffs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid --backoffs: %v\n", err)
		os.Exit(2)
	}

	var ratings []modeleval.Rating
	if *ratingsPath != "" {
		ratings, err = loadRatings(*ratingsPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "load ratings: %v\n", err)
			os.Exit(1)
		}
	}

	report, err := modeleval.CompareWithRatings(words, ratings, uint32(*seed), *negatives, strengths)
	if err != nil {
		fmt.Fprintf(os.Stderr, "evaluate models: %v\n", err)
		os.Exit(1)
	}
	printReport(report)
}

func parseBackoffStrengths(value string) ([]float64, error) {
	parts := strings.Split(value, ",")
	strengths := make([]float64, 0, len(parts))
	for _, part := range parts {
		strength, err := strconv.ParseFloat(strings.TrimSpace(part), 64)
		if err != nil || strength < 0 {
			return nil, fmt.Errorf("%q must be a non-negative number", part)
		}
		strengths = append(strengths, strength)
	}
	return strengths, nil
}

func loadRatings(path string) ([]modeleval.Rating, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1
	var ratings []modeleval.Rating
	line := 0
	for {
		record, err := reader.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("read CSV row %d: %w", line+1, err)
		}
		line++
		if len(record) == 1 && strings.TrimSpace(record[0]) == "" {
			continue
		}
		if len(record) != 2 {
			return nil, fmt.Errorf("row %d must have name,rating columns", line)
		}
		if line == 1 && strings.EqualFold(strings.TrimSpace(record[0]), "name") && strings.EqualFold(strings.TrimSpace(record[1]), "rating") {
			continue
		}
		value, err := strconv.ParseFloat(strings.TrimSpace(record[1]), 64)
		if err != nil || value < 1 || value > 5 || math.IsNaN(value) || math.IsInf(value, 0) {
			return nil, fmt.Errorf("row %d rating must be between 1 and 5", line)
		}
		ratings = append(ratings, modeleval.Rating{Name: strings.TrimSpace(record[0]), Value: value})
	}
	return ratings, nil
}

func printReport(report modeleval.Report) {
	fmt.Printf("Corpus: %d unique names (train=%d, validation=%d, test=%d)\n", report.CorpusNames, report.TrainNames, report.ValidationNames, report.TestNames)
	fmt.Printf("Bigram validation: mean log-probability=%.4f, cross-entropy=%.4f nats/transition\n", report.ValidationBigramLogProb, -report.ValidationBigramLogProb)
	fmt.Println("Interpolated trigram validation by backoff strength:")
	for _, score := range report.TrigramValidationScores {
		fmt.Printf("  strength=%-6g mean log-probability=%.4f cross-entropy=%.4f nats/transition\n", score.BackoffStrength, score.MeanLogProb, score.CrossEntropy)
	}
	fmt.Printf("Selected trigram backoff strength: %g\n", report.SelectedBackoffStrength)
	fmt.Println("Untouched test comparison against hard-rule-valid generated names:")
	printMetrics("bigram", report.BigramTest)
	printMetrics("trigram", report.TrigramTest)
	if ratings := report.HumanRatings; ratings != nil {
		fmt.Printf("Human ratings (%d names): Spearman rho bigram=%.4f trigram=%.4f\n", ratings.Names, ratings.BigramSpearman, ratings.TrigramSpearman)
	} else {
		fmt.Println("Human ratings: not supplied (use --ratings name,rating.csv)")
	}
}

func printMetrics(model string, metrics modeleval.Metrics) {
	fmt.Printf("  %-7s heldout=%d generated=%d heldout-cross-entropy=%.4f generated-cross-entropy=%.4f gap=%.4f pairwise-accuracy=%.3f\n",
		model,
		metrics.HeldoutNames,
		metrics.GeneratedNames,
		metrics.HeldoutCrossEntropy,
		metrics.GeneratedCrossEntropy,
		metrics.MeanPairwiseLogProbGap,
		metrics.PairwiseAccuracy,
	)
}
