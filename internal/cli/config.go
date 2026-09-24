package cli

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/dashmage/namegen/internal/defaults"
)

type Config struct {
	MaxAttempts    int
	Count          int
	Length         int
	LengthProvided bool
	Substring      string
	Seed           int64
	UserSeed       bool
	DebugEnabled   bool
	TuneEnabled    bool
	Threshold      int
}

func NewConfig(attempts, count, length int, seed int64, userSeed, debug, tune bool, threshold int) Config {
	return Config{
		MaxAttempts:    attempts,
		Count:          count,
		Length:         length,
		LengthProvided: true,
		Seed:           seed,
		UserSeed:       userSeed,
		DebugEnabled:   debug,
		TuneEnabled:    tune,
		Threshold:      threshold,
	}
}

func Parse() Config {
	attempts := flag.Int("attempts", defaults.MaxAttempts, "maximum total candidate attempts for the entire run (default: 200)")
	count := flag.Int("count", defaults.Count, "number of names to generate (default: 10)")
	length := flag.Int("length", defaults.Length, "length of generated name(s) (default: 5)")
	substring := flag.String("substring", "", "substring required in generated names (requires --length at least 2 characters longer)")
	seed := flag.Int64("seed", 0, "RNG seed for reproducible output (optional)")
	debug := flag.Bool("debug", false, "print scores and generation diagnostics")
	tune := flag.Bool("tune", false, "interactive tuning mode")
	threshold := flag.Int("threshold", defaults.Threshold, "minimum score required for acceptance")
	flag.Parse()

	userSeed := false
	lengthProvided := false
	flag.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "seed":
			userSeed = true
		case "length":
			lengthProvided = true
		}
	})

	resolvedSeed := *seed
	if !userSeed {
		resolvedSeed = time.Now().UnixNano()
	}

	config := NewConfig(*attempts, *count, *length, resolvedSeed, userSeed, *debug, *tune, *threshold)
	config.LengthProvided = lengthProvided
	config.Substring = *substring
	if err := Validate(config); err != nil {
		fmt.Fprintf(os.Stderr, "invalid flags: %v\n", err)
		os.Exit(2)
	}
	config.Substring = strings.ToLower(config.Substring)
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
	if config.Substring == "" {
		return nil
	}
	if !config.LengthProvided {
		return fmt.Errorf("--substring requires an explicit --length")
	}
	if !isASCIIAlpha(config.Substring) {
		return fmt.Errorf("substring must contain only ASCII letters")
	}
	if config.Length < len(config.Substring)+2 {
		return fmt.Errorf("length must be at least %d when substring length is %d", len(config.Substring)+2, len(config.Substring))
	}
	return nil
}

func isASCIIAlpha(value string) bool {
	for i := 0; i < len(value); i++ {
		if (value[i] < 'a' || value[i] > 'z') && (value[i] < 'A' || value[i] > 'Z') {
			return false
		}
	}
	return true
}
