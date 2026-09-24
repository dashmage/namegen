package gen

import (
	"hash/crc32"
	"math"
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

func TestInterpolatedTrigramBacksOffForUnseenContext(t *testing.T) {
	model := NewInterpolatedTrigramModel(defaults.BaseAlpha)
	model.Train([]string{"lora", "lena"})

	got := model.InterpolatedLogProb('x', 'z', 'q')
	want := model.bigram.LogProb('z', 'q')
	if math.Abs(got-want) > 1e-12 {
		t.Fatalf("unseen-context interpolated log probability = %.12f, want bigram backoff %.12f", got, want)
	}
}

func TestInterpolatedTrigramUsesTwoCharacterContext(t *testing.T) {
	words := []string{"shana", "shana", "shana", "shana", "shana", "thena", "thena", "thena", "thena", "thena"}
	model := NewInterpolatedTrigramModel(defaults.BaseAlpha)
	model.Train(words)

	trigramProbability := math.Exp(model.InterpolatedLogProb('s', 'h', 'a'))
	bigramProbability := math.Exp(model.bigram.LogProb('h', 'a'))
	if trigramProbability <= bigramProbability {
		t.Fatalf("P(a|sh) = %.4f, P(a|h) = %.4f; want context-specific probability to be higher", trigramProbability, bigramProbability)
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

func TestBigramAndInterpolatedTrigramHeldOutComparison(t *testing.T) {
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
	trigramModel := NewInterpolatedTrigramModel(defaults.BaseAlpha)
	trigramModel.Train(train)

	SetSeed(42)
	bigramHeldoutTotal := 0.0
	bigramGeneratedTotal := 0.0
	trigramHeldoutTotal := 0.0
	trigramGeneratedTotal := 0.0
	for _, word := range heldout {
		generated := RandomName(len(word))
		bigramHeldoutTotal += model.AvgLogProb(word)
		bigramGeneratedTotal += model.AvgLogProb(generated)
		trigramHeldoutTotal += trigramModel.AvgLogProb(word)
		trigramGeneratedTotal += trigramModel.AvgLogProb(generated)
	}

	bigramHeldoutMean := bigramHeldoutTotal / float64(len(heldout))
	bigramGeneratedMean := bigramGeneratedTotal / float64(len(heldout))
	trigramHeldoutMean := trigramHeldoutTotal / float64(len(heldout))
	trigramGeneratedMean := trigramGeneratedTotal / float64(len(heldout))
	bigramGap := bigramHeldoutMean - bigramGeneratedMean
	trigramGap := trigramHeldoutMean - trigramGeneratedMean
	t.Logf("held-out vs generated mean log-probability gap: bigram=%.4f interpolated-trigram=%.4f", bigramGap, trigramGap)

	if bigramGap <= 0 {
		t.Fatalf("bigram held-out mean = %.4f, generated-name mean = %.4f; want held-out names to score higher", bigramHeldoutMean, bigramGeneratedMean)
	}
	if trigramGap <= 0 {
		t.Fatalf("interpolated trigram held-out mean = %.4f, generated-name mean = %.4f; want held-out names to score higher", trigramHeldoutMean, trigramGeneratedMean)
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
