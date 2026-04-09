package main

import (
	"github.com/dashmage/namegen/internal/cli"
	"github.com/dashmage/namegen/internal/gen"
)

func main() {
	config := cli.Parse()
	gen.SetSeed(config.Seed)
	if config.Tune {
		cli.RunTuneSession(config.Length)
		return
	}

	result := gen.Generate(gen.GenConfig{
		Attempts:  config.Attempts,
		Count:     config.Count,
		Length:    config.Length,
		Threshold: config.Threshold,
		Tune:      config.Tune,
	})
	cli.PrintRunResult(result, config.Debug, config.Tune, config.Seed, config.UserSeed)
}
