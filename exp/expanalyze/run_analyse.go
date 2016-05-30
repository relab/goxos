package main

import (
	"math"
	"time"

	e "github.com/relab/goxos/elog/event"
	"github.com/relab/goxos/exp/util"

	m "github.com/relab/goxos/Godeps/_workspace/src/github.com/pkg/math"
)

func (r *Run) Analyse(genStats bool) {
	if !r.Parsed {
		return
	}
	if valid := r.verifyState(); !valid {
		return
	}
	if valid := r.calculateTimeZero(); !valid {
		return
	}
	if genStats {
		if valid := r.generateStatistics(); !valid {
			return
		}
	}
	if valid := r.calcAverageThroughput(); !valid {
		return
	}

	if valid := r.calcRequestsDecided(); !valid {
		return
	}

	r.Valid = true
}

func (r *Run) verifyState() (valid bool) {
	if len(r.StateHashes) == 0 {
		r.Errors = append(r.Errors, "No state hashes found")
		return false
	}

	var initialHash string
	for _, initialHash = range r.StateHashes {
		break
	}
	for _, hash := range r.StateHashes {
		if hash != initialHash {
			r.Errors = append(r.Errors, "State hashes differ")
			return false
		}
	}

	return true
}

func (r *Run) calculateTimeZero() (vaild bool) {
	timeZero := time.Unix(math.MaxInt32, 0)
	for _, tle := range r.Timeline {
		if tle.Event.Type != e.Start {
			continue
		}
		if tle.Event.Time.Before(timeZero) {
			timeZero = tle.Event.Time
		}
	}

	if timeZero.Unix() == math.MaxInt64 {
		r.Errors = append(r.Errors, "Could not calculate time zero")
		return false
	}
	r.TimeZero = timeZero

	return true
}

func (r *Run) generateStatistics() (valid bool) {
	if valid := r.calcDurations(); !valid {
		return false
	}
	if valid := r.calcThroughput(); !valid {
		return false
	}

	return true
}

// NB: For now specific for Reconfig-LR-ARec analysis
func (r *Run) calcDurations() (valid bool) {
	// No stanbys specified, nothing to analyse
	if len(expreport.Hostnames.Standbys) == 0 {
		r.Errors = append(r.Errors, "calcDurations: no standby replica found")
		return false
	}

	r.ActivationStart = make(map[string]time.Time, 1)
	r.ActivationDone = make(map[string]time.Time, 1)
	r.Processing = make(map[string]time.Time, 1)
	r.ActivationDur = make([]time.Duration, 0, 1)
	r.WaitforFirstAcc = make([]time.Duration, 0, 1)

	// Find elog for stanby replica, assume that there is only one standby
	for _, stdb := range expreport.Hostnames.Standbys {
		parsed := r.getStdbEvents(stdb)
		if !parsed {
			return false
		}
	}

	// Calculate durations
	r.InitDur = r.id.Time.Sub(r.is.Time)
	r.ActiDur = util.MaxDurationN(r.ActivationDur...)
	r.FirstAcc = util.MaxDurationN(r.WaitforFirstAcc...)

	return true
}

func (r *Run) getStdbEvents(stdb string) bool {
	standbyElog, found := r.ReplicaEvents[stdb]
	if !found {
		r.Errors = append(r.Errors, "calcDurations: no elog found for standby replica")
		return false
	}

	// Scan for initStart
	for _, event := range standbyElog {
		if event.Type == initStart {
			r.is = event
			break
		}
	}
	if r.is.Type == e.Unknown {
		r.Errors = append(r.Errors, "calcDurations: no InitStart event found for standby replica")
		return false
	}

	// Scan for initDone
	for _, event := range standbyElog {
		if event.Type == initDone {
			r.id = event
			break
		}
	}
	if r.is.Type == e.Unknown {
		r.Errors = append(r.Errors, "calcDurations: no InitDone event found for standby replica")
		return false
	}

	// Scan for activationStart
	for _, event := range standbyElog {
		if event.Type == activationStart {
			r.as = event
			break
		}
	}
	if r.as.Type == e.Unknown {
		r.Errors = append(r.Errors, "calcDurations: no ActivationStart event found for standby replica")
		return false
	}

	// Scan for activationDone
	for _, event := range standbyElog {
		if event.Type == activationDone {
			r.ad = event
			break
		}
	}
	if r.ad.Type == e.Unknown {
		r.Errors = append(r.Errors, "calcDurations: no ActivationDone event found for standby replica")
		return false
	}

	// Scan for processing
	for _, event := range standbyElog {
		if event.Type == processing {
			r.pr = event
			break
		}
	}
	if r.pr.Type != e.Unknown {
		r.Processing[stdb] = r.pr.Time
		r.WaitforFirstAcc = append(r.WaitforFirstAcc, r.pr.Time.Sub(r.as.Time))
	}

	r.ActivationStart[stdb] = r.as.Time
	r.ActivationDone[stdb] = r.ad.Time
	r.ActivationDur = append(r.ActivationDur, r.ad.Time.Sub(r.as.Time))

	// ARec specific: Calculate duration from Init to ActivatedFromCPs
	if expreport.ARecEnabled {
		// Scan for arecActivatedFromCPs
		for _, event := range standbyElog {
			if event.Type == arecActivatedFromCPs {
				r.afcps = event
				break
			}
		}
		if r.afcps.Type == e.Unknown {
			r.Errors = append(r.Errors, "calcDurations: no ActivatingFromCPs event found for standby replica")
			return false
		}
		r.ARecActiDur = r.afcps.Time.Sub(r.as.Time)
	}

	return true
}

// NB: For now specific for Reconfig-LR-ARec analysis
func (r *Run) calcThroughput() (valid bool) {
	// No failures specified, nothing to analyse
	if len(expreport.Hostnames.Failures) == 0 {
		r.Errors = append(r.Errors, "calcThroughput: no failures specified")
		return false
	}

	// Find elog for failure replica, assume that there is only one failure
	failureElog, found := r.ReplicaEvents[expreport.Hostnames.Failures[0]]
	if !found {
		r.Errors = append(r.Errors, "calcThroughput: no elog found for failure replica")
		return false
	}

	// Scan for failure event
	for _, event := range failureElog {
		if event.Type == failure {
			r.fa = event
			break
		}
	}
	if r.fa.Type == e.Unknown {
		r.Errors = append(r.Errors, "calcThroughput: no failure event found for failure replica")
		return false
	}

	r.BaseThroughSamples = make([]uint64, 0)
	r.InitThroughSamples = make([]uint64, 0)
	r.ActiThroughSamples = make([]uint64, 0)

	// Leader samples
	leader := expreport.Hostnames.Nodes[len(expreport.Hostnames.Nodes)-1]
	leaderSamples, found := r.ReplicaThroughput[leader]
	if !found {
		r.Errors = append(r.Errors, "calcThroughput: no throughput samples found for paxos leader replica")
		return false
	}

	// Sort samples
	for _, ts := range leaderSamples {
		switch {
		case ts.Time.Before(r.fa.Time):
			r.BaseThroughSamples = append(r.BaseThroughSamples, ts.Value)
		case ts.Time.After(r.is.Time) && ts.Time.Before(r.id.Time):
			r.InitThroughSamples = append(r.InitThroughSamples, ts.Value)
		case ts.Time.After(r.as.Time) && ts.Time.Before(r.ad.Time):
			r.ActiThroughSamples = append(r.ActiThroughSamples, ts.Value)
		default:
			// Drop sample
		}
	}

	if len(r.BaseThroughSamples) > 0 {
		// Make sure to filter out any ramp up
		r.BaseThroughSamples = r.BaseThroughSamples[5:]

		r.BaseTputMean = util.MeanUint64(r.BaseThroughSamples...)
		r.BaseTputSSD = util.SSDUint64(r.BaseThroughSamples...)
		r.BaseTputStdErrOfMean = util.StdErrMeanFloat(r.BaseTputSSD, len(r.BaseThroughSamples))
		r.BaseTputMin = m.MinUint64N(r.BaseThroughSamples...)
		r.BaseTputMax = m.MaxUint64N(r.BaseThroughSamples...)
		// Adjust results to req/second
		r.BaseTputMean *= tsc
		r.BaseTputSSD *= tsc
		r.BaseTputStdErrOfMean *= tsc
		r.BaseTputMin *= uint64(tsc)
		r.BaseTputMax *= uint64(tsc)
	}

	if len(r.InitThroughSamples) > 0 {
		r.InitTputMean = util.MeanUint64(r.InitThroughSamples...)
		r.InitTputSSD = util.SSDUint64(r.InitThroughSamples...)
		r.InitTputStdErrOfMean = util.StdErrMeanFloat(r.InitTputSSD, len(r.InitThroughSamples))
		r.InitTputMin = m.MinUint64N(r.InitThroughSamples...)
		r.InitTputMax = m.MaxUint64N(r.InitThroughSamples...)
		// Adjust results to req/second
		r.InitTputMean *= tsc
		r.InitTputSSD *= tsc
		r.InitTputStdErrOfMean *= tsc
		r.InitTputMin *= uint64(tsc)
		r.InitTputMax *= uint64(tsc)
	}

	if len(r.ActiThroughSamples) > 0 {
		r.ActiTputMean = util.MeanUint64(r.ActiThroughSamples...)
		r.ActiTputSSD = util.SSDUint64(r.ActiThroughSamples...)
		r.ActiTputStdErrOfMean = util.StdErrMeanFloat(r.ActiTputSSD, len(r.ActiThroughSamples))
		r.ActiTputMin = m.MinUint64N(r.ActiThroughSamples...)
		r.ActiTputMax = m.MaxUint64N(r.ActiThroughSamples...)
		// Adjust results to req/second
		r.ActiTputMean *= tsc
		r.ActiTputSSD *= tsc
		r.ActiTputStdErrOfMean *= tsc
		r.ActiTputMin *= uint64(tsc)
		r.ActiTputMax *= uint64(tsc)
	}

	return true
}

func (r *Run) calcAverageThroughput() (valid bool) {
	// Leader samples
	leader := expreport.Hostnames.Nodes[len(expreport.Hostnames.Nodes)-1]
	leaderSamples, found := r.ReplicaThroughput[leader]
	if !found {
		r.Errors = append(r.Errors, "calcAverageThroughput: no throughput samples found for paxos leader replica")
		return false
	}
	var throughPuts []uint64
	for _, throughPut := range leaderSamples {
		throughPuts = append(throughPuts, throughPut.Value)
	}
	r.AverageThroughput = util.MeanUint64(throughPuts...)
	//r.AverageThroughput = r.AverageThroughput / float64(expdesc.ElogTSInterval) * 1000
	r.AverageThroughput *= tsc
	return true
}

func (r *Run) calcRequestsDecided() (valid bool) {
	// Leader samples
	leader := expreport.Hostnames.Nodes[len(expreport.Hostnames.Nodes)-1]
	leaderSamples, found := r.ReplicaThroughput[leader]
	if !found {
		r.Errors = append(r.Errors, "calcPercentageOfRequestsDecided: no throughput samples found for paxos leader replica")
		return false
	}
	var sum uint64
	for _, throughput := range leaderSamples {
		sum += throughput.Value
	}
	r.RequestsDecided = sum
	return true
}
