package main

import (
	"fmt"
	"io/ioutil"
	"path"
	"sort"
	"strconv"

	e "github.com/relab/goxos/elog/event"
	"github.com/relab/goxos/exp/util"
)

func ParseRun(id int) *Run {
	run := Run{
		ID:          strconv.Itoa(id),
		StateHashes: make(map[string]string),
		Errors:      make([]string, 0),
		Warnings:    make([]string, 0),
	}

	if abort := run.checkRunRoot(); abort {
		return &run
	}
	if abort := run.checkFolders(); abort {
		return &run
	}
	if abort := run.parseFiles(); abort {
		return &run
	}

	run.generateTimeline()
	run.filterThroughput()
	run.Parsed = true

	return &run
}

func (r *Run) checkRunRoot() (failed bool) {
	if exists, _ := util.Exists(r.getRunRootCollect()); !exists {
		r.Valid = false
		r.Errors = append(r.Errors, "Missing run root folder")
		return true
	}

	return false
}

func (r *Run) checkFolders() (failed bool) {
	for _, replica := range expreport.Hostnames.Nodes {
		if exists, _ := util.Exists(path.Join(r.getRunRootCollect(), replica)); !exists {
			r.Valid = false
			r.Errors = append(
				r.Errors,
				fmt.Sprintf("Missing at least log folder for replica %s", replica),
			)
			failed = true
			break
		}
	}

	for _, client := range expreport.Hostnames.Clients {
		if exists, _ := util.Exists(path.Join(r.getRunRootCollect(), client)); !exists {
			r.Valid = false
			r.Errors = append(
				r.Errors,
				fmt.Sprintf("Missing at least log folder for client %s", client),
			)
			failed = true
			break
		}
	}

	for _, standby := range expreport.Hostnames.Standbys {
		if exists, _ := util.Exists(path.Join(r.getRunRootCollect(), standby)); !exists {
			r.Valid = false
			r.Errors = append(
				r.Errors,
				fmt.Sprintf("Missing at least log folder for standby %s", standby),
			)
			failed = true
			break
		}
	}

	return false
}

func (r *Run) parseFiles() (failed bool) {
	if failed = r.readState(); failed {
		return
	}

	if failed = r.readElogs(); failed {
		return
	}

	return false
}

func (r *Run) readState() (failed bool) {
	// Read replica statehashs
	for _, replica := range expreport.Hostnames.Nodes {
		if failed = r.readStateHash(replica); failed {
			return
		}
	}

	// Read standby statehashs
	for _, standby := range expreport.Hostnames.Standbys {
		if failed = r.readStateHash(standby); failed {
			return
		}
	}

	return false
}

func (r *Run) readStateHash(replica string) (failed bool) {
	// Only add statehash if a non-failing replica
	if _, found := failures[replica]; found {
		return false
	}

	hash, err := ioutil.ReadFile(path.Join(r.getRunRootCollect(), replica, cfg.GetString("statehashName", "")))
	if err != nil {
		r.Errors = append(r.Errors, fmt.Sprintf("Reading state hash for replica %s failed: %v", replica, err))
		return true
	}
	r.StateHashes[replica] = string(hash)

	return false
}

func (r *Run) readElogs() (failed bool) {
	r.ReplicaEvents = make(map[string][]e.Event)
	r.ReplicaThroughput = make(map[string][]e.Event)
	for _, replica := range expreport.Hostnames.Nodes {
		events, err := e.Parse(path.Join(r.getRunRootCollect(), replica, cfg.GetString("elogNameReplica", "")))
		if err != nil {
			r.Errors = append(r.Errors, fmt.Sprintf("Reading event log for replica %s failed: %v", replica, err))
			return true
		}
		r.ReplicaEvents[replica], r.ReplicaThroughput[replica] = e.ExtractThroughput(events)
	}

	for _, standby := range expreport.Hostnames.Standbys {
		events, err := e.Parse(path.Join(r.getRunRootCollect(), standby, cfg.GetString("elogNameReplica", "")))
		if err != nil {
			r.Errors = append(r.Errors, fmt.Sprintf("Reading event log for standby %s failed: %v", standby, err))
			return true
		}
		r.ReplicaEvents[standby], r.ReplicaThroughput[standby] = e.ExtractThroughput(events)
	}

	r.ClientLatencies = make(map[string][]e.Event)
	for _, client := range expreport.Hostnames.Clients {
		lats, err := e.Parse(path.Join(r.getRunRootCollect(), client, cfg.GetString("elogNameClient", "")))
		if err != nil {
			r.Errors = append(r.Errors, fmt.Sprintf("Reading event log for client %s failed: %v", client, err))
			return true
		}
		if len(lats) > 10 {
			// Drop first ten requests
			r.ClientLatencies[client] = lats[9:]
		} else {
			r.ClientLatencies[client] = lats
		}
	}

	return false
}

func (r *Run) generateTimeline() {
	r.Timeline = make([]timelineEvent, 0)
	for _, replica := range expreport.Hostnames.Nodes {
		for _, event := range r.ReplicaEvents[replica] {
			r.Timeline = append(r.Timeline, timelineEvent{event, replica, event.String()})
		}
	}
	for _, standby := range expreport.Hostnames.Standbys {
		for _, event := range r.ReplicaEvents[standby] {
			r.Timeline = append(r.Timeline, timelineEvent{event, standby, event.String()})
		}
	}
	sort.Sort(r.Timeline)
}

func (r *Run) filterThroughput() {
	for replica, samples := range r.ReplicaThroughput {
		for i := 0; i < len(samples); i++ {
			if samples[i].Value > 0 {
				r.ReplicaThroughput[replica] = samples[i:]
				break
			}
		}
	}
	for replica, samples := range r.ReplicaThroughput {
		for j := len(samples) - 1; j >= 0; j-- {
			if samples[j].Value > 0 {
				if j == len(samples)-1 {
					break
				} else {
					r.ReplicaThroughput[replica] = samples[:j+1]
				}
				break
			}
		}
	}
}
