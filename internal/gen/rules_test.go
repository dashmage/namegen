package gen

import (
	"testing"

	"github.com/dashmage/namegen/internal/defaults"
)

func TestUncommonSequencesAreReachableSoftRules(t *testing.T) {
	for _, sequence := range UncommonSequences {
		word := "a" + sequence + "a"
		t.Run(sequence, func(t *testing.T) {
			for _, rule := range HardRules {
				if rule.Check(word) {
					t.Fatalf("sequence %q is unreachable because hard rule %q rejects %q", sequence, rule.Name, word)
				}
			}
			if !UncommonSequence(word) {
				t.Fatalf("UncommonSequence(%q) = false, want true", word)
			}
		})
	}
}

func TestUncommonSequenceDoesNotDuplicateDedicatedSoftRules(t *testing.T) {
	for _, word := range []string{"aiia", "auua", "aiqa", "auqa"} {
		if UncommonSequence(word) {
			t.Errorf("UncommonSequence(%q) = true, want false", word)
		}
	}

	if !RepeatedSameVowelPair("aiia") || !RepeatedSameVowelPair("auua") {
		t.Fatal("expected dedicated repeated-vowel rule to catch ii and uu")
	}
	if !QWithoutU("aiqa") || !QWithoutU("auqa") {
		t.Fatal("expected dedicated q rule to catch iq and uq")
	}
}

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

func TestConsonantPoolMatchesVowelClassifier(t *testing.T) {
	for i := 0; i < len(defaults.Consonants); i++ {
		if isVowel(defaults.Consonants[i]) {
			t.Errorf("consonant pool contains vowel %q", defaults.Consonants[i])
		}
	}

	if !isVowel('y') {
		t.Fatal("y should remain classified as a vowel")
	}
}

func TestIllegalConsonantAdjacencyUsesExplicitNextAllowLists(t *testing.T) {
	tests := []struct {
		name string
		word string
		want bool
	}{
		{name: "common st cluster is not restricted by missing s entry", word: "sta", want: false},
		{name: "unlisted m has no explicit restriction", word: "mta", want: false},
		{name: "listed b rejects disallowed m", word: "bma", want: true},
		{name: "listed b permits l", word: "bla", want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := IllegalConsonantAdjacency(test.word); got != test.want {
				t.Fatalf("IllegalConsonantAdjacency(%q) = %t, want %t", test.word, got, test.want)
			}
		})
	}
}

func TestEvaluateCapturesSoftPenaltiesAndDetails(t *testing.T) {
	hits := NewRuleHits()

	evaluation := Evaluate("quux", &hits, true)

	if evaluation.HardReject {
		t.Fatalf("expected soft-rule evaluation, got hard reject %q", evaluation.HardRule)
	}
	if len(evaluation.SoftRules) != 2 {
		t.Fatalf("SoftRules length = %d, want 2", len(evaluation.SoftRules))
	}
	if evaluation.SoftRules[0].Name != "rare_letter_density" {
		t.Fatalf("first soft rule = %q, want %q", evaluation.SoftRules[0].Name, "rare_letter_density")
	}
	if evaluation.SoftRules[1].Name != "repeated_same_vowel_pair" {
		t.Fatalf("second soft rule = %q, want %q", evaluation.SoftRules[1].Name, "repeated_same_vowel_pair")
	}
	if hits.Soft["uncommon_sequence"] != 0 {
		t.Fatalf("uncommon_sequence hits = %d, want 0", hits.Soft["uncommon_sequence"])
	}
	if hits.Soft["rare_letter_density"] != 1 {
		t.Fatalf("rare_letter_density hits = %d, want 1", hits.Soft["rare_letter_density"])
	}
	if hits.Soft["repeated_same_vowel_pair"] != 1 {
		t.Fatalf("repeated_same_vowel_pair hits = %d, want 1", hits.Soft["repeated_same_vowel_pair"])
	}
	if evaluation.Score <= evaluation.BigramAdjustment {
		t.Fatalf("Score = %d, want score to remain above bigram adjustment alone", evaluation.Score)
	}
	if evaluation.Score >= 100+evaluation.BigramAdjustment {
		t.Fatalf("Score = %d, want soft rules to reduce score below base-plus-bigram", evaluation.Score)
	}
}
