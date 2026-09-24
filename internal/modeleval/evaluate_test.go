package modeleval

import (
	"fmt"
	"math"
	"strings"
	"testing"
)

func TestSplitCorpusIsDeterministicAndKeepsNormalizedDuplicatesTogether(t *testing.T) {
	words := make([]string, 0, 502)
	for i := 0; i < 500; i++ {
		words = append(words, fmt.Sprintf("n%c%c%c", 'a'+i/676, 'a'+(i/26)%26, 'a'+i%26))
	}
	words = append(words, strings.ToUpper(words[0]), words[0])

	first, err := SplitCorpus(words, 42)
	if err != nil {
		t.Fatalf("SplitCorpus() error = %v", err)
	}
	second, err := SplitCorpus(words, 42)
	if err != nil {
		t.Fatalf("second SplitCorpus() error = %v", err)
	}
	if !equalStrings(first.Train, second.Train) || !equalStrings(first.Validation, second.Validation) || !equalStrings(first.Test, second.Test) {
		t.Fatal("same seed produced different corpus partitions")
	}

	assignments := make(map[string]string)
	for partition, names := range map[string][]string{
		"train": first.Train, "validation": first.Validation, "test": first.Test,
	} {
		for _, name := range names {
			if previous, exists := assignments[name]; exists {
				t.Fatalf("name %q occurs in both %s and %s", name, previous, partition)
			}
			assignments[name] = partition
		}
	}
	if len(assignments) != 500 {
		t.Fatalf("unique normalized names = %d, want 500", len(assignments))
	}
}

func TestRankCorrelationHandlesTies(t *testing.T) {
	positive := rankCorrelation([]float64{1, 2, 2, 4}, []float64{10, 20, 20, 40})
	if math.Abs(positive-1) > 1e-12 {
		t.Fatalf("positive rank correlation = %f, want 1", positive)
	}
	negative := rankCorrelation([]float64{1, 2, 2, 4}, []float64{40, 20, 20, 10})
	if math.Abs(negative+1) > 1e-12 {
		t.Fatalf("negative rank correlation = %f, want -1", negative)
	}
}

func TestNormalizeRatingsAveragesRepeatedNames(t *testing.T) {
	ratings, err := normalizeRatings([]Rating{
		{Name: "Lora", Value: 5},
		{Name: "lo-ra", Value: 3},
		{Name: "Mira", Value: 4},
	})
	if err != nil {
		t.Fatalf("normalizeRatings() error = %v", err)
	}
	if len(ratings) != 2 || ratings[0].Name != "lora" || ratings[0].Value != 4 {
		t.Fatalf("normalized ratings = %+v, want averaged lora and mira ratings", ratings)
	}
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}
