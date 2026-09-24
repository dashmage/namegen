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

func TestScoreAdjustmentInterpolatesAndClamps(t *testing.T) {
	tests := []struct {
		name  string
		value float64
		want  int
	}{
		{name: "below minimum", value: defaults.VeryLowProbCutoff - 1, want: -defaults.VeryLowProbPenalty},
		{name: "very-low anchor", value: defaults.VeryLowProbCutoff, want: -defaults.VeryLowProbPenalty},
		{name: "between very-low and low anchors", value: (defaults.VeryLowProbCutoff + defaults.LowProbCutoff) / 2, want: -25},
		{name: "low anchor", value: defaults.LowProbCutoff, want: -defaults.LowProbPenalty},
		{name: "between low and mid anchors", value: (defaults.LowProbCutoff + defaults.MidProbCutoff) / 2, want: -15},
		{name: "mid anchor", value: defaults.MidProbCutoff, want: -defaults.MidProbPenalty},
		{name: "between mid and bonus anchors", value: (defaults.MidProbCutoff + defaults.GoodProbBonusCutoff) / 2, want: -3},
		{name: "bonus anchor", value: defaults.GoodProbBonusCutoff, want: defaults.GoodProbBonus},
		{name: "above maximum", value: 0, want: defaults.GoodProbBonus},
	}

	previous := -defaults.VeryLowProbPenalty - 1
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := scoreAdjustmentFor(test.value)
			if got != test.want {
				t.Fatalf("scoreAdjustmentFor(%f) = %d, want %d", test.value, got, test.want)
			}
			if got < previous {
				t.Fatalf("score adjustment decreased from %d to %d as probability increased", previous, got)
			}
		})
		previous = test.want
	}
}

func TestBigramModelPrefersSeenTransitions(t *testing.T) {
	model := NewBigramModel(defaults.BaseAlpha)
	model.Train([]string{"lena", "lora", "nora", "mila", "mira", "sora"})

	goodWord := "lora"
	badWord := "zxzx"

	goodScore := model.AvgLogProb(goodWord)
	badScore := model.AvgLogProb(badWord)
	band, avgLogProb := model.ScoreAdjustment(goodWord)

	if band.Value != scoreAdjustmentFor(avgLogProb) {
		t.Fatalf("ScoreAdjustment value = %d, want %d", band.Value, scoreAdjustmentFor(avgLogProb))
	}
	if !(goodScore > badScore) {
		t.Fatalf("AvgLogProb(%q) = %f, AvgLogProb(%q) = %f, want seen word to score higher", goodWord, goodScore, badWord, badScore)
	}
}
