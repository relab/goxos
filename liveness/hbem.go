package liveness

import (
	"sync"
	"time"

	"github.com/relab/goxos/config"
	"github.com/relab/goxos/grp"

	"github.com/relab/goxos/Godeps/_workspace/src/github.com/golang/glog"
)

// State associated with the heartbeat emitter module.
type HbEm struct {
	emInterval  time.Duration
	id          grp.ID
	ticker      *time.Ticker
	resetChan   <-chan bool
	outB        chan<- interface{}
	stop        chan bool
	stopCheckIn *sync.WaitGroup
}

// Create a new heartbeat emitter (HbEm).
func NewHbEm(cfg config.Config, id grp.ID, rc <-chan bool,
	outB chan<- interface{}, stopCheckIn *sync.WaitGroup) *HbEm {
	return &HbEm{
		emInterval:  cfg.GetDuration("hbEmitterInterval", config.DefHbEmitterInterval),
		id:          id,
		resetChan:   rc,
		outB:        outB,
		stop:        make(chan bool),
		stopCheckIn: stopCheckIn,
	}
}

// Start the heartbeat emitter, which periodically broadcasts Heartbeat messages
// onto the network to all replicas.
func (hbem *HbEm) Start() {
	glog.V(1).Info("starting")
	go func() {
		defer hbem.stopCheckIn.Done()
		hbMsg := Heartbeat{ID: hbem.id}
		hbem.outB <- hbMsg
		for {
			select {
			case <-time.After(hbem.emInterval):
				hbem.outB <- hbMsg
			case <-hbem.resetChan:
				if glog.V(4) {
					glog.Info("received reset")
				}
			case <-hbem.stop:
				glog.V(1).Info("exiting")
				return
			}
		}
	}()
}

// Stop the heartbeat emitter.
func (hbem *HbEm) Stop() {
	hbem.stop <- true
}
