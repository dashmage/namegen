package main

import (
	"path/filepath"
	"testing"

	"github.com/dashmage/namegen/internal/data"
	"github.com/dashmage/namegen/internal/gen"
)

func TestFilterModelCompatibleNames(t *testing.T) {
	input := []string{"lora", "palo", "bcd", "kfc", "acmeq", "aaa"}
	got := filterModelCompatibleNames(input)
	want := []string{"lora", "palo"}
	if len(got) != len(want) {
		t.Fatalf("filtered names = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("filtered names = %v, want %v", got, want)
		}
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
