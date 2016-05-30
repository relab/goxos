package main

import (
	"github.com/relab/goxos/Godeps/_workspace/src/github.com/davecheney/profile"
)

func profilingEnabled() bool {
	return *cpuprofile || *memprofile || *blockprofile
}

func generateProfilingConfig() *profile.Config {
	return &profile.Config{
		Quiet:          false,
		CPUProfile:     *cpuprofile,
		MemProfile:     *memprofile,
		BlockProfile:   *blockprofile,
		ProfilePath:    "profile",
		NoShutdownHook: true,
	}
}
