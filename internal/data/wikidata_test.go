package data

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestNormalizeCompanyBrandName(t *testing.T) {
	tests := []struct {
		label string
		want  string
	}{
		{label: "Acme, Inc.", want: "acme"},
		{label: "The Acme Company, LLC", want: "theacme"},
		{label: "Acme Limited Liability Company", want: "acme"},
		{label: "Thé Brand GmbH", want: "thbrand"},
		{label: "A", want: ""},
		{label: "A Very Long Company Name Incorporated", want: ""},
	}

	for _, test := range tests {
		t.Run(test.label, func(t *testing.T) {
			if got := NormalizeCompanyBrandName(test.label); got != test.want {
				t.Fatalf("NormalizeCompanyBrandName(%q) = %q, want %q", test.label, got, test.want)
			}
		})
	}
}

func TestFetchWikidataCompanyBrandNames(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("request method = %q, want GET", r.Method)
		}
		if r.Header.Get("User-Agent") != wikidataUserAgent {
			t.Errorf("User-Agent = %q, want %q", r.Header.Get("User-Agent"), wikidataUserAgent)
		}
		query, err := url.QueryUnescape(r.URL.Query().Get("query"))
		if err != nil {
			t.Errorf("decode query: %v", err)
		}
		if !strings.Contains(query, "LIMIT 50") {
			t.Errorf("query %q does not contain requested limit", query)
		}
		w.Header().Set("Content-Type", "application/sparql-results+json")
		_, _ = w.Write([]byte(`{"results":{"bindings":[{"label":{"value":"Acme Corporation"}},{"label":{"value":"Acme, Inc."}},{"label":{"value":"Foo & Co."}},{"label":{"value":"A"}}]}}`))
	}))
	defer server.Close()

	names, err := FetchWikidataCompanyBrandNames(context.Background(), server.Client(), server.URL, 50)
	if err != nil {
		t.Fatalf("FetchWikidataCompanyBrandNames() error = %v", err)
	}
	want := []string{"acme", "foo"}
	if len(names) != len(want) {
		t.Fatalf("names = %v, want %v", names, want)
	}
	for i := range want {
		if names[i] != want[i] {
			t.Fatalf("names = %v, want %v", names, want)
		}
	}
}

func TestLoadTrainingWordsCombinesAndDeduplicatesCorpora(t *testing.T) {
	humanNames, err := loadEmbeddedWords("names.txt")
	if err != nil {
		t.Fatalf("load human-name corpus: %v", err)
	}
	trainingWords, err := LoadTrainingWords()
	if err != nil {
		t.Fatalf("LoadTrainingWords() error = %v", err)
	}
	if len(trainingWords) <= len(humanNames)+3000 {
		t.Fatalf("training corpus size = %d, want human names %d plus company/brand names", len(trainingWords), len(humanNames))
	}

	seen := make(map[string]struct{}, len(trainingWords))
	for _, word := range trainingWords {
		key := strings.ToLower(word)
		if _, duplicate := seen[key]; duplicate {
			t.Errorf("duplicate training corpus word %q", word)
		}
		seen[key] = struct{}{}
	}
}

func TestCommittedWikidataCorpusIsNormalizedAndUnique(t *testing.T) {
	words, err := loadEmbeddedWords("corpora/wikidata_company_brand.txt")
	if err != nil {
		t.Fatalf("load company/brand corpus: %v", err)
	}
	if len(words) < 3000 {
		t.Fatalf("committed company/brand corpus has %d words, want at least 3000", len(words))
	}

	previous := ""
	for _, word := range words {
		if len(word) < 2 || len(word) > 12 {
			t.Errorf("word %q has unsupported length", word)
		}
		for i := 0; i < len(word); i++ {
			if word[i] < 'a' || word[i] > 'z' {
				t.Errorf("word %q is not normalized lowercase ASCII", word)
				break
			}
		}
		if previous != "" && word <= previous {
			t.Errorf("corpus is not sorted and unique at %q after %q", word, previous)
		}
		previous = word
	}
}

func TestFetchWikidataCompanyBrandNamesRejectsInvalidLimit(t *testing.T) {
	if _, err := FetchWikidataCompanyBrandNames(context.Background(), nil, "", 0); err == nil {
		t.Fatal("expected non-positive limit to be rejected")
	}
}
