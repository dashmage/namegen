package main

import (
	"github.com/dashmage/namegen/internal/cli"
	"github.com/dashmage/namegen/internal/gen"
)

func main() {
	config := cli.Parse()
	gen.SetSeed(config.Seed)
	if config.TuneEnabled {
		cli.RunTuneSession(config.Length)
		return
	}

	result := gen.Generate(gen.Options{
		MaxAttempts: config.MaxAttempts,
		Count:       config.Count,
		Length:      config.Length,
		Threshold:   config.Threshold,
		TuneEnabled: config.TuneEnabled,
	})
	cli.PrintResult(result, config.DebugEnabled, config.TuneEnabled, config.Seed, config.UserSeed)
}
