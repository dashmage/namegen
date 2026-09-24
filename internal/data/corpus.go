package data

import (
	"bufio"
	"embed"
	"io"
	"os"
	"strings"
)

// corpusFS stores the in-repo source corpora.
//
//go:embed names.txt corpora/wikidata_company_brand.txt
var corpusFS embed.FS

// LoadWords loads normalized corpus entries from the embedded file.
func LoadWords() ([]string, error) {
	return loadEmbeddedWords("names.txt")
}

// LoadProductionWords combines the legacy name list and cleaned Wikidata
// company/brand corpus, deduplicating normalized entries across sources.
func LoadProductionWords() ([]string, error) {
	words := make([]string, 0, 6000)
	seen := make(map[string]struct{}, 6000)
	for _, path := range []string{"names.txt", "corpora/wikidata_company_brand.txt"} {
		entries, err := loadEmbeddedWords(path)
		if err != nil {
			return nil, err
		}
		for _, entry := range entries {
			name := strings.ToLower(entry)
			if _, duplicate := seen[name]; duplicate {
				continue
			}
			seen[name] = struct{}{}
			words = append(words, name)
		}
	}
	return words, nil
}

func loadEmbeddedWords(path string) ([]string, error) {
	f, err := corpusFS.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return scanWords(f)
}

// LoadWordsFromFile loads corpus entries from an external text file. Blank
// lines and comments are ignored, matching the embedded corpus format.
func LoadWordsFromFile(path string) ([]string, error) {
	f, err := os.Open(path)
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
