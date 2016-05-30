package grp

import (
	"errors"
	"sync"
	"time"

	"github.com/relab/goxos/Godeps/_workspace/src/github.com/golang/glog"
)

type GrpMgr struct {
	id                ID
	nodeMap           *NodeMap
	arEnabled         bool
	lrEnabled         bool
	subscribers       map[string]Subscriber
	execHoldReqChan   chan chan bool
	globalReleaseChan chan bool
	stop              chan bool
	stopCheckIn       *sync.WaitGroup
}

func NewGrpMgr(id ID, nm *NodeMap, lrEnabled bool, arEnabled bool, stopCheckIn *sync.WaitGroup) *GrpMgr {
	return &GrpMgr{
		id:                id,
		nodeMap:           nm,
		arEnabled:         arEnabled,
		lrEnabled:         lrEnabled,
		subscribers:       make(map[string]Subscriber),
		execHoldReqChan:   make(chan chan bool),
		globalReleaseChan: make(chan bool),
		stop:              make(chan bool),
		stopCheckIn:       stopCheckIn,
	}
}

func (grpmgr *GrpMgr) Start() {
	glog.V(1).Info("starting")
	go func() {
		defer grpmgr.stopCheckIn.Done()
		for {
			select {
			case releaseChan := <-grpmgr.execHoldReqChan:
				grpmgr.handleHoldReq(releaseChan)
			case <-grpmgr.stop:
				glog.V(1).Info("exiting")
				return
			}
		}
	}()
}

func (grpmgr *GrpMgr) RequestHold(releaseChan chan bool) {
	go func() {
		grpmgr.execHoldReqChan <- releaseChan
	}()
}

func (grpmgr *GrpMgr) SubscribeToHold(name string) Subscriber {
	grpsub := Subscriber{make(chan *sync.WaitGroup, 1), grpmgr.globalReleaseChan}
	grpmgr.subscribers[name] = grpsub
	return grpsub
}

func (grpmgr *GrpMgr) Stop() {
	grpmgr.stop <- true
}

func (grpmgr *GrpMgr) NodeMap() *NodeMap {
	return grpmgr.nodeMap
}

func (grpmgr *GrpMgr) NrOfNodes() uint {
	return grpmgr.nodeMap.NrOfNodes()
}

func (grpmgr *GrpMgr) NrOfAcceptors() uint {
	return grpmgr.nodeMap.NrOfAcceptors()
}

func (grpmgr *GrpMgr) Quorum() uint {
	return grpmgr.nodeMap.Quorum()
}

func (grpmgr *GrpMgr) Epochs() []Epoch {
	return grpmgr.nodeMap.Epochs()
}

func (grpmgr *GrpMgr) LrEnabled() bool {
	return grpmgr.lrEnabled
}

func (grpmgr *GrpMgr) ArEnabled() bool {
	return grpmgr.arEnabled
}

func (grpmgr *GrpMgr) SetNewNodeMap(nm map[ID]Node) {
	grpmgr.nodeMap = NewNodeMap(nm)
}

const groupReadyTimeout = 500 * time.Millisecond

func (grpmgr *GrpMgr) handleHoldReq(releaseChan chan bool) {
	glog.V(2).Info("received hold request")
	nrOfSubs := len(grpmgr.subscribers)
	wg := new(sync.WaitGroup)
	wg.Add(nrOfSubs)

	glog.V(2).Info("sending hold prepare to every subsciber")
	for name, subscriber := range grpmgr.subscribers {
		glog.V(2).Infoln("sending hold prepare to", name)
		subscriber.prepareChan <- wg
	}

	glog.V(2).Info("waiting for every subscriber to signal hold ready")
	groupReady := make(chan bool)
	go func() {
		wg.Wait()
		groupReady <- true
	}()

	select {
	case <-groupReady:
		glog.V(2).Info("received ready from every subscriber")
	case <-time.After(groupReadyTimeout):
		glog.Warningln("group ready timeout:", groupReadyTimeout)
	}

	glog.V(2).Info("allowing read access")
	<-releaseChan
	glog.V(2).Info("waiting for hold release")
	<-releaseChan
	glog.V(2).Info("hold release received")

	glog.V(2).Info("releasing every subscriber")
	close(grpmgr.globalReleaseChan)

	grpmgr.globalReleaseChan = make(chan bool)
}

func (grpmgr *GrpMgr) GetID() ID {
	return grpmgr.id
}

func (grpmgr *GrpMgr) SetID(newID ID) error {
	if newID.PaxosID == grpmgr.id.PaxosID && newID.Epoch > grpmgr.id.Epoch {
		grpmgr.id = newID
		return nil
	}
	return errors.New("invalid ID change")
}
