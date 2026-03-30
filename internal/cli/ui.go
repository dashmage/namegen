package cli

import (
	"fmt"
	"math"
	"strings"

	"github.com/dashmage/namegen/internal/gen"
)

func PrintAcceptedWord(candidate gen.ScoredWord, verbose bool) {
	if verbose {
		fmt.Printf("%s (score=%d, bigram_prob=%s)\n", candidate.Word, candidate.Score, candidate.BigramProb)
		return
	}
	fmt.Println(candidate.Word)
}

func PrintRunResult(result gen.RunResult, debug, tune bool, runSeed int64, seedSet bool) {
	verbose := debug || tune
	for _, candidate := range result.Words {
		PrintAcceptedWord(candidate, verbose)
	}

	if tune {
		PrintTuneReport(result, runSeed, seedSet)
		return
	}

	if !debug {
		return
	}

	PrintDebugSummary(result.Stats, runSeed, seedSet)
}

func PrintTuneReport(result gen.RunResult, runSeed int64, seedSet bool) {
	PrintDebugSummary(result.Stats, runSeed, seedSet)

	fmt.Println()
	fmt.Println("Tune entries (all attempts)")
	if len(result.TuneEntries) == 0 {
		fmt.Println("- none")
		return
	}

	for i, entry := range result.TuneEntries {
		decision := "accepted"
		if !entry.Accepted {
			decision = "rejected:" + entry.RejectReason
			if entry.HardRule != "" {
				decision = decision + "(" + entry.HardRule + ")"
			}
		}

		fmt.Printf("\n- #%d: %s\n", i+1, strings.ToUpper(entry.Word))
		fmt.Printf("  word=%s score=%d threshold=%d decision=%s\n", entry.Word, entry.Score, entry.Threshold, decision)
		fmt.Printf("  soft_rules=%s\n", formatSoftRules(entry.SoftRules))
		if math.IsNaN(entry.AvgLogProb) {
			fmt.Printf("  bigram=unavailable band=%s adjustment=%d\n", entry.BigramProb, entry.BigramAdjustment)
			continue
		}
		fmt.Printf("  bigram_avg_log_prob=%.4f band=%s adjustment=%d\n", entry.AvgLogProb, entry.BigramProb, entry.BigramAdjustment)
	}
}

func formatSoftRules(rules []gen.RulePenalty) string {
	if len(rules) == 0 {
		return "none"
	}

	parts := make([]string, 0, len(rules))
	for _, rule := range rules {
		parts = append(parts, fmt.Sprintf("%s(-%d)", rule.Name, rule.Penalty))
	}

	return strings.Join(parts, ",")
}

func PrintDebugSummary(summary gen.GenStats, runSeed int64, seedSet bool) {
	fmt.Printf("\nDebug summary\n")
	fmt.Printf("- attempts: %d\n", summary.Attempts)
	fmt.Printf("- accepted: %d\n", summary.Accepted)
	fmt.Printf("- hard rejects: %d\n", summary.HardRejects)
	fmt.Printf("- low-score rejects: %d\n", summary.LowScoreRejects)
	fmt.Printf("- threshold: %d\n", summary.Threshold)
	if seedSet {
		fmt.Printf("- seed: %d (provided)\n", runSeed)
	} else {
		fmt.Printf("- seed: %d (auto-generated)\n", runSeed)
	}

	fmt.Println()
	printHardRuleHits(summary.HardRuleStats())
	fmt.Println()
	printSoftRuleHits(summary.SoftRuleStats())
}

func printHardRuleHits(stats []gen.RuleStat) {
	fmt.Println("Hard rule hits (instant rejection)")
	if len(stats) == 0 {
		fmt.Println("- none")
		return
	}

	for _, stat := range stats {
		fmt.Printf("- %s x%d: %s\n", stat.Name, stat.Hits, stat.Description)
	}
}

func printSoftRuleHits(stats []gen.RuleStat) {
	fmt.Println("Soft rule hits (score penalized)")
	if len(stats) == 0 {
		fmt.Println("- none")
		return
	}

	for _, stat := range stats {
		fmt.Printf("- %s x%d: %s (penalty=%d)\n", stat.Name, stat.Hits, stat.Description, stat.Penalty)
	}
}
