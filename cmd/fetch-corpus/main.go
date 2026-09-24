package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/dashmage/namegen/internal/data"
	"github.com/dashmage/namegen/internal/gen"
)

func main() {
	limit := flag.Int("limit", 20000, "maximum Wikidata labels to query")
	output := flag.String("output", "internal/data/corpora/wikidata_company_brand.txt", "output corpus file")
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	client := &http.Client{Timeout: 2 * time.Minute}
	names, err := data.FetchWikidataCompanyBrandNames(ctx, client, data.WikidataSPARQLEndpoint, *limit)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fetch Wikidata corpus: %v\n", err)
		os.Exit(1)
	}

	normalizedCount := len(names)
	names = filterModelCompatibleNames(names)

	if err := os.MkdirAll(filepath.Dir(*output), 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "create output directory: %v\n", err)
		os.Exit(1)
	}
	file, err := os.Create(*output)
	if err != nil {
		fmt.Fprintf(os.Stderr, "create corpus file: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	fmt.Fprintln(file, "# English company and brand labels from Wikidata")
	fmt.Fprintln(file, "# Wikidata data license: CC0 1.0 Universal")
	fmt.Fprintf(file, "# Retrieved: %s\n", time.Now().UTC().Format(time.RFC3339))
	fmt.Fprintf(file, "# Requested label limit: %d\n", *limit)
	fmt.Fprintf(file, "# Normalized unique labels before pronunciation filtering: %d\n", normalizedCount)
	fmt.Fprintf(file, "# Names passing generator hard rules: %d\n", len(names))
	for _, name := range names {
		if _, err := fmt.Fprintln(file, name); err != nil {
			fmt.Fprintf(os.Stderr, "write corpus file: %v\n", err)
			os.Exit(1)
		}
	}
	if err := file.Close(); err != nil {
		fmt.Fprintf(os.Stderr, "close corpus file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("wrote %d pronunciation-compatible names to %s (from %d normalized labels)\n", len(names), *output, normalizedCount)
}

func filterModelCompatibleNames(names []string) []string {
	filtered := make([]string, 0, len(names))
	for _, name := range names {
		valid := true
		for _, rule := range gen.HardRules {
			if rule.Check(name) {
				valid = false
				break
			}
		}
		if valid {
			filtered = append(filtered, name)
		}
	}
	return filtered
}
