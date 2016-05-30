package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"runtime/debug"

	"github.com/relab/goxos/config"
	"github.com/relab/goxos/kvs/common"
)

var (
	// General
	gcOff    = flag.Bool("gc-off", false, "turn garbage collection off")
	showHelp = flag.Bool("help", false, "show this help message and exit")

	// Config
	configFile = flag.String("config-file", "config.ini", "path for configuration file to be used")

	// Mode
	mode = flag.String("mode", "", "run mode: (user | bench | bench-async | exp)")

	// Benchmark mode
	report   = flag.Bool("report", false, "bench: save run report to disk")
	runs     = flag.Int("runs", 5, "bench: number of runs")
	noreport = flag.Bool("noreport", false, "bench: disable report")

	// Experiement mode
	nclients = flag.Int("nclients", 1, "exp: number of clients to run")
	keyset   = flag.String("keyset", "", "exp: path to gob encoded key set")

	// Common for both benchmark and experiment mode
	cmds    = flag.Int("cmds", 500, "bench/exp: number of commands per run")
	kl      = flag.Int("kl", 16, "bench/exp: number of bytes for key")
	vl      = flag.Int("vl", 16, "bench/exp: number of bytes for value")
	prewait = flag.Duration("prewait", 0, "batch/exp: pre-start wait")
)

func Usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s [OPTIONS]\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "\nOptions:\n")
	flag.PrintDefaults()
}

var (
	buffer       = new(bytes.Buffer)
	clientConfig *config.Config
)

func main() {
	parseFlags()

	if *gcOff {
		debug.SetGCPercent(-1)
	}

	parseConfig()

	switch *mode {
	case "", "user":
		runUser()
	case "bench":
		runBench()
	case "bench-async":
		runBenchAsync()
	case "exp":
		runExp()
	default:
		fmt.Fprintf(os.Stderr, "Unkown mode specified: %q\n", *mode)
		flag.Usage()
	}
}

func parseConfig() {
	iniconfig, err := common.LoadFile(*configFile)
	if err != nil {
		fmt.Printf("Could not parse config file %s, error: %v", *configFile, err)
	}

	clientConfig = config.NewConfig()
	for k, v := range iniconfig.Section("goxos") {
		clientConfig.Set(k, v)
	}

	for k, v := range iniconfig.Section("client") {
		clientConfig.Set(k, v)
	}
}

func parseFlags() {
	flag.Usage = Usage
	flag.Parse()
	if *showHelp {
		flag.Usage()
		os.Exit(0)
	}
}
