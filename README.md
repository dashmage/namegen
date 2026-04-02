# namegen

I've always wanted to have a way to reliably generate plausible sounding names for new projects.

`namegen` is a Go CLI app that randomly generates short, pronounceable names.

The current implementation uses three layers:

1. Template-based random word construction (using a vowel/consonant rhythm)
2. Rule-based filtering and penalties
3. Corpus-trained bigram scoring

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
```

Here's all the possible flags (from `internal/cli/config.go`):

- `--attempts` max random word generation attempts
- `--count` number of words to generate
- `--length` generated word length
- `--seed` optional RNG seed for reproducible output
- `--threshold` minimum acceptance score
- `--debug` print scores and generation diagnostics
- `--tune` print stats for all randomly generated words for tuning values

## How does it work?

At a high level, the CLI loops until it has produced the requested number of words or exhausted `attempts` total tries:

1. Build a candidate word with a weighted rhythm template (`CV`, `CVC`, `CVV`, `VC`)
2. Apply hard rules (rules that reject the word immediately on failure)
3. Apply soft rules (rules that subtract penalties)
4. Apply a bigram score adjustment from the trained model
5. Accept the candidate if final score is above threshold

The core flow is implemented in:

- `internal/gen/generator.go`
- `internal/gen/rules.go`
- `internal/gen/score.go`
- `internal/gen/model.go`

## Template-based random word generation

Instead of drawing each letter uniformly from `a-z`, candidates are built from vowel/consonant patterns to create more natural rhythm.

- `C` = consonant
- `V` = vowel

The generator samples from weighted templates:

- `CV` (weight 5)
- `CVC` (weight 6)
- `CVV` (weight 2)
- `VC` (weight 1)

Templates are concatenated until the requested length is reached, then trimmed to exact length.

Additional shaping:

- prevent `VVV` triplets by converting the middle `V` to `C`
- slightly bias final character toward consonants
- de-emphasize `y` in vowel sampling

This structure dramatically improves pronounceability compared to fully uniform random letters. Check out [generator.go](./internal/gen/generator.go) to get a better idea.

## Rules: hard vs soft

Rules are separated by behavior:

- Hard rules: immediate reject
- Soft rules: keep candidate, subtract score

Hard rules

- three consecutive consonants
- illegal ending characters
- missing a core vowel (`a/e/i/o/u`)
- triple repeated letters
- disallowed consonant adjacency

Soft rules

- uncommon or awkward sequences (`qx`, `jq`, `qj`, etc.)
- `q` not followed by `u`
- too many rare letters (`j`, `q`, `x`, `z`)
- repeated identical vowel pairs
- doubled consonant endings

## Bigram model

The bigram model scores how plausible adjacent letter transitions are, based on a corpus.

- [Corpus file](./internal/data/names.txt)
- [Loader](./internal/data/corpus.go)
- [Model](./internal/gen/model.go)

### BigramModel fields

`BigramModel` stores:

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

For each corpus word:

1. normalize to lowercase `a-z`
2. add boundaries: `^word$`
3. for each adjacent pair `(a,b)`:
   - `Count[(a,b)]++`
   - `Row[a]++`

### Laplace smoothing

Without smoothing, unseen transitions have probability 0, which can collapse the whole word probability.

Laplace smoothing avoids that:

`P(b|a) = (Count(a,b) + alpha) / (Row(a) + alpha * VocabSize)`

This keeps unseen pairs possible but still low-probability.

### Log probability

Word probability is a product of many small values. Multiplication underflows and is harder to debug.

Using logs converts products into sums:

`log P(word) = sum(log P(next|current))`

The model uses **average** log probability so scores are comparable across lengths.

### End-to-end example

Corpus words:

- `lena`, `lora`, `nora`, `mila`, `mira`, `sora`

Candidate:

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
5. candidate accepted
