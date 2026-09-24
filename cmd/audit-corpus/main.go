package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/dashmage/namegen/internal/data"
	"github.com/dashmage/namegen/internal/gen"
)

func main() {
	corpusPath := flag.String("corpus", "internal/data/corpora/wikidata_company_brand_raw.txt", "normalized corpus to audit")
	examples := flag.Int("examples", 10, "maximum examples to print per rule")
	flag.Parse()

	if *examples < 0 {
		fmt.Fprintln(os.Stderr, "--examples must be non-negative")
		os.Exit(2)
	}
	words, err := data.LoadWordsFromFile(*corpusPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load corpus: %v\n", err)
		os.Exit(1)
	}
	_, audit := gen.FilterHardRuleValidNames(words, *examples)

	fmt.Printf("Candidates: %d\n", audit.CandidateCount)
	fmt.Printf("Pass every hard rule: %d\n", audit.AcceptedCount)
	fmt.Printf("Rejected by one or more hard rules: %d\n", audit.RejectedCount)
	fmt.Println("Rule hits (overlapping counts; a name can hit multiple rules):")
	for _, rule := range audit.Rules {
		fmt.Printf("- %-34s %5d", rule.Name, rule.Hits)
		if len(rule.Examples) > 0 {
			fmt.Printf("  examples: %s", strings.Join(rule.Examples, ", "))
		}
		fmt.Println()
	}
}
