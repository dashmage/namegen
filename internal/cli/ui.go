package cli

import (
	"fmt"

	"github.com/dashmage/namegen/internal/gen"
)

func PrintAcceptedWord(candidate gen.ScoredWord, debug bool) {
	if debug {
		fmt.Printf("%s (score=%d, bigram_prob=%s)\n", candidate.Word, candidate.Score, candidate.BigramProb)
		return
	}
	fmt.Println(candidate.Word)
}

func PrintRunResult(result gen.RunResult, debug bool, runSeed int64, seedSet bool) {
	for _, candidate := range result.Words {
		PrintAcceptedWord(candidate, debug)
	}

	if !debug {
		return
	}

	PrintDebugSummary(result.Stats, runSeed, seedSet)
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
