package main

import (
	"path/filepath"
	"testing"

	"github.com/dashmage/namegen/internal/data"
	"github.com/dashmage/namegen/internal/gen"
)

func TestHardRuleAuditCountsOverlappingHits(t *testing.T) {
	input := []string{"lora", "palo", "bcd", "kfc", "acmeq", "aaa"}
	got, audit := gen.FilterHardRuleValidNames(input, 2)
	want := []string{"lora", "palo"}
	if len(got) != len(want) {
		t.Fatalf("filtered names = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("filtered names = %v, want %v", got, want)
		}
	}
	if audit.CandidateCount != 6 || audit.AcceptedCount != 2 || audit.RejectedCount != 4 {
		t.Fatalf("audit totals = %+v, want 6 candidates, 2 accepted, 4 rejected", audit)
	}
	hits := make(map[string]int, len(audit.Rules))
	for _, rule := range audit.Rules {
		hits[rule.Name] = rule.Hits
		if len(rule.Examples) > 2 {
			t.Fatalf("rule %q example count = %d, want max 2", rule.Name, len(rule.Examples))
		}
	}
	if hits["three_consecutive_consonants"] != 2 || hits["illegal_consonant_adjacency"] != 3 || hits["missing_core_vowel"] != 2 {
		t.Fatalf("independent rule hits = %v, want overlapping counts", hits)
	}
}

func TestCommittedCorpusPassesGeneratorHardRules(t *testing.T) {
	path := filepath.Join("..", "..", "internal", "data", "corpora", "wikidata_company_brand.txt")
	words, err := data.LoadWordsFromFile(path)
	if err != nil {
		t.Fatalf("LoadWordsFromFile() error = %v", err)
	}
	if len(words) < 1000 {
		t.Fatalf("cleaned corpus has %d names, want at least 1000", len(words))
	}

	for _, word := range words {
		for _, rule := range gen.HardRules {
			if rule.Check(word) {
				t.Errorf("corpus name %q fails hard rule %q", word, rule.Name)
				break
			}
		}
	}
}
