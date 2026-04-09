package cli

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/dashmage/namegen/internal/defaults"
)

type Config struct {
	MaxAttempts  int
	Count        int
	Length       int
	Seed         int64
	UserSeed     bool
	DebugEnabled bool
	TuneEnabled  bool
	Threshold    int
}

func NewConfig(attempts, count, length int, seed int64, userSeed, debug, tune bool, threshold int) Config {
	return Config{
		MaxAttempts:  attempts,
		Count:        count,
		Length:       length,
		Seed:         seed,
		UserSeed:     userSeed,
		DebugEnabled: debug,
		TuneEnabled:  tune,
		Threshold:    threshold,
	}
}

func Parse() Config {
	attempts := flag.Int("attempts", defaults.MaxAttempts, "max attempts per requested name before failing (default: 200)")
	count := flag.Int("count", defaults.Count, "number of names to generate (default: 10)")
	length := flag.Int("length", defaults.Length, "length of generated name(s) (default: 5)")
	seed := flag.Int64("seed", 0, "RNG seed for reproducible output (optional)")
	debug := flag.Bool("debug", false, "print scores and generation diagnostics")
	tune := flag.Bool("tune", false, "interactive tuning mode")
	threshold := flag.Int("threshold", defaults.Threshold, "minimum score required for acceptance")
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

	config := NewConfig(*attempts, *count, *length, resolvedSeed, userSeed, *debug, *tune, *threshold)
	if err := Validate(config); err != nil {
		fmt.Fprintf(os.Stderr, "invalid flags: %v\n", err)
		os.Exit(2)
	}
	return config
}

// Validate rejects CLI configurations that would produce invalid or misleading runs.
func Validate(config Config) error {
	if config.MaxAttempts <= 0 {
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
