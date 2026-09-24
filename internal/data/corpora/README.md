# Company and brand corpus

`wikidata_company_brand.txt` contains normalized English labels for Wikidata entities whose direct `instance of` relation is `company` (Q783794) or `brand` (Q431289).

- **License:** Wikidata data is available under CC0 1.0 Universal.
- **Refresh:** `go run ./cmd/fetch-corpus --limit=10000`
- **Preparation:** common trailing legal designators are removed, punctuation/non-ASCII bytes are discarded, normalized duplicates are removed, and only names that pass every generator hard rule are retained.
- **Use:** the corpus is embedded and combined with the human-name source by the data loader.

The fetcher uses direct instance types rather than recursively including subclasses to keep the public Wikidata Query Service request bounded. It reports per-rule hit counts and sample labels while preparing the corpus.
