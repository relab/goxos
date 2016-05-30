package main

import (
	"fmt"
	"time"

	e "github.com/relab/goxos/elog/event"
)

type Run struct {
	ID     string
	Parsed bool
	Valid  bool

	Errors   []string
	Warnings []string

	LogURLs        map[string]*logURL
	ClientPlots    map[string]string
	ThroughputPlot string

	ReplicaEvents     map[string][]e.Event
	ReplicaThroughput map[string][]e.Event
	ClientLatencies   map[string][]e.Event
	StateHashes       map[string]string

	TimeZero time.Time
	Timeline timeline

	RunSummary

	AverageThroughput float64

	RequestsDecided uint64

	MedianClLatency    float64
	Low90ErrClLatency  float64
	High90ErrClLatency float64
	MaxClLatency       float64
	MaxClLatencyTs     time.Time

	MaxActClLatency   float64
	MaxActClLatencyTs time.Time
}

func (r *Run) getRunRootCollect() string {
	return fmt.Sprintf("%s/%s/%s", exproot, cfg.GetString("localCollectDir", ""), r.ID)
}

func (r *Run) getRunRootReport() string {
	return fmt.Sprintf("%s/%s/%s", exproot, reportFolder, r.ID)
}

type logURL struct {
	Elog, Glog, Stderr string
}

// Reconf-LR-ARec specific for now
type RunSummary struct {
	// InitStart, InitDone, ActivationStart, ActivationDone
	fa, is, id, as, ad, pr e.Event

	ActivationStart map[string]time.Time
	ActivationDone  map[string]time.Time
	Processing      map[string]time.Time
	ActivationDur   []time.Duration
	WaitforFirstAcc []time.Duration

	BaseThroughSamples []uint64
	InitThroughSamples []uint64
	ActiThroughSamples []uint64

	InitDur, ActiDur, FirstAcc time.Duration

	BaseTputMean, BaseTputSSD, BaseTputStdErrOfMean float64
	BaseTputMin, BaseTputMax                        uint64
	InitTputMean, InitTputSSD, InitTputStdErrOfMean float64
	InitTputMin, InitTputMax                        uint64
	ActiTputMean, ActiTputSSD, ActiTputStdErrOfMean float64
	ActiTputMin, ActiTputMax                        uint64

	LatencySample map[string][]float64
	MaxLatency    map[string]float64
	MaxLatencyTs  map[string]time.Time

	//ActMaxLatency   map[string]float64
	//ActMaxLatencyTs map[string]time.Time

	// ARec specific: Activated but not running yet
	afcps       e.Event
	ARecActiDur time.Duration
}

const (
	initStart       = e.InitTransferStart
	initDone        = e.InitInitialized
	activationStart = e.InitInitialized
	activationDone  = e.Running
	processing      = e.Processing
	failure         = e.ShutdownStart
	// ARec specific: Activated but not running yet
	arecActivatedFromCPs = e.ARecActivatedFromCPs
)
