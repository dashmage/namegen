package gen

import (
	"testing"

	"github.com/dashmage/namegen/internal/defaults"
)

func TestNormalizeWord(t *testing.T) {
	got := normalizeWord("Lo-ra_123!")
	if got != "lora" {
		t.Fatalf("normalizeWord() = %q, want %q", got, "lora")
	}
}

func TestBigramModelPrefersSeenTransitions(t *testing.T) {
	model := NewBigramModel(defaults.BaseAlpha)
	model.Train([]string{"lena", "lora", "nora", "mila", "mira", "sora"})

	goodWord := "lora"
	badWord := "zxzx"

	goodScore := model.AvgLogProb(goodWord)
	badScore := model.AvgLogProb(badWord)

	if !(goodScore > badScore) {
		t.Fatalf("AvgLogProb(%q) = %f, AvgLogProb(%q) = %f, want seen word to score higher", goodWord, goodScore, badWord, badScore)
	}
}
