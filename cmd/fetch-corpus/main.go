package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dashmage/namegen/internal/data"
	"github.com/dashmage/namegen/internal/gen"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "fetch Wikidata corpus: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	limit := flag.Int("limit", 20000, "maximum Wikidata labels to query")
	output := flag.String("output", "internal/data/corpora/wikidata_company_brand.txt", "hard-rule-filtered output corpus")
	rawOutput := flag.String("raw-output", "internal/data/corpora/wikidata_company_brand_raw.txt", "normalized corpus before hard-rule filtering")
	flag.Parse()

	if filepath.Clean(*output) == filepath.Clean(*rawOutput) {
		return fmt.Errorf("--output and --raw-output must be different paths")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	client := &http.Client{Timeout: 2 * time.Minute}
	names, err := data.FetchWikidataCompanyBrandNames(ctx, client, data.WikidataSPARQLEndpoint, *limit)
	if err != nil {
		return err
	}

	filtered, audit := gen.FilterHardRuleValidNames(names, 10)
	retrievedAt := time.Now().UTC().Format(time.RFC3339)
	if err := writeCorpus(*rawOutput, names, *limit, retrievedAt, "not applied"); err != nil {
		return fmt.Errorf("write raw corpus: %w", err)
	}
	if err := writeCorpus(*output, filtered, *limit, retrievedAt, "applied"); err != nil {
		return fmt.Errorf("write filtered corpus: %w", err)
	}

	fmt.Printf("raw labels: %d; hard-rule compatible: %d; rejected: %d\n", audit.CandidateCount, audit.AcceptedCount, audit.RejectedCount)
	printAudit(os.Stderr, audit)
	fmt.Printf("wrote filtered corpus to %s and raw corpus to %s\n", *output, *rawOutput)
	return nil
}

func writeCorpus(path string, names []string, requestedLimit int, retrievedAt, hardRuleFilter string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	writer := bufio.NewWriterSize(file, 64*1024)
	writeErr := func() error {
		if _, err := fmt.Fprintln(writer, "# English company and brand labels from Wikidata"); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(writer, "# Wikidata data license: CC0 1.0 Universal"); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(writer, "# Retrieved: %s\n", retrievedAt); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(writer, "# Requested label limit: %d\n", requestedLimit); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(writer, "# Hard-rule filtering: %s\n", hardRuleFilter); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(writer, "# Names in file: %d\n", len(names)); err != nil {
			return err
		}
		for _, name := range names {
			if _, err := fmt.Fprintln(writer, name); err != nil {
				return err
			}
		}
		return writer.Flush()
	}()
	closeErr := file.Close()
	if writeErr != nil {
		return writeErr
	}
	return closeErr
}

func printAudit(w io.Writer, audit gen.HardRuleAudit) {
	for _, rule := range audit.Rules {
		if rule.Hits == 0 {
			continue
		}
		fmt.Fprintf(w, "- %s: %d hit(s); examples: %s\n", rule.Name, rule.Hits, strings.Join(rule.Examples, ", "))
	}
}
