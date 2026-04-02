package cli

import (
	"flag"
	"fmt"
	"os"
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
	if err := Validate(config); err != nil {
		fmt.Fprintf(os.Stderr, "invalid flags: %v\n", err)
		os.Exit(2)
	}
	return config
}

// Validate rejects CLI configurations that would produce invalid or misleading runs.
func Validate(config CLIConfig) error {
	if config.Attempts <= 0 {
		return fmt.Errorf("attempts must be greater than 0")
	}
	if config.Count <= 0 {
		return fmt.Errorf("count must be greater than 0")
	}
	if config.Length <= 0 {
		return fmt.Errorf("length must be greater than 0")
	}
	return nil
}
