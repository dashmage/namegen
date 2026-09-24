package gen

import (
	"hash/crc32"
	"testing"

	"github.com/dashmage/namegen/internal/data"
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

func TestAvgLogProbNormalizesWithoutChangingTransitions(t *testing.T) {
	model := NewBigramModel(defaults.BaseAlpha)
	model.Train([]string{"lora"})

	want := model.AvgLogProb("lora")
	if got := model.AvgLogProb("Lo-ra_123!"); got != want {
		t.Fatalf("AvgLogProb(normalized variant) = %f, want %f", got, want)
	}
}

func TestDefaultBigramModelScoresHeldOutCorpusAboveGeneratedNames(t *testing.T) {
	words, err := data.LoadWords()
	if err != nil {
		t.Fatalf("LoadWords() error = %v", err)
	}

	train := make([]string, 0, len(words)*4/5)
	heldout := make([]string, 0, len(words)/5)
	// A content-based split is deterministic and keeps duplicate spellings together.
	for _, word := range words {
		if crc32.ChecksumIEEE([]byte(word))%5 == 0 {
			heldout = append(heldout, word)
		} else {
			train = append(train, word)
		}
	}
	if len(train) == 0 || len(heldout) == 0 {
		t.Fatalf("corpus split produced train=%d and heldout=%d words", len(train), len(heldout))
	}

	model := NewBigramModel(defaults.BaseAlpha)
	model.Train(train)

	SetSeed(42)
	heldoutTotal := 0.0
	generatedTotal := 0.0
	for _, word := range heldout {
		heldoutTotal += model.AvgLogProb(word)
		generatedTotal += model.AvgLogProb(RandomName(len(word)))
	}

	heldoutMean := heldoutTotal / float64(len(heldout))
	generatedMean := generatedTotal / float64(len(heldout))
	if heldoutMean <= generatedMean {
		t.Fatalf("held-out corpus mean log-probability = %.4f, generated-name mean = %.4f; want held-out names to score higher", heldoutMean, generatedMean)
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
