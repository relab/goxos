package profile_test

import (
	"github.com/relab/goxos/Godeps/_workspace/src/github.com/davecheney/profile"
)

func ExampleStart() {
	// start a simple CPU profile and register
	// a defer to Stop (flush) the profiling data.
	defer profile.Start(profile.CPUProfile).Stop()
}
