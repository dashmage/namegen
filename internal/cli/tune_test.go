package cli

import (
	"strings"
	"testing"

	"github.com/dashmage/namegen/internal/gen"
)

func TestTuneSignalsCompareAgainstControlRatings(t *testing.T) {
	observations := make([]tuneObservation, 0, 10)
	for range 5 {
		observations = append(observations, tuneObservation{
			Rating: 4,
			Evaluation: gen.Evaluation{
				SoftRules: []gen.Rule{{Name: "uncommon_sequence"}},
			},
		})
	}
	for range 5 {
		observations = append(observations, tuneObservation{Rating: 3})
	}

	signals := summarizeSoftRuleSignals(observations)
	if len(signals) != 1 {
		t.Fatalf("soft-rule signals = %d, want 1", len(signals))
	}
	if signals[0].Rule.Name != "uncommon_sequence" || signals[0].Score != 5 || signals[0].Count != 5 {
		t.Fatalf("signal = %+v, want uncommon_sequence with score 5 and count 5", signals[0])
	}

	recommendations := collectTuneRecommendations(observations)
	if len(recommendations) != 1 || !strings.Contains(recommendations[0].Message, "uncommon_sequence penalty may be too high") {
		t.Fatalf("recommendations = %+v, want penalty-lowering recommendation", recommendations)
	}
}

func TestTuneSignalsIgnoreOverallRatingBias(t *testing.T) {
	observations := make([]tuneObservation, 0, 10)
	for range 5 {
		observations = append(observations, tuneObservation{
			Rating: 4,
			Evaluation: gen.Evaluation{
				SoftRules: []gen.Rule{{Name: "uncommon_sequence"}},
			},
		})
	}
	for range 5 {
		observations = append(observations, tuneObservation{Rating: 4})
	}

	if signals := summarizeSoftRuleSignals(observations); len(signals) != 0 {
		t.Fatalf("signals = %+v, want none when group and control ratings are equal", signals)
	}
	if recommendations := collectTuneRecommendations(observations); len(recommendations) != 0 {
		t.Fatalf("recommendations = %+v, want none when all names are rated equally", recommendations)
	}
}

func TestTuneSignalsRequireMinimumGroupAndControlSamples(t *testing.T) {
	observations := make([]tuneObservation, 0, 9)
	for range 4 {
		observations = append(observations, tuneObservation{
			Rating: 5,
			Evaluation: gen.Evaluation{
				SoftRules: []gen.Rule{{Name: "uncommon_sequence"}},
			},
		})
	}
	for range 5 {
		observations = append(observations, tuneObservation{Rating: 1})
	}

	if signals := summarizeSoftRuleSignals(observations); len(signals) != 0 {
		t.Fatalf("signals = %+v, want none with fewer than %d group samples", signals, tuneMinimumGroupSamples)
	}
}

func TestBandSignalsCompareAgainstOtherBands(t *testing.T) {
	observations := make([]tuneObservation, 0, 10)
	for range 5 {
		observations = append(observations, tuneObservation{
			Rating: 4,
			Evaluation: gen.Evaluation{
				ProbabilityBand: gen.ProbabilityBand{Name: "low"},
			},
		})
	}
	for range 5 {
		observations = append(observations, tuneObservation{
			Rating: 3,
			Evaluation: gen.Evaluation{
				ProbabilityBand: gen.ProbabilityBand{Name: "good"},
			},
		})
	}

	signals := summarizeBandSignals(observations)
	if len(signals) != 2 {
		t.Fatalf("band signals = %d, want 2", len(signals))
	}
	for _, signal := range signals {
		want := 5
		if signal.BandName == "good" {
			want = -5
		}
		if signal.Score != want || signal.Count != 5 {
			t.Errorf("signal = %+v, want score %d and count 5", signal, want)
		}
	}
}
