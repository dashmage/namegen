package main

import (
	"github.com/dashmage/namegen/internal/cli"
	"github.com/dashmage/namegen/internal/gen"
)

func main() {
	config := cli.Parse()
	if config.SeedSet {
		gen.SetSeed(config.Seed)
		config.RunSeed = config.Seed
	} else {
		config.RunSeed = gen.SeedWithTime()
	}

	result := gen.Generate(gen.Config{
		Attempts:  config.Attempts,
		Count:     config.Count,
		Length:    config.Length,
		Threshold: config.Threshold,
	})
	cli.PrintRunResult(result, config.Debug, config.RunSeed, config.SeedSet)
}
