package reconfig

import (
	"sync"

	"github.com/relab/goxos/app"
	"github.com/relab/goxos/config"
	"github.com/relab/goxos/grp"
	"github.com/relab/goxos/liveness"
	"github.com/relab/goxos/net"
	"github.com/relab/goxos/nodeinit"
	"github.com/relab/goxos/paxos"
	rr "github.com/relab/goxos/ringreplacer"

	"github.com/relab/goxos/elog"
	e "github.com/relab/goxos/elog/event"

	"github.com/relab/goxos/Godeps/_workspace/src/github.com/golang/glog"
)

// ReconfigHandler represents a Goxos submodule handling startup for replicas
// joining a Goxos cluster as the result of a reconfiguration. The module is
// also responsible for starting reconfigurations based on failuire
// indications.
type ReconfigHandler struct {
	id                   grp.ID
	pxLeader             grp.ID
	config               *config.Config
	appID                string
	reconfInProgress     bool
	muReconfInProgress   sync.Mutex
	replicaProvider      nodeinit.ReplicaProvider
	dmx                  net.Demuxer
	grpmgr               grp.GroupManager
	ld                   liveness.LeaderDetector
	pxLdChan             <-chan grp.ID
	reconfMsgChan        chan rr.ReconfMsg
	reconfMsgQueue       []rr.ReconfMsg
	bcast                chan<- interface{}
	ucast                chan<- net.Packet
	appStateReqChan      chan<- app.StateReq
	reconfigCmdChan      chan<- paxos.ReconfigCmd
	firstSlotChan        <-chan FirstSlot
	firstSlot            *FirstSlot
	waitForFirstSlotChan chan paxos.SlotID
	joinChan             chan Join
	joined               bool
	waitForJoinChan      chan bool
	stop                 chan bool
	stopCheckIn          *sync.WaitGroup
}

// NewReconfigHandler returns a new reconfiguration handler.
func NewReconfigHandler(id grp.ID, conf *config.Config, appID string,
	grpmgr grp.GroupManager, ld liveness.LeaderDetector,
	recMC chan rr.ReconfMsg, bcast chan<- interface{},
	ucast chan<- net.Packet, dmx net.Demuxer,
	asrch chan<- app.StateReq, reconfCmdChan chan<- paxos.ReconfigCmd,
	stopCheckIn *sync.WaitGroup) *ReconfigHandler {
	return &ReconfigHandler{
		id:                   id,
		pxLeader:             ld.PaxosLeader(),
		config:               conf,
		appID:                appID,
		replicaProvider:      nodeinit.GetReplicaProvider(0, conf),
		dmx:                  dmx,
		grpmgr:               grpmgr,
		ld:                   ld,
		reconfMsgChan:        recMC,
		reconfMsgQueue:       make([]rr.ReconfMsg, 0),
		bcast:                bcast,
		ucast:                ucast,
		appStateReqChan:      asrch,
		reconfigCmdChan:      reconfCmdChan,
		waitForFirstSlotChan: make(chan paxos.SlotID),
		waitForJoinChan:      make(chan bool),
		stop:                 make(chan bool),
		stopCheckIn:          stopCheckIn,
	}
}

// Start starts the ReconfigHandler by spawning a goroutine to handle
// incoming and outgoing messages.
func (rh *ReconfigHandler) Start() {
	glog.V(1).Info("starting")
	rh.registerAndSubscribe()
	go func() {
		defer rh.stopCheckIn.Done()
		for {
			select {
			case recMsg := <-rh.reconfMsgChan:
				rh.handleRecMsg(recMsg)
			case ldMsg := <-rh.pxLdChan:
				rh.handleLdMsg(ldMsg)
			case fs := <-rh.firstSlotChan:
				rh.handleFirstSlot(fs)
			case j := <-rh.joinChan:
				rh.handleJoin(j)
			case <-rh.stop:
				glog.V(1).Info("exiting")
				return
			}
		}
	}()
}

// Stop stops the ReconfigHandler.
func (rh *ReconfigHandler) Stop() {
	rh.stop <- true
}

// SetReconfigInProgress is used to indicate if a reconfiguration is in
// progress or is complete.
func (rh *ReconfigHandler) SetReconfigInProgress(inProgress bool) {
	glog.V(2).Infoln("setting reconfig in progress to", inProgress)
	rh.muReconfInProgress.Lock()
	defer rh.muReconfInProgress.Unlock()
	rh.reconfInProgress = inProgress
	if inProgress == false || rh.nodeIsPaxosLeader() {
		// We are Paxos leader and reconfiguration has either finished
		// or aborted. Reevaluate system state by letting the FD send
		// any current suspected nodes.
		//rh.fd.ResendCurrentSuspected()
	}
}

func (rh *ReconfigHandler) SignalReconfCompleted() {
	glog.V(2).Infoln("Setting Reconf-Completed")
	rh.muReconfInProgress.Lock()
	defer rh.muReconfInProgress.Unlock()
	rh.reconfInProgress = false
	if len(rh.reconfMsgQueue) == 0 {
		return
	}
	rh.reconfMsgChan <- rh.reconfMsgQueue[0]
	if len(rh.reconfMsgQueue) == 1 {
		rh.reconfMsgQueue = rh.reconfMsgQueue[:0]
	} else {
		rh.reconfMsgQueue = rh.reconfMsgQueue[1:]
	}
}

// WaitForFirstSlot blocks until the replica receives the identifier for the
// first slot in a new configuration. The method returns this identifier. This
// method should be used by a new replica starting as part of a new
// configuration.
func (rh *ReconfigHandler) WaitForFirstSlot() paxos.SlotID {
	return <-rh.waitForFirstSlotChan
}

// WaitForJoin blocks until the replica receives its first Join message from
// another replica in the new configuration.  This method should be used by a
// new replica starting as part of a new configuration.
func (rh *ReconfigHandler) WaitForJoin() {
	<-rh.waitForJoinChan
}

func (rh *ReconfigHandler) registerAndSubscribe() {
	firstSlotChan := make(chan FirstSlot, rh.grpmgr.NrOfNodes())
	rh.dmx.RegisterChannel(firstSlotChan)
	rh.firstSlotChan = firstSlotChan

	joinChan := make(chan Join, rh.grpmgr.NrOfNodes())
	rh.dmx.RegisterChannel(joinChan)
	rh.joinChan = joinChan

	rh.dmx.RegisterChannel(rh.reconfMsgChan)

	rh.pxLdChan = rh.ld.SubscribeToPaxosLdMsgs("reconf")
}

func (rh *ReconfigHandler) nodeIsPaxosLeader() bool {
	return rh.id == rh.pxLeader
}

func (rh *ReconfigHandler) handleLdMsg(pxLeaderID grp.ID) {
	// Set the leader when method returns
	defer func() {
		rh.pxLeader = pxLeaderID
	}()

	// If the Paxos leader id is different from ours, do nothing
	//TODO: Resend RecMsg to new leader?
	if pxLeaderID != rh.id {
		return
	}

	// We are not the current Paxos leader, and the trust message is
	// for our id, handle it (if we are already for some reason are the
	// replacement leader, we do nothing).
	if rh.pxLeader != rh.id {
		// Let the FD resend every current suspected node in case the
		// failed replacement leader has failed to replace them.
		//rh.fd.ResendCurrentSuspected()
	}
}

func (rh *ReconfigHandler) handleRecMsg(recMsg rr.ReconfMsg) {
	// Only handle if there are no reconfiguration in progress.
	glog.V(2).Info("handling reconf message")

	// If not the leader, send On
	//TODO: Remember, for resend on Leader Change.
	if !rh.nodeIsPaxosLeader() {
		rh.sendToLeader(recMsg)
		return
	}

	//TODO: Remember, and propose later.
	rh.muReconfInProgress.Lock()
	if rh.reconfInProgress {
		rh.reconfMsgQueue = append(rh.reconfMsgQueue, recMsg)
		rh.muReconfInProgress.Unlock()
		return
	}

	elog.Log(e.NewEvent(e.ReconfigStart))
	rh.reconfInProgress = true
	rh.muReconfInProgress.Unlock()

	rh.createAndProposeReconfigCmd(recMsg)
}

func (rh *ReconfigHandler) createAndProposeReconfigCmd(recMsg rr.ReconfMsg) {
	glog.V(2).Info("attempting to create and propose reconfig command")

	// Create reconfiguration command
	if len(recMsg.NewIds) != 1 {
		glog.Errorln("Multiple new Nodes in one Reconfiguration not yet supported. Replacing only First Node")
	}
	reconfigCmd := paxos.ReconfigCmd{Nodes: recMsg.NodeMap, New: recMsg.NewIds[0]}

	// Propose reconfiguration command
	glog.V(2).Info("proposing reconfiguration command")
	rh.reconfigCmdChan <- reconfigCmd
	elog.Log(e.NewEvent(e.ReconfigPropose))
}

func (rh *ReconfigHandler) sendToLeader(recMsg rr.ReconfMsg) {
	rh.ucast <- net.Packet{DestID: rh.pxLeader, Data: recMsg}
}

func (rh *ReconfigHandler) handleFirstSlot(fs FirstSlot) {
	if rh.firstSlot == nil {
		rh.waitForFirstSlotChan <- paxos.SlotID(fs.Slot)
		rh.firstSlot = &fs
		close(rh.waitForFirstSlotChan)
	}
}

func (rh *ReconfigHandler) handleJoin(j Join) {
	if !rh.joined {
		close(rh.waitForJoinChan)
		rh.joined = true
	}
}
