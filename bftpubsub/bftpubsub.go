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

	"github.com/relab/goxos/Godeps/_workspace/src/github.com/golang/glog"
)

const appID = "bftpubsub"

var (
	id         = flag.Uint("id", 0, "id for node (must match entry in config file)")
	configFile = flag.String("config-file", "config.ini", "path for configuration file to be used")
	allCores   = flag.Bool("all-cores", false, "use all available logical CPUs")
	gcOff      = flag.Bool("gc-off", false, "turn garbage collection off")
	showHelp   = flag.Bool("help", false, "show this help message and exit")
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

	defer glog.Flush()
	glog.V(1).Infoln("replica id is", *id)

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

	gh := &GoxosHandler{}

	iniconfig, err := common.LoadFile(*configFile)
	if err != nil {
		glog.Errorf("Could not parse config file %s, error: %v", *configFile, err)
	}

	goxosConfig := config.NewConfig()
	for k, v := range iniconfig.Section("goxos") {
		goxosConfig.Set(k, v)
	}

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
