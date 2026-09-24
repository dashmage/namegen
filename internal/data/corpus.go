package data

import (
	"bufio"
	"embed"
	"io"
	"strings"
)

// corpusFS stores the in-repo source corpora.
//
//go:embed names.txt corpora/wikidata_company_brand.txt
var corpusFS embed.FS

// LoadTrainingWords combines the human-name list and cleaned Wikidata
// company/brand corpus, deduplicating normalized entries across sources.
func LoadTrainingWords() ([]string, error) {
	humanNames, err := loadEmbeddedWords("names.txt")
	if err != nil {
		return nil, err
	}
	companyBrandNames, err := LoadCompanyBrandWords()
	if err != nil {
		return nil, err
	}

	words := make([]string, 0, len(humanNames)+len(companyBrandNames))
	seen := make(map[string]struct{}, len(humanNames)+len(companyBrandNames))
	for _, entry := range append(humanNames, companyBrandNames...) {
		name := strings.ToLower(entry)
		if _, duplicate := seen[name]; duplicate {
			continue
		}
		seen[name] = struct{}{}
		words = append(words, name)
	}
	return words, nil
}

// LoadCompanyBrandWords loads the cleaned embedded company/brand corpus.
func LoadCompanyBrandWords() ([]string, error) {
	return loadEmbeddedWords("corpora/wikidata_company_brand.txt")
}

func loadEmbeddedWords(path string) ([]string, error) {
	f, err := corpusFS.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return scanWords(f)
}

func scanWords(reader io.Reader) ([]string, error) {
	words := make([]string, 0, 1024)
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		words = append(words, line)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return words, nil
}
