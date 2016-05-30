package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"runtime/debug"
	"syscall"

	"github.com/relab/goxos"
	"github.com/relab/goxos/config"
	"github.com/relab/goxos/kvs/common"

	"github.com/relab/goxos/Godeps/_workspace/src/github.com/davecheney/profile"
	"github.com/relab/goxos/Godeps/_workspace/src/github.com/golang/glog"
)

const appID = "kvs"

var (
	id             = flag.Uint("id", 0, "id for node (must match entry in config file)")
	configFile     = flag.String("config-file", "config.ini", "path for configuration file to be used")
	mode           = flag.String("mode", "normal", "replica operation mode (normal | standby)")
	allCores       = flag.Bool("all-cores", false, "use all available logical CPUs")
	gcOff          = flag.Bool("gc-off", false, "turn garbage collection off")
	loadState      = flag.String("load-state", "", "file path to gob encoded data to use as initial state")
	standbyIP      = flag.String("standby-ip", "127.0.0.1", "the node's IP address if started as replacer")
	writeStateHash = flag.Bool("write-state-hash", true, "write hash of state to disk on exit")
	cpuprofile     = flag.Bool("cpuprofile", false, "write cpu profile to disk")
	memprofile     = flag.Bool("memprofile", false, "write memory profile to disk")
	blockprofile   = flag.Bool("blockprofile", false, "wirte contention profile to disk")
	showHelp       = flag.Bool("help", false, "show this help message and exit")
)

func Usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s [OPTIONS]\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "\nOptions:\n")
	flag.PrintDefaults()
}

func main() {
	parseFlags()

	if *gcOff {
		debug.SetGCPercent(-1)
	}

	if profilingEnabled() {
		pconfig := generateProfilingConfig()
		defer profile.Start(pconfig).Stop()
	}

	defer glog.Flush()
	glog.V(1).Infoln("replica id is", *id)
	glog.V(1).Infoln("mode is", *mode)

	if *allCores {
		cpus := runtime.NumCPU()
		runtime.GOMAXPROCS(cpus)
		glog.V(1).Infoln("#cpus:", cpus)
	} else {
		glog.V(1).Info("#cpus: single")
	}

	start()
}

func parseFlags() {
	flag.Usage = Usage
	flag.Parse()

	if *showHelp {
		flag.Usage()
		os.Exit(0)
	}
}

func start() {
	// Log any runtime panic to file
	defer func() {
		if r := recover(); r != nil {
			glog.Fatalln("Runtime panic:", r)
		}
	}()

	gh := &GoxosHandler{
		kvmap: make(map[string][]byte),
	}

	if *loadState != "" {
		if *mode == "replacer" || *mode == "reconfig" {
			glog.Warning("configured to load an initial state for a standby node")
		}

		if err := gh.loadState(*loadState); err != nil {
			glog.Fatalln("loading intial state failed:", err)
		}
	}

	defer func() {
		if *writeStateHash {
			if err := gh.writeStateHash(); err != nil {
				glog.Warning(err)
			}
		}
	}()

	iniconfig, err := common.LoadFile(*configFile)
	if err != nil {
		glog.Errorf("Could not parse config file %s, error: %v", *configFile, err)
	}

	goxosConfig := config.NewConfig()
	for k, v := range iniconfig.Section("goxos") {
		goxosConfig.Set(k, v)
	}

	switch *mode {

	case "normal":
		goxos := goxos.NewReplica(*id, appID, *goxosConfig, gh)

		goxos.Init()
		if err := goxos.Start(); err != nil {
			glog.Fatal(err)
		}

		defer func() {
			err := goxos.Stop()
			if err != nil {
				glog.Errorln("error when stopping goxos:", err)
			}
		}()

	case "standby":
		goxos := goxos.NewStandbyReplica(gh, appID, *standbyIP)
		if err := goxos.Standby(); err != nil {
			glog.Fatalln("initialization of goxos replacer failed:", err)
		}

		defer func() {
			err := goxos.Stop()
			if err != nil {
				glog.Errorln("error when stopping goxos:", err)
			}
		}()

	default:
		fmt.Fprintf(os.Stderr, "Unkown operation mode provided (%q)", *mode)
		flag.Usage()
		os.Exit(0)

	}

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, os.Kill, syscall.SIGTERM)

	for {
		select {
		case signal := <-signalChan:
			if exit := handleSignal(signal); exit {
				return
			}
		}
	}
}

func handleSignal(signal os.Signal) bool {
	glog.V(1).Infoln("received signal,", signal)
	switch signal {
	case os.Interrupt, os.Kill, syscall.SIGTERM:
		return true
	default:
		glog.Warningln("unhandled signal", signal)
		return false
	}
}
