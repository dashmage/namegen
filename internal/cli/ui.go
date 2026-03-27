package cli

import "fmt"

type RuleStat struct {
	Name        string
	Hits        int
	Penalty     int
	Description string
}

type DebugSummary struct {
	Attempts        int
	Accepted        int
	HardRejects     int
	LowScoreRejects int
	Threshold       int
	RunSeed         int64
	SeedSet         bool
	HardRuleHits    []RuleStat
	SoftRuleHits    []RuleStat
}

func PrintAcceptedWord(word string, score int, debug bool) {
	if debug {
		fmt.Printf("%s (%d)\n", word, score)
		return
	}
	fmt.Println(word)
}

func PrintDebugSummary(summary DebugSummary) {
	fmt.Printf("\nDebug summary\n")
	fmt.Printf("- attempts: %d\n", summary.Attempts)
	fmt.Printf("- accepted: %d\n", summary.Accepted)
	fmt.Printf("- hard rejects: %d\n", summary.HardRejects)
	fmt.Printf("- low-score rejects: %d\n", summary.LowScoreRejects)
	fmt.Printf("- threshold: %d\n", summary.Threshold)
	if summary.SeedSet {
		fmt.Printf("- seed: %d (provided)\n", summary.RunSeed)
	} else {
		fmt.Printf("- seed: %d (auto-generated)\n", summary.RunSeed)
	}

	fmt.Println()
	printHardRuleHits(summary.HardRuleHits)
	fmt.Println()
	printSoftRuleHits(summary.SoftRuleHits)
}

func printHardRuleHits(stats []RuleStat) {
	fmt.Println("Hard rule hits (instant rejection)")
	if len(stats) == 0 {
		fmt.Println("- none")
		return
	}

	for _, stat := range stats {
		fmt.Printf("- %s x%d: %s\n", stat.Name, stat.Hits, stat.Description)
	}
}

func printSoftRuleHits(stats []RuleStat) {
	fmt.Println("Soft rule hits (score penalized)")
	if len(stats) == 0 {
		fmt.Println("- none")
		return
	}

	for _, stat := range stats {
		fmt.Printf("- %s x%d: %s (penalty=%d)\n", stat.Name, stat.Hits, stat.Description, stat.Penalty)
	}
}
