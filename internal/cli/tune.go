package cli

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/dashmage/namegen/internal/defaults"
	"github.com/dashmage/namegen/internal/gen"
)

type tuneObservation struct {
	Word       string
	Rating     int
	Evaluation gen.Evaluation
}

type tuneRecommendation struct {
	Weight  int
	Message string
}

type softRuleSignal struct {
	Rule  gen.Rule
	Score int
	Count int
}

type bandSignal struct {
	BandName string
	Score    int
	Count    int
}

const (
	tuneSignalsFormat         = "Signals: hard_rule=%s soft_rules=%s probability_band=%s\n"
	tuneSuggestionsHeader     = "Manual tuning suggestions after %d rating(s)\n"
	tuneNoStrongSignalMessage = "- No strong signal yet. Keep rating more words."

	tunePenaltyTooHighFormat = "%s penalty may be too high; consider lowering from %d to %d"
	tunePenaltyTooLowFormat  = "%s penalty may be too low; consider raising from %d to %d"

	tuneVeryLowHarshFormat   = "very-low probability scoring looks too harsh; consider lowering VeryLowProbCutoff from %.1f to %.1f or lowering VeryLowProbPenalty from %d to %d"
	tuneVeryLowLenientFormat = "very-low probability scoring may be too lenient; consider raising VeryLowProbCutoff from %.1f to %.1f or raising VeryLowProbPenalty from %d to %d"
	tuneLowHarshFormat       = "low probability scoring looks too harsh; consider lowering LowProbCutoff from %.1f to %.1f or lowering LowProbPenalty from %d to %d"
	tuneLowLenientFormat     = "low probability scoring may be too lenient; consider raising LowProbCutoff from %.1f to %.1f or raising LowProbPenalty from %d to %d"
	tuneMidHarshFormat       = "mid probability scoring looks too harsh; consider lowering MidProbCutoff from %.1f to %.1f or lowering MidProbPenalty from %d to %d"
	tuneMidLenientFormat     = "mid probability scoring may be too lenient; consider raising MidProbCutoff from %.1f to %.1f or raising MidProbPenalty from %d to %d"
	tuneGoodAlignedFormat    = "good probability bonus lines up with your ratings so far; keep GoodProbBonus at %d unless later feedback shifts"
	tuneGoodWeakFormat       = "good probability words are landing weakly; consider lowering GoodProbBonus from %d to %d or raising MidProbCutoff from %.1f to %.1f"
)

// RunTuneSession starts the interactive tuning loop.
func RunTuneSession(length int) {
	reader := bufio.NewReader(os.Stdin)
	observations := make([]tuneObservation, 0, 16)

	fmt.Println("Tune mode")
	fmt.Println("Rate each generated word from 1-5, or press q to quit.")
	fmt.Println("1=very bad 2=bad 3=ok 4=good 5=very good")

	for {
		word := gen.RandomWord(length)
		evaluation := gen.Evaluate(word, nil, true)

		fmt.Println()
		fmt.Printf("Word: %s\n", word)

		rating, quit, err := readTuneRating(reader)
		if err != nil {
			fmt.Fprintf(os.Stderr, "tune input error: %v\n", err)
			return
		}
		if quit {
			fmt.Println()
			fmt.Println("Exiting tune mode.")
			if len(observations) > 0 {
				printTuneRecommendations(observations)
			}
			return
		}

		observations = append(observations, tuneObservation{
			Word:       word,
			Rating:     rating,
			Evaluation: evaluation,
		})

		printTuneFeedback(word, rating, evaluation, observations)
	}
}

func readTuneRating(reader *bufio.Reader) (rating int, quit bool, err error) {
	for {
		fmt.Print("Rating [1-5, q]: ")
		line, readErr := reader.ReadString('\n')
		if readErr != nil {
			return 0, false, readErr
		}

		input := strings.TrimSpace(strings.ToLower(line))
		if input == "q" {
			return 0, true, nil
		}

		rating, convErr := strconv.Atoi(input)
		if convErr == nil && rating >= 1 && rating <= 5 {
			return rating, false, nil
		}

		fmt.Println("Enter 1, 2, 3, 4, 5, or q.")
	}
}

func printTuneFeedback(word string, rating int, evaluation gen.Evaluation, observations []tuneObservation) {
	fmt.Printf("Recorded %q as %s.\n", word, tuneLabel(rating))
	fmt.Printf(tuneSignalsFormat, formatHardRule(evaluation.HardRule), formatSoftRules(evaluation.SoftRules), evaluation.ProbabilityBand.Name)
	printTuneRecommendations(observations)
}

func printTuneRecommendations(observations []tuneObservation) {
	fmt.Println()
	fmt.Printf(tuneSuggestionsHeader, len(observations))

	recommendations := collectTuneRecommendations(observations)
	if len(recommendations) == 0 {
		fmt.Println(tuneNoStrongSignalMessage)
		return
	}

	for _, recommendation := range recommendations {
		fmt.Printf("- %s\n", recommendation.Message)
	}
}

func collectTuneRecommendations(observations []tuneObservation) []tuneRecommendation {
	softSignals := summarizeSoftRuleSignals(observations)
	bandSignals := summarizeBandSignals(observations)
	recommendations := make([]tuneRecommendation, 0, len(softSignals)+len(bandSignals))

	for _, signal := range softSignals {
		if signal.Score == 0 {
			continue
		}

		step := penaltyStep(signal.Score)
		if signal.Score > 0 {
			updated := max(0, signal.Rule.Penalty-step)
			recommendations = append(recommendations, tuneRecommendation{
				Weight:  abs(signal.Score),
				Message: fmt.Sprintf(tunePenaltyTooHighFormat, signal.Rule.Name, signal.Rule.Penalty, updated),
			})
			continue
		}

		updated := signal.Rule.Penalty + step
		recommendations = append(recommendations, tuneRecommendation{
			Weight:  abs(signal.Score),
			Message: fmt.Sprintf(tunePenaltyTooLowFormat, signal.Rule.Name, signal.Rule.Penalty, updated),
		})
	}

	for _, signal := range bandSignals {
		if signal.Score == 0 {
			continue
		}

		recommendations = append(recommendations, recommendationForBand(signal))
	}

	sort.SliceStable(recommendations, func(i, j int) bool {
		if recommendations[i].Weight == recommendations[j].Weight {
			return recommendations[i].Message < recommendations[j].Message
		}
		return recommendations[i].Weight > recommendations[j].Weight
	})

	if len(recommendations) > 5 {
		return recommendations[:5]
	}

	return recommendations
}

func summarizeSoftRuleSignals(observations []tuneObservation) []softRuleSignal {
	signals := make(map[string]*softRuleSignal, len(gen.SoftRules))
	for _, rule := range gen.SoftRules {
		signals[rule.Name] = &softRuleSignal{Rule: rule}
	}

	for _, observation := range observations {
		signal := ratingSignal(observation.Rating)
		for _, rule := range observation.Evaluation.SoftRules {
			entry, ok := signals[rule.Name]
			if !ok {
				continue
			}
			entry.Score += signal
			entry.Count++
		}
	}

	out := make([]softRuleSignal, 0, len(signals))
	for _, signal := range signals {
		if signal.Count == 0 {
			continue
		}
		out = append(out, *signal)
	}

	sort.SliceStable(out, func(i, j int) bool {
		if abs(out[i].Score) == abs(out[j].Score) {
			return out[i].Rule.Name < out[j].Rule.Name
		}
		return abs(out[i].Score) > abs(out[j].Score)
	})

	return out
}

func summarizeBandSignals(observations []tuneObservation) []bandSignal {
	signals := map[string]*bandSignal{
		"vlow": {BandName: "vlow"},
		"low":  {BandName: "low"},
		"mid":  {BandName: "mid"},
		"good": {BandName: "good"},
	}

	for _, observation := range observations {
		entry, ok := signals[observation.Evaluation.ProbabilityBand.Name]
		if !ok {
			continue
		}
		entry.Score += ratingSignal(observation.Rating)
		entry.Count++
	}

	out := make([]bandSignal, 0, len(signals))
	for _, signal := range signals {
		if signal.Count == 0 {
			continue
		}
		out = append(out, *signal)
	}

	sort.SliceStable(out, func(i, j int) bool {
		if abs(out[i].Score) == abs(out[j].Score) {
			return out[i].BandName < out[j].BandName
		}
		return abs(out[i].Score) > abs(out[j].Score)
	})

	return out
}

func recommendationForBand(signal bandSignal) tuneRecommendation {
	step := cutoffStep(signal.Score)
	weight := abs(signal.Score)

	switch signal.BandName {
	case "vlow":
		if signal.Score > 0 {
			return tuneRecommendation{
				Weight:  weight,
				Message: fmt.Sprintf(tuneVeryLowHarshFormat, defaults.VeryLowProbCutoff, defaults.VeryLowProbCutoff-step, defaults.VeryLowProbPenalty, max(0, defaults.VeryLowProbPenalty-5)),
			}
		}
		return tuneRecommendation{
			Weight:  weight,
			Message: fmt.Sprintf(tuneVeryLowLenientFormat, defaults.VeryLowProbCutoff, defaults.VeryLowProbCutoff+step, defaults.VeryLowProbPenalty, defaults.VeryLowProbPenalty+5),
		}
	case "low":
		if signal.Score > 0 {
			return tuneRecommendation{
				Weight:  weight,
				Message: fmt.Sprintf(tuneLowHarshFormat, defaults.LowProbCutoff, defaults.LowProbCutoff-step, defaults.LowProbPenalty, max(0, defaults.LowProbPenalty-5)),
			}
		}
		return tuneRecommendation{
			Weight:  weight,
			Message: fmt.Sprintf(tuneLowLenientFormat, defaults.LowProbCutoff, defaults.LowProbCutoff+step, defaults.LowProbPenalty, defaults.LowProbPenalty+5),
		}
	case "mid":
		if signal.Score > 0 {
			return tuneRecommendation{
				Weight:  weight,
				Message: fmt.Sprintf(tuneMidHarshFormat, defaults.MidProbCutoff, defaults.MidProbCutoff-step, defaults.MidProbPenalty, max(0, defaults.MidProbPenalty-5)),
			}
		}
		return tuneRecommendation{
			Weight:  weight,
			Message: fmt.Sprintf(tuneMidLenientFormat, defaults.MidProbCutoff, defaults.MidProbCutoff+step, defaults.MidProbPenalty, defaults.MidProbPenalty+5),
		}
	default:
		if signal.Score > 0 {
			return tuneRecommendation{
				Weight:  weight,
				Message: fmt.Sprintf(tuneGoodAlignedFormat, defaults.GoodProbBonus),
			}
		}
		return tuneRecommendation{
			Weight:  weight,
			Message: fmt.Sprintf(tuneGoodWeakFormat, defaults.GoodProbBonus, max(0, defaults.GoodProbBonus-5), defaults.MidProbCutoff, defaults.MidProbCutoff+step),
		}
	}
}

func ratingSignal(rating int) int {
	return rating - 3
}

func penaltyStep(score int) int {
	if abs(score) >= 4 {
		return 10
	}
	return 5
}

func cutoffStep(score int) float64 {
	if abs(score) >= 4 {
		return 0.2
	}
	return 0.1
}

func tuneLabel(rating int) string {
	switch rating {
	case 1:
		return "very bad"
	case 2:
		return "bad"
	case 3:
		return "ok"
	case 4:
		return "good"
	case 5:
		return "very good"
	default:
		return "unknown"
	}
}

func formatHardRule(name string) string {
	if name == "" {
		return "none"
	}
	return name
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// func max(a, b int) int {
// 	if a > b {
// 		return a
// 	}
// 	return b
// }
