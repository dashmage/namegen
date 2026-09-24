# namegen

I've always wanted to have a way to reliably generate plausible sounding names for new projects.

`namegen` is a Go CLI app that randomly generates short, pronounceable names.

The current implementation uses three layers:

1. Template-based random name construction (using a vowel/consonant rhythm)
2. Rule-based filtering and penalties
3. Corpus-trained character trigram scoring

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
5. Apply a score adjustment using the corpus-trained trigram model
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

## Character trigram model

Name scoring uses one character trigram model with Laplace smoothing, conditioning each letter on the previous two letters.

- [Human-name corpus](./internal/data/names.txt)
- [Company/brand corpus](./internal/data/corpora/wikidata_company_brand.txt)
- [Corpus refresh command](./cmd/fetch-corpus/main.go)
- [Corpus loader](./internal/data/corpus.go)
- [Model implementation](./internal/gen/model.go)

Both corpora are embedded, combined, and deduplicated before training.

### Training and smoothing

For each corpus word:

1. normalize to lowercase `a-z`
2. add two start tokens and one end token: `^^word$`
3. count every three-character window `(a,b,c)` and the number of transitions for each `(a,b)` context

The trigram probability is:

`P(c|ab) = (Count(a,b,c) + alpha) / (Row(a,b) + alpha * VocabSize)`

`alpha` defaults to `0.5`; `VocabSize` is 28 (`a-z`, `^`, and `$`). Smoothing gives unseen contexts a uniform probability across the vocabulary rather than consulting a separate model.

The model sums log probabilities, including start and end transitions, then averages them so scores are comparable across lengths. The resulting average maps to the configured score adjustment using the `VeryLowProbCutoff`, `LowProbCutoff`, `MidProbCutoff`, and `GoodProbBonusCutoff` anchors. Values outside the anchors are clamped; probability bands are diagnostic labels.
