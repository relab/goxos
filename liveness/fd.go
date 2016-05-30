package liveness

import (
	"fmt"
	"sync"
	"time"

	"github.com/relab/goxos/config"
	"github.com/relab/goxos/grp"

	"github.com/relab/goxos/elog"
	e "github.com/relab/goxos/elog/event"

	"github.com/relab/goxos/Godeps/_workspace/src/github.com/golang/glog"
)

type fdEvent bool

const (
	Suspect fdEvent = false
	Restore fdEvent = true
)

// The state of the failure detector (Fd)
type Fd struct {
	grpmgr          grp.GroupManager
	grpSubscriber   grp.Subscriber
	alive           map[grp.ID]bool
	suspected       map[grp.ID]bool
	timeout         time.Duration
	Δ               time.Duration
	ticker          *time.Ticker
	heartbeatChan   <-chan grp.ID
	fdSubscribers   map[string]chan FdMsg
	resendSuspected chan bool
	getSuspected    chan SuspectedRequest
	stop            chan bool
	stopCheckIn     *sync.WaitGroup
}

// The type of message sent to other modules who are interested in receiving
// notifications about possible failures.
//
// There are two types of events, as defined in the eventually perfect failure
// detector algorithm description: Suspect and Restore.
type FdMsg struct {
	Event fdEvent
	ID    grp.ID
}

// Construct a new failure detector
func NewFd(id grp.ID, gm grp.GroupManager, cfg config.Config,
	heartbeatChan <-chan grp.ID, stopCheckIn *sync.WaitGroup) *Fd {
	return &Fd{
		grpmgr:          gm,
		alive:           make(map[grp.ID]bool),
		suspected:       make(map[grp.ID]bool),
		fdSubscribers:   make(map[string]chan FdMsg),
		resendSuspected: make(chan bool),
		getSuspected:    make(chan SuspectedRequest),
		timeout:         cfg.GetDuration("fdTimeoutInterval", config.DefFdTimeoutInterval),
		Δ:               cfg.GetDuration("fdDeltaIncrease", config.DefFdDeltaIncrease),
		heartbeatChan:   heartbeatChan,
		stop:            make(chan bool),
		stopCheckIn:     stopCheckIn,
	}
}

// Start running the failure detector, spawn a goroutine to handle incoming and
// outgoing messages.
func (fd *Fd) Start() {
	glog.V(1).Info("starting")
	fd.grpSubscriber = fd.grpmgr.SubscribeToHold("fd")

	go func() {
		defer fd.stopCheckIn.Done()
		fd.ticker = time.NewTicker(fd.timeout)
		for {
			select {
			case <-fd.ticker.C:
				fd.timeoutProcedure()
			case id := <-fd.heartbeatChan:
				fd.alive[id] = true
			case grpPrepare := <-fd.grpSubscriber.PrepareChan():
				fd.handleGrpHold(grpPrepare)
			case <-fd.resendSuspected:
				fd.handleResendSuspected()
			case sr := <-fd.getSuspected:
				fd.handleSuspectedRequest(sr)
			case <-fd.stop:
				glog.V(1).Info("exiting")
				return
			}
		}
	}()
}

// Stop the failure detector.
func (fd *Fd) Stop() {
	fd.stop <- true
}

// If a module is interested in receiving Suspect and Restore events, this interest
// can be registered with this method. A name is needed to uniquely identify the
// subscription.
func (fd *Fd) SubscribeToFdMsgs(name string) <-chan FdMsg {
	fdChan := make(chan FdMsg, fd.grpmgr.Quorum())
	fd.fdSubscribers[name] = fdChan

	return fdChan
}

// Let the current replica know about all currently suspected replicas. Used by the
// Live Replacement module.
func (fd *Fd) ResendCurrentSuspected() {
	fd.resendSuspected <- true
}

// Set a replica to be considered alive by the failure detector
func (fd *Fd) SetAlive(id grp.ID) {
	fd.alive[id] = true
	delete(fd.suspected, id)
}

func (fd *Fd) timeoutProcedure() {
	if glog.V(4) {
		glog.Info("timeout")
	}

	if !fd.isAliveSuspectedIntersectionEmpty() {
		fd.timeout = fd.timeout + fd.Δ
		fd.ticker = time.NewTicker(fd.timeout)
	}

	fd.alive[fd.grpmgr.GetID()] = true // add ourselves
	for _, id := range fd.grpmgr.NodeMap().IDs() {
		if fd.notInAliveAndSuspected(id) {
			fd.suspected[id] = true
			elog.Log(e.NewEventWithMetric(e.FailureHandlingSuspect, uint64(id.PaxosID)))
			fd.publishFdMsg(FdMsg{Suspect, id})
		} else if fd.inAliveAndSuspected(id) {
			delete(fd.suspected, id)
			fd.publishFdMsg(FdMsg{Restore, id})
		}
	}

	fd.alive = make(map[grp.ID]bool)
}

func (fd *Fd) isAliveSuspectedIntersectionEmpty() bool {
	for k := range fd.suspected {
		if _, found := fd.alive[k]; found {
			return false
		}
	}

	return true
}

func (fd *Fd) notInAliveAndSuspected(id grp.ID) bool {
	_, inAlive := fd.alive[id]
	_, inSuspected := fd.suspected[id]

	return !inAlive && !inSuspected
}

func (fd *Fd) inAliveAndSuspected(id grp.ID) bool {
	_, inAlive := fd.alive[id]
	_, inSuspected := fd.suspected[id]

	return inAlive && inSuspected
}

func (fd *Fd) publishFdMsg(msg FdMsg) {
	for _, sub := range fd.fdSubscribers {
		sub <- msg
	}
}

// Return a string representation of an FdMsg.
func (fdmsg FdMsg) String() string {
	switch fdmsg.Event {
	case Suspect:
		return fmt.Sprintf("suspecting node %v", fdmsg.ID)
	case Restore:
		return fmt.Sprintf("restore msg for node %v", fdmsg.ID)
	}

	return "unknown message from fd"
}

func (fd *Fd) handleGrpHold(gp *sync.WaitGroup) {
	glog.V(2).Info("grpmgr hold req")
	gp.Done()
	<-fd.grpSubscriber.ReleaseChan()
	glog.V(2).Info("grpmgr release")
}

func (fd *Fd) handleResendSuspected() {
	glog.V(2).Infoln("resending suspected nodes, size:", len(fd.suspected))
	for id := range fd.suspected {
		// TODO(tormod): The publish channels are buffered up to quorum
		// size, but it's probably best somehow to ensure that this resending
		// can't block the FD.
		fd.publishFdMsg(FdMsg{ID: id})
	}
}

// Is Id suspected
func (fd *Fd) IsSuspected(id grp.ID) bool {
	SusChan := make(chan bool, 1)
	fd.getSuspected <- SuspectedRequest{id, SusChan}
	return <-SusChan
}

func (fd *Fd) handleSuspectedRequest(sr SuspectedRequest) {
	if fd.suspected[sr.id] {
		sr.susChan <- true
	} else {
		sr.susChan <- false
	}
}

type SuspectedRequest struct {
	id      grp.ID
	susChan chan bool
}
