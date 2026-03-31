package cli

import (
	"flag"
	"time"

	"github.com/dashmage/namegen/internal/defaults"
)

type CLIConfig struct {
	Attempts  int
	Count     int
	Length    int
	Seed      int64
	UserSeed  bool
	Debug     bool
	Tune      bool
	Threshold int
}

func NewCLIConfig(attempts, count, length int, seed int64, userSeed, debug, tune bool, threshold int) CLIConfig {
	return CLIConfig{
		Attempts:  attempts,
		Count:     count,
		Length:    length,
		Seed:      seed,
		UserSeed:  userSeed,
		Debug:     debug,
		Tune:      tune,
		Threshold: threshold,
	}
}

func Parse() CLIConfig {
	attempts := flag.Int("attempts", defaults.CLIAttemptsDefault, "max attempts per requested name before failing (default: 200)")
	count := flag.Int("count", defaults.CLICountDefault, "number of words to generate (default: 10)")
	length := flag.Int("length", defaults.CLILengthDefault, "length of generated word(s) (default: 5)")
	seed := flag.Int64("seed", 0, "RNG seed for reproducible output (optional)")
	debug := flag.Bool("debug", false, "print scores and generation diagnostics")
	tune := flag.Bool("tune", false, "print per-attempt score breakdown for tuning")
	threshold := flag.Int("threshold", defaults.AcceptThreshold, "minimum score required for acceptance")
	flag.Parse()

	userSeed := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "seed" {
			userSeed = true
		}
	})

	resolvedSeed := *seed
	if !userSeed {
		resolvedSeed = time.Now().UnixNano()
	}

	config := NewCLIConfig(*attempts, *count, *length, resolvedSeed, userSeed, *debug, *tune, *threshold)
	return config
}
