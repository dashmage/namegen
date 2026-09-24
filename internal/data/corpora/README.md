# External corpora

`wikidata_company_brand.txt` is a separate evaluation corpus of English Wikidata labels whose entities have a direct `instance of` relation to `company` (Q783794) or `brand` (Q431289).

- **License:** Wikidata data is available under CC0 1.0 Universal.
- **Fetch:** `go run ./cmd/fetch-corpus --limit=10000` writes both the normalized raw sample and the hard-rule-filtered corpus.
- **Preparation:** labels are lowercased, common trailing legal designators are removed, punctuation/non-ASCII bytes are discarded, and normalized duplicates are removed. The `_raw.txt` file preserves all 2-12 letter names before pronunciation rules; `wikidata_company_brand.txt` contains the subset that passes every generator hard rule.
- **Usage:** this source is embedded and combined with the legacy human-name corpus by `data.LoadProductionWords()`. It can also be evaluated independently with `go run ./cmd/model-eval --corpus internal/data/corpora/wikidata_company_brand.txt`. The checked-in file preserves the exact retrieved sample; rerunning the query can produce a different sample as Wikidata changes.

The importer uses direct instance types rather than recursively including subclasses. This keeps the query bounded and fast enough for the public Wikidata Query Service. Run `go run ./cmd/audit-corpus` to see per-rule hit counts and example labels from the raw sample. Counts overlap when a label triggers multiple rules.
