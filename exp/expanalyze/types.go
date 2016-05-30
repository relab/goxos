package main

import (
	"time"

	"github.com/relab/goxos/config"
)

type ExpReport struct {
	Hostnames
	Versions
	Cfg       config.Config
	Timestamp string
	OutputFlags
	ExpRuns
	ExpSummary
}

type Hostnames struct {
	Nodes    []string
	Standbys []string
	Failures []string
	Clients  []string
}

type Versions struct {
	Go    string
	Goxos string
}

type OutputFlags struct {
	LREnabled   bool
	ARecEnabled bool
}

type ExpRuns []*Run

type ExpSummary struct {
	Analysed  bool
	Comment   string
	ValidRuns int

	InitDurMean, InitDurSD, InitDurSdOfMean, InitDurMin, InitDurMax time.Duration
	ActiDurMean, ActiDurSD, ActiDurSdOfMean, ActiDurMin, ActiDurMax time.Duration
	FAccDurMean, FAccDurSD, FAccDurSdOfMean, FAccDurMin, FAccDurMax time.Duration

	BaseTputMean, BaseTputSSD, BaseTputStdErrOfMean float64
	BaseTputMin, BaseTputMax                        uint64
	InitTputMean, InitTputSSD, InitTputStdErrOfMean float64
	InitTputMin, InitTputMax                        uint64
	ActiTputMean, ActiTputSSD, ActiTputStdErrOfMean float64
	ActiTputMin, ActiTputMax                        uint64

	MeanMaxActLatency, MeanLatency, MeanMaxLatency                                      float64
	MeanLatMin90, MeanLatMax90, MMALMin90, MMALMax90                                    float64
	ARecActiDurMean, ARecActiDurSD, ARecActiDurSdOfMean, ARecActiDurMin, ARecActiDurMax time.Duration

	TotalAverageThroughput float64
}
