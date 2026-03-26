package data

import (
	"bufio"
	"embed"
	"strings"
)

// corpusFS stores the default in-repo training corpus.
//
//go:embed names.txt
var corpusFS embed.FS

// LoadWords loads normalized corpus entries from the embedded file.
func LoadWords() ([]string, error) {
	f, err := corpusFS.Open("names.txt")
	if err != nil {
		return nil, err
	}
	defer f.Close()

	words := make([]string, 0, 1024)
	scanner := bufio.NewScanner(f)
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
