package data

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
)

const WikidataSPARQLEndpoint = "https://query.wikidata.org/sparql"

const wikidataUserAgent = "namegen-corpus-import/1.0 (https://github.com/dashmage/namegen)"

var legalCompanySuffixes = [][]string{
	{"limited", "liability", "company"},
	{"public", "limited", "company"},
	{"private", "limited", "company"},
	{"pty", "limited"},
	{"pty", "ltd"},
	{"limited"},
	{"incorporated"},
	{"corporation"},
	{"company"},
	{"gmbh"},
	{"sarl"},
	{"srl"},
	{"llc"},
	{"llp"},
	{"plc"},
	{"ltd"},
	{"inc"},
	{"corp"},
	{"co"},
	{"ag"},
	{"sa"},
	{"bv"},
	{"nv"},
	{"kg"},
	{"lp"},
	{"pty"},
}

type wikidataResponse struct {
	Results struct {
		Bindings []struct {
			Label struct {
				Value string `json:"value"`
			} `json:"label"`
		} `json:"bindings"`
	} `json:"results"`
}

// FetchWikidataCompanyBrandNames retrieves English labels for Wikidata company
// and brand entities, removes common trailing legal suffixes, and returns
// unique normalized names of 2-12 ASCII letters.
func FetchWikidataCompanyBrandNames(ctx context.Context, client *http.Client, endpoint string, limit int) ([]string, error) {
	if limit <= 0 {
		return nil, fmt.Errorf("limit must be greater than 0")
	}
	if endpoint == "" {
		endpoint = WikidataSPARQLEndpoint
	}
	if client == nil {
		client = http.DefaultClient
	}

	query := fmt.Sprintf(`SELECT DISTINCT ?label WHERE {
  VALUES ?type { wd:Q783794 wd:Q431289 }
  ?item wdt:P31 ?type;
        rdfs:label ?label.
  FILTER(LANG(?label) = "en")
}
LIMIT %d`, limit)
	requestURL := endpoint + "?" + url.Values{
		"query":  []string{query},
		"format": []string{"json"},
	}.Encode()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create Wikidata request: %w", err)
	}
	request.Header.Set("Accept", "application/sparql-results+json")
	request.Header.Set("User-Agent", wikidataUserAgent)

	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("query Wikidata: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("query Wikidata: unexpected HTTP status %s", response.Status)
	}

	var result wikidataResponse
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode Wikidata response: %w", err)
	}

	unique := make(map[string]struct{}, len(result.Results.Bindings))
	for _, binding := range result.Results.Bindings {
		if name := NormalizeCompanyBrandName(binding.Label.Value); name != "" {
			unique[name] = struct{}{}
		}
	}

	names := make([]string, 0, len(unique))
	for name := range unique {
		names = append(names, name)
	}
	sort.Strings(names)
	return names, nil
}

// NormalizeCompanyBrandName converts a label to the lowercase ASCII format
// used by the name models, strips trailing legal designators, and rejects names
// outside the project's supported 2-12 character corpus range.
func NormalizeCompanyBrandName(label string) string {
	tokens := asciiTokens(label)
	for {
		removed := false
		for _, suffix := range legalCompanySuffixes {
			if hasTokenSuffix(tokens, suffix) {
				tokens = tokens[:len(tokens)-len(suffix)]
				removed = true
				break
			}
		}
		if !removed {
			break
		}
	}

	name := strings.Join(tokens, "")
	if len(name) < 2 || len(name) > 12 {
		return ""
	}
	return name
}

func asciiTokens(value string) []string {
	value = strings.ToLower(value)
	tokens := make([]string, 0, 4)
	start := -1
	for i := 0; i <= len(value); i++ {
		isLetter := i < len(value) && value[i] >= 'a' && value[i] <= 'z'
		if isLetter && start < 0 {
			start = i
		}
		if !isLetter && start >= 0 {
			tokens = append(tokens, value[start:i])
			start = -1
		}
	}
	return tokens
}

func hasTokenSuffix(tokens, suffix []string) bool {
	if len(tokens) < len(suffix) {
		return false
	}
	start := len(tokens) - len(suffix)
	for i := range suffix {
		if tokens[start+i] != suffix[i] {
			return false
		}
	}
	return true
}
