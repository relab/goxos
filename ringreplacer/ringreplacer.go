package ringreplacer

import (
	"sync"
	//"time"

	"github.com/relab/goxos/app"
	"github.com/relab/goxos/config"
	"github.com/relab/goxos/grp"
	"github.com/relab/goxos/liveness"
	"github.com/relab/goxos/nodeinit"

	"github.com/relab/goxos/elog"
	e "github.com/relab/goxos/elog/event"

	"github.com/relab/goxos/Godeps/_workspace/src/github.com/golang/glog"
)

// RingReplacer is a failure handling module that listens to suspect messages
// for the previous replica in a Ring, and on receiving a suspect, initializes a replacement.
type RingReplacer struct {
	id                 grp.ID
	config             *config.Config
	appID              string
	reconfInProgress   bool
	muReconfInProgress sync.Mutex
	replicaProvider    nodeinit.ReplicaProvider
	getReplacerEpoch   func(grp.Epoch) grp.Epoch
	watching           grp.ID
	grpmgr             grp.GroupManager
	fd                 *liveness.Fd
	fdChan             <-chan liveness.FdMsg
	appStateReqChan    chan<- app.StateReq
	recMsgChan         chan<- ReconfMsg
	stop               chan bool
	stopCheckIn        *sync.WaitGroup
}

// NewRingReplacer returns a new ringreplacer.
func NewRingReplacer(id grp.ID, conf *config.Config, appID string,
	grpmgr grp.GroupManager, fd *liveness.Fd,
	asrch chan<- app.StateReq, recMsgChan chan<- ReconfMsg,
	stopCheckIn *sync.WaitGroup) *RingReplacer {
	return &RingReplacer{
		id:               id,
		config:           conf,
		appID:            appID,
		replicaProvider:  nodeinit.GetReplicaProvider(int(id.PaxosID), conf),
		getReplacerEpoch: epochGenerator(id),
		watching:         grpmgr.NodeMap().GetNext(id),
		grpmgr:           grpmgr,
		fd:               fd,
		appStateReqChan:  asrch,
		recMsgChan:       recMsgChan,
		stop:             make(chan bool),
		stopCheckIn:      stopCheckIn,
	}
}

//TODO: Reset watching on the new process.
// Start starts the RingReplacer by spawning a goroutine to handle
// incoming and outgoing messages.
func (rr *RingReplacer) Start() {
	glog.V(1).Info("starting")
	rr.registerAndSubscribe()
	go func() {
		defer rr.stopCheckIn.Done()
		for {
			select {
			case fdMsg := <-rr.fdChan:
				rr.handleFdMsg(fdMsg)
			case <-rr.stop:
				glog.V(1).Info("exiting")
				return
			}
		}
	}()
}

// Stop stops the RingReplacer.
func (rr *RingReplacer) Stop() {
	rr.stop <- true
}

func (rr *RingReplacer) registerAndSubscribe() {
	rr.fdChan = rr.fd.SubscribeToFdMsgs("reconf")
	//rr.pxLdChan = rr.ld.SubscribeToPaxosLdMsgs("reconf")
}

func (rr *RingReplacer) handleFdMsg(fdMsg liveness.FdMsg) {
	//glog.V(2).Info("fd message, waiting for full second")
	//ts := time.Now().Truncate(time.Second).Add(time.Second)
	//time.Sleep(ts.Sub(time.Now()))

	glog.V(2).Info("handling fd message")

	if fdMsg.Event != liveness.Suspect {
		return
	}
	//If the node I'm watching is suspected:
	if rr.watching == fdMsg.ID {
		glog.V(2).Info("got suspect for the one I'm watching")
		replace := []grp.ID{rr.watching}
		for {
			//Set Watch on next Node.
			rr.watching = rr.grpmgr.NodeMap().GetNext(rr.watching)
			//Ask Failure Detector if this Node is suspected as well.
			if !rr.fd.IsSuspected(rr.watching) {
				break
			}
			replace = append(replace, rr.watching)
		}
		if uint(len(replace)) >= rr.grpmgr.Quorum() {
			glog.Fatalln("replacing a hole quorum")
		}
		elog.Log(e.NewEvent(e.ReconfigStart))

		rr.createAndProposeReconfigMsg(&replace)
	}

}

func (rr *RingReplacer) createAndProposeReconfigMsg(watching *[]grp.ID) {
	glog.V(2).Info("attempting to create and propose reconfig command")

	// Get new Nodes and their Configs
	recMsg, newConfigs, err := rr.getNewConfigs(watching)
	if err != nil {
		glog.Errorln("error preparing replacements, aborting:", err)
		return
	}

	appState := rr.getAppState()
	recMsg.AduSlot = appState.SlotMarker

	rr.doRemoteSetupAndSend(recMsg, newConfigs, appState)

}

func (rr *RingReplacer) doRemoteSetupAndSend(rcMsg *ReconfMsg,
	newNodes map[grp.Node]config.TransferWrapper,
	appState *app.State) {

	if err := reconfSetupGlobal(newNodes, *appState); err != nil {
		glog.Errorln("error while preparing remote nodes, aborting")
	} else {
		rr.recMsgChan <- *rcMsg
		elog.Log(e.NewEvent(e.ARecRMSent))
		glog.V(2).Infoln("initalization done, sending ReconfMsg")
	}
}
