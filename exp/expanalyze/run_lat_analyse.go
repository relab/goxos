package main

import (
	"time"

	"github.com/relab/goxos/exp/util"
)

func (r *Run) calcLatency() {

	r.LatencySample = make(map[string][]float64)
	r.MaxLatency = make(map[string]float64)
	r.MaxLatencyTs = make(map[string]time.Time)

	//Range through Clients
	for cl := range r.ClientLatencies {
		r.LatencySample[cl] = make([]float64, 0)
		r.MaxLatency[cl] = float64(0)
		r.MaxActClLatency = float64(0)

		//Compute Latencies and MaxLatency
		for _, ts := range r.ClientLatencies[cl] {
			lat := float64(ts.EndTime.Sub(ts.Time)) / float64(time.Millisecond)
			r.LatencySample[cl] = append(r.LatencySample[cl], lat)
			if lat > r.MaxLatency[cl] {
				r.MaxLatency[cl] = lat
				r.MaxLatencyTs[cl] = ts.Time
			}

			for stdb, ast := range r.ActivationStart {
				if (ts.EndTime.After(ast)) && (ts.Time.Before(r.ActivationDone[stdb])) {
					if lat > r.MaxActClLatency {
						r.MaxActClLatency = lat
						r.MaxActClLatencyTs = ts.Time
					}
					break
				}
			}
		}
	}

	// Compute Mean of the different Clients Max and Mean Latency
	var latencies = make([]float64, 0)
	for cl := range r.LatencySample {
		latencies = append(latencies, r.LatencySample[cl]...)
	}
	r.MedianClLatency, r.Low90ErrClLatency, r.High90ErrClLatency = util.MedianMinMax90FloatSlice(latencies)
	maxClient := util.MaxFloat64Map(r.MaxLatency)
	if maxClient == "" {
		return
	}
	r.MaxClLatency = r.MaxLatency[maxClient]
	r.MaxClLatencyTs = r.MaxLatencyTs[maxClient]
}
