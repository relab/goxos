package main

import (
	"log"
	"time"

	"github.com/relab/goxos/exp/util"
)

func generateReport(runDurations []time.Duration, reqLats [][]time.Duration) {
	var (
		meanRunDurations       time.Duration
		ssdRunDurations        time.Duration
		stdErrOfTheMeanRunDurs time.Duration
	)
	// Durations
	if runDurations != nil {
		meanRunDurations = util.MeanDuration(runDurations...)
		ssdRunDurations = util.SSDDuration(runDurations...)
		stdErrOfTheMeanRunDurs = util.StdErrMeanDuration(ssdRunDurations, len(runDurations))
	}

	// Request latencies
	meanRequestLatenciesPerRun := make([]time.Duration, *runs)
	ssdRequestLatenciesPerRun := make([]time.Duration, *runs)
	stdErrOfTheMeanLatenciesPerRun := make([]time.Duration, *runs)
	for i := range reqLats {
		meanRequestLatenciesPerRun[i] = util.MeanDuration(reqLats[i]...)
		ssdRequestLatenciesPerRun[i] = util.SSDDuration(reqLats[i]...)
		stdErrOfTheMeanLatenciesPerRun[i] = util.StdErrMeanDuration(ssdRequestLatenciesPerRun[i], len(reqLats[i]))
	}

	log.Println("")
	log.Println("---------------------------------------------------")
	log.Println("Report:")
	log.Println("")
	log.Println("Number of runs:", *runs)
	log.Println("Number of clients:", *nclients)
	log.Println("Number of commands per run:", *cmds)
	log.Println("Key byte size:", *kl)
	log.Println("Value byte size:", *vl)
	log.Println("")
	log.Println("")
	if runDurations != nil {
		log.Println("Duration per run:", runDurations)
		log.Println("Mean duration for runs:", meanRunDurations)
		log.Println("Sample standard deviation for run durations:", ssdRunDurations)
		log.Println("Standard error of the mean duration:", stdErrOfTheMeanRunDurs)
		log.Println("")
		log.Println("")
	}
	for i := range reqLats {
		log.Println("")
		log.Println("Run nr.", i)
		log.Println("Mean request latency:", meanRequestLatenciesPerRun[i])
		log.Println("Sample standard deviation for request latency:", ssdRequestLatenciesPerRun[i])
		log.Println("Standard error of the mean request latency:", stdErrOfTheMeanLatenciesPerRun[i])
		log.Println("")
	}
	log.Println("---------------------------------------------------")
	log.Println("")
}
