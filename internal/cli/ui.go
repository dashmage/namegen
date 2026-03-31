package cli

import (
	"fmt"
	"math"
	"strings"

	"github.com/dashmage/namegen/internal/gen"
)

// PrintAcceptedWord prints an accepted candidate in concise or verbose form.
func PrintAcceptedWord(candidate gen.ScoredWord, verbose bool) {
	if verbose {
		fmt.Printf("%s (score=%d, probability_band=%s)\n", candidate.Word, candidate.Score, candidate.ProbabilityBand.Name)
		return
	}
	fmt.Println(candidate.Word)
}

// PrintRunResult prints accepted words and optional debug or tuning output.
func PrintRunResult(result gen.RunResult, debug, tune bool, seed int64, userSeed bool) {
	verbose := debug || tune
	for _, candidate := range result.Words {
		PrintAcceptedWord(candidate, verbose)
	}

	if tune {
		PrintTuneReport(result, seed, userSeed)
		return
	}

	if !debug {
		return
	}

	PrintDebugSummary(result.Stats, seed, userSeed)
}

// PrintTuneReport prints debug stats and per-attempt tuning diagnostics.
func PrintTuneReport(result gen.RunResult, seed int64, userSeed bool) {
	PrintDebugSummary(result.Stats, seed, userSeed)

	fmt.Println()
	fmt.Println("Generation attempts (all attempts)")
	if len(result.GenAttempts) == 0 {
		fmt.Println("- none")
		return
	}

	for i, entry := range result.GenAttempts {
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
			fmt.Printf("  bigram=unavailable band=%s adjustment=%d\n", entry.ProbabilityBand.Name, entry.BigramAdjustment)
			continue
		}
		fmt.Printf("  bigram_avg_log_prob=%.4f band=%s adjustment=%d\n", entry.AvgLogProb, entry.ProbabilityBand.Name, entry.BigramAdjustment)
	}
}

// formatSoftRules formats soft-rule penalties as comma-separated labels.
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

// PrintDebugSummary prints aggregate generation counts and rule hit summaries.
func PrintDebugSummary(summary gen.GenStats, seed int64, userSeed bool) {
	fmt.Printf("\nDebug summary\n")
	fmt.Printf("- attempts: %d\n", summary.Attempts)
	fmt.Printf("- accepted: %d\n", summary.Accepted)
	fmt.Printf("- hard rejects: %d\n", summary.HardRejects)
	fmt.Printf("- low-score rejects: %d\n", summary.LowScoreRejects)
	fmt.Printf("- threshold: %d\n", summary.Threshold)
	if userSeed {
		fmt.Printf("- seed: %d (provided)\n", seed)
	} else {
		fmt.Printf("- seed: %d (auto-generated)\n", seed)
	}

	fmt.Println()
	printHardRuleHits(summary.HardRuleStats())
	fmt.Println()
	printSoftRuleHits(summary.SoftRuleStats())
}

// printHardRuleHits prints hard-rule hit counts and descriptions.
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

// printSoftRuleHits prints soft-rule hit counts, descriptions, and penalties.
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
