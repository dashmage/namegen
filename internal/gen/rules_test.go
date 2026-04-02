package gen

import (
	"testing"

	"github.com/dashmage/namegen/internal/defaults"
)

func TestEvaluateHardRuleShortCircuitsScoring(t *testing.T) {
	hits := NewRuleHits()

	evaluation := Evaluate("bcd", &hits, true)

	if !evaluation.HardReject {
		t.Fatalf("expected hard reject")
	}
	if evaluation.HardRule != "three_consecutive_consonants" {
		t.Fatalf("HardRule = %q, want %q", evaluation.HardRule, "three_consecutive_consonants")
	}
	if evaluation.Score != 0 {
		t.Fatalf("Score = %d, want 0", evaluation.Score)
	}
	if len(evaluation.SoftRules) != 0 {
		t.Fatalf("SoftRules length = %d, want 0", len(evaluation.SoftRules))
	}
	if hits.Hard["three_consecutive_consonants"] != 1 {
		t.Fatalf("hard hit count = %d, want 1", hits.Hard["three_consecutive_consonants"])
	}
}

func TestEvaluateCapturesSoftPenaltiesAndDetails(t *testing.T) {
	hits := NewRuleHits()

	evaluation := Evaluate("quux", &hits, true)

	if evaluation.HardReject {
		t.Fatalf("expected soft-rule evaluation, got hard reject %q", evaluation.HardRule)
	}
	if len(evaluation.SoftRules) != 3 {
		t.Fatalf("SoftRules length = %d, want 3", len(evaluation.SoftRules))
	}
	if evaluation.SoftRules[0].Name != "uncommon_sequence" {
		t.Fatalf("first soft rule = %q, want %q", evaluation.SoftRules[0].Name, "uncommon_sequence")
	}
	if evaluation.SoftRules[1].Name != "rare_letter_density" {
		t.Fatalf("second soft rule = %q, want %q", evaluation.SoftRules[1].Name, "rare_letter_density")
	}
	if evaluation.SoftRules[2].Name != "repeated_same_vowel_pair" {
		t.Fatalf("third soft rule = %q, want %q", evaluation.SoftRules[2].Name, "repeated_same_vowel_pair")
	}
	if hits.Soft["uncommon_sequence"] != 1 {
		t.Fatalf("uncommon_sequence hits = %d, want 1", hits.Soft["uncommon_sequence"])
	}
	if hits.Soft["rare_letter_density"] != 1 {
		t.Fatalf("rare_letter_density hits = %d, want 1", hits.Soft["rare_letter_density"])
	}
	if hits.Soft["repeated_same_vowel_pair"] != 1 {
		t.Fatalf("repeated_same_vowel_pair hits = %d, want 1", hits.Soft["repeated_same_vowel_pair"])
	}

	expectedScore := defaults.BaseScore - 25 - 20 - 10 + evaluation.BigramAdjustment
	if evaluation.Score != expectedScore {
		t.Fatalf("Score = %d, want %d", evaluation.Score, expectedScore)
	}
}

func TestIllegalConsonantAdjacencyRespectsAllowLists(t *testing.T) {
	if IllegalConsonantAdjacency("blar") {
		t.Fatalf("expected allowed adjacency for %q", "blar")
	}
	if !IllegalConsonantAdjacency("bdar") {
		t.Fatalf("expected disallowed adjacency for %q", "bdar")
	}
}
