package cli

import (
	"flag"

	"github.com/dashmage/namegen/internal/defaults"
)

type Config struct {
	Attempts  int
	Count     int
	Length    int
	Seed      int64
	SeedSet   bool
	RunSeed   int64
	Debug     bool
	Threshold int
}

func NewConfig(attempts, count, length int, seed int64, seedSet, debug bool, threshold int) Config {
	return Config{
		Attempts:  attempts,
		Count:     count,
		Length:    length,
		Seed:      seed,
		SeedSet:   seedSet,
		RunSeed:   0,
		Debug:     debug,
		Threshold: threshold,
	}
}

func Parse() Config {
	attempts := flag.Int("attempts", defaults.CLIAttemptsDefault, "max attempts per requested name before failing (default: 200)")
	count := flag.Int("count", defaults.CLICountDefault, "number of words to generate (default: 10)")
	length := flag.Int("length", defaults.CLILengthDefault, "length of generated word(s) (default: 5)")
	seed := flag.Int64("seed", 0, "RNG seed for reproducible output (optional)")
	debug := flag.Bool("debug", false, "print scores and generation diagnostics")
	threshold := flag.Int("threshold", defaults.AcceptThreshold, "minimum score required for acceptance")
	flag.Parse()

	seedSet := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "seed" {
			seedSet = true
		}
	})

	config := NewConfig(*attempts, *count, *length, *seed, seedSet, *debug, *threshold)
	return config
}
