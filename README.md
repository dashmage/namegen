# namegen

I've always wanted to have a way to reliably generate plausible sounding names for new projects.

`namegen` is a Go CLI app that randomly generates short, pronounceable names.

The current implementation uses three layers:

1. Template-based random name construction (using a vowel/consonant rhythm)
2. Rule-based filtering and penalties
3. Corpus-trained interpolated trigram scoring with bigram backoff

## Install
```bash
git clone https://github.com/dashmage/namegen.git
cd namegen
go build
./namegen
```

## Usage
```bash
# by default, namegen generates 10 random 5-letter names
$ namegen
matog
sebeg
xaire
cuzer
moevy
lagok
hukar
pemox
pasit
rioqu

# generate 3 6-letter names
$ namegen --count=3 --length=6
nezila
pepyom
fozlar

# optional seed value for deterministic output
$ namegen --count=5 --length=5 --seed=42
libuf
padai
saire
keipy
jifat

# require "ora" in every generated 5-letter name
$ namegen --substring=ora --length=5
```

Here's all the possible flags (see [internal/cli/config.go](./internal/cli/config.go)):

- `--attempts` maximum total candidate attempts for the entire run
- `--count` number of names to generate
- `--length` generated name length
- `--substring` required substring (must be paired with an explicit `--length` at least two characters longer)
- `--seed` optional RNG seed for reproducible output
- `--threshold` minimum acceptance score
- `--debug` print scores and generation diagnostics
- `--tune` start an interactive tuning session: rate each generated name from `1-5`, and `namegen` provides penalty/cutoff suggestions

## How does it work?

At a high level, the CLI loops until it has produced the requested number of names or exhausted `attempts` total tries:

1. Build a randomly generated candidate name with a weighted rhythm template (`CV`, `CVC`, `CVV`, `VC`)
2. Set a baseline score and threshold for an acceptable name.
3. Apply hard rules (rules that reject the candidate immediately on failure)
4. Apply soft rules (rules that subtract penalties from the score)
5. Apply a score adjustment using the production interpolated trigram model trained on human and company/brand names
6. Accept the candidate if final score is above threshold

Accepted names are unique within a run; repeated candidates are rejected and count against the attempt limit.

The core flow is implemented in:

- `internal/gen/generator.go`
- `internal/gen/rules.go`
- `internal/gen/score.go`
- `internal/gen/model.go`

## Template-based random name generation

Instead of drawing each letter uniformly from `a-z`, name candidates are built from vowel/consonant patterns to create more natural rhythm. The optional `--substring` constraint reserves an internal span for the requested letters and fills the surrounding positions from the rhythm pattern.

- `C` = consonant
- `V` = vowel

The generator samples complete syllable-shaped templates where the requested length allows them:

- `CV` (weight 5)
- `CVC` (weight 6)
- `CVV` (weight 2)
- `VC` (weight 1)

Templates are concatenated only when the remaining length can be filled by complete templates; the final template is no longer cut off. A one-letter request uses a small fallback because it cannot form a complete syllable shape.

Additional shaping:

- prevent `VVV` triplets by converting the middle `V` to `C`
- slightly bias the final template toward a consonant ending without removing its only vowel nucleus
- de-emphasize `y` in vowel sampling
- reject a required `--substring` if it contains a hard-rule violation that would make every candidate impossible

These heuristics encourage pronounceable-looking output, but they are not a guarantee of how people will pronounce a coined name. Check out [generator.go](./internal/gen/generator.go) for the implementation.

## Rules: hard vs soft

Rules are separated by behavior:

- Hard rules: immediate reject
- Soft rules: keep the candidate, subtract score

Hard rules

- three consecutive consonants
- illegal ending characters
- missing a core vowel (`a/e/i/o/u`)
- triple repeated letters
- disallowed consonant adjacency

Soft rules

- uncommon or awkward letter sequences
- `q` not followed by `u`
- too many rare letters (`j`, `q`, `x`, `z`)
- repeated identical vowel pairs
- doubled consonant endings

## Character n-gram model

Production scoring uses a character trigram model with smoothed bigram backoff. The extra context helps distinguish sequences with the same adjacent letters, while bigram backoff stabilizes sparse contexts. A standalone bigram model remains as the evaluation baseline.

- [Default corpus file](./internal/data/names.txt)
- [External company/brand corpus](./internal/data/corpora/wikidata_company_brand.txt)
- [Corpus preparation command](./cmd/fetch-corpus/main.go)
- [Hard-rule audit command](./cmd/audit-corpus/main.go)
- [Loader](./internal/data/corpus.go)
- [Model](./internal/gen/model.go)

Both corpus files are embedded; the production loader combines and deduplicates them while preserving each source file and its provenance.

### Bigram backoff fields

`BigramModel` stores the fallback transition statistics:

- `Count map[[2]byte]int`
  - counts of each transition, e.g. (`t`,`h`) -> 1842
- `Row map[byte]int`
  - total transitions leaving a character, e.g. `t` -> sum of all `t -> *`
- `Alpha float64`
  - Laplace smoothing factor

Constants:

- `StartToken = '^'`
- `EndToken = '$'`
- `VocabSize = 28` (`a-z` plus `^`, `$`)

### Training

For each corpus word, the bigram backoff model:

1. normalizes to lowercase `a-z`
2. adds boundaries: `^word$`
3. counts each adjacent pair `(a,b)` in `Count[(a,b)]` and increments `Row[a]`

The production trigram model also prepends a second start token, then counts each next character given the previous two characters. It interpolates that smoothed estimate with the bigram backoff; the backoff strength is selected using the validation split by `cmd/model-eval`.

### Laplace smoothing

Without smoothing, unseen transitions have probability 0, which can collapse the whole word probability.

Laplace smoothing avoids that:

`P(b|a) = (Count(a,b) + alpha) / (Row(a) + alpha * VocabSize)`

This keeps unseen pairs possible but still low-probability.

### Log probability

Word probability is a product of many small values. Multiplication underflows and is harder to debug.

Using logs converts products into sums:

`log P(word) = sum(log P(next|current))`

The models use **average** log probability so scores are comparable across lengths.

The score adjustment is a bounded, piecewise-linear mapping of that average, rather than one fixed adjustment per band. It interpolates between these anchors:

- `VeryLowProbCutoff` -> `-VeryLowProbPenalty`
- `LowProbCutoff` -> `-LowProbPenalty`
- `MidProbCutoff` -> `-MidProbPenalty`
- `GoodProbBonusCutoff` -> `+GoodProbBonus`

Values beyond the anchors are clamped. Probability bands remain as coarse diagnostic labels; the actual adjustment is stored with the band and uses the continuous score.

Production uses one `InterpolatedTrigramModel` to score names. It estimates `P(c|ab)` with Laplace smoothing and interpolates it with bigram backoff `P(c|b)`. The trigram weight is `count(ab) / (count(ab) + backoffStrength)`, so sparse contexts rely more on bigrams. Production uses backoff strength 20, selected on validation data from the combined corpus. The standalone bigram model is retained only as the backoff component and evaluation baseline, not as a second selectable production scorer.

### Bigram baseline example

This calculation illustrates the bigram fallback probabilities used by the production model.

Corpus words:

- `lena`, `lora`, `nora`, `mila`, `mira`, `sora`

Candidate name:

- `lora`

Transitions with boundaries:

- `^ -> l`
- `l -> o`
- `o -> r`
- `r -> a`
- `a -> $`

Assume `alpha = 0.5`, `VocabSize = 28`, and trained counts give:

- `Count(^,l)=2`, `Row(^)=6`
- `Count(l,o)=1`, `Row(l)=3`
- `Count(o,r)=3`, `Row(o)=3`
- `Count(r,a)=4`, `Row(r)=4`
- `Count(a,$)=6`, `Row(a)=6`

Then:

- `P(l|^) = (2+0.5)/(6+14) = 0.125`, `ln = -2.079`
- `P(o|l) = (1+0.5)/(3+14) = 0.0882`, `ln = -2.428`
- `P(r|o) = (3+0.5)/(3+14) = 0.2059`, `ln = -1.580`
- `P(a|r) = (4+0.5)/(4+14) = 0.2500`, `ln = -1.386`
- `P($|a) = (6+0.5)/(6+14) = 0.3250`, `ln = -1.124`

Log sum:

- `-8.597`

Average log probability:

- `-8.597 / 5 = -1.719`

Scoring flow example:

1. hard rules pass
2. no soft penalties triggered
3. probability band for `-1.719` gives a small bonus
4. final score stays above acceptance threshold
5. candidate accepted as a name

## Comparing model variants

Use the separate Wikidata company/brand corpus to evaluate scoring changes without changing the embedded production corpus:

```sh
go run ./cmd/model-eval \
  --corpus internal/data/corpora/wikidata_company_brand.txt \
  --seed=42 \
  --backoffs=0,1,5,10,20,50
```

The command creates a deterministic 70/15/15 train/validation/test split. It chooses trigram backoff strength using validation likelihood, then reports untouched-test cross-entropy, held-out-versus-generated score gaps, and pairwise ranking accuracy. Generated comparison names are length-matched to test names and must pass the generator's hard rules. Try multiple `--seed` values to check split sensitivity.

Optional human ratings can be supplied as CSV:

```csv
name,rating
lora,5
mira,3
```

```sh
go run ./cmd/model-eval --corpus internal/data/corpora/wikidata_company_brand.txt --ratings ratings.csv
```

Rated names are excluded from the corpus split to avoid exact-name leakage; the report includes Spearman rank correlation between model likelihood and ratings. The evaluation command does not change production scoring.
