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
	gen.RandomPronounceableWord(config)
}
