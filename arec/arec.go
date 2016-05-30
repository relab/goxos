package arec

import (
	"errors"
	"sync"

	"github.com/relab/goxos/app"
	"github.com/relab/goxos/config"
	"github.com/relab/goxos/grp"
	"github.com/relab/goxos/liveness"
	"github.com/relab/goxos/net"
	"github.com/relab/goxos/nodeinit"
	"github.com/relab/goxos/paxos"

	"github.com/relab/goxos/elog"
	e "github.com/relab/goxos/elog/event"

	"github.com/relab/goxos/Godeps/_workspace/src/github.com/golang/glog"
)

// ReplacementHandler represents a Goxos submodule handling the core Asynchronous
// Reconfiguration protocol. It is responsible for handling startup for replicas
// joining a Goxos cluster as the result of a replacement. The module is also
// responsible for initiating replacements based failure indications.
type AReconfHandler struct {
	id                 grp.ID
	rleader            grp.ID
	config             *config.Config
	appID              string
	stateful           bool
	valid              bool
	acceptorState      *paxos.AcceptorSlotMap
	accFirstSlot       paxos.SlotID
	acceptorRelease    chan bool
	reconfInProgress   bool
	muReconfInProgress sync.Mutex
	confs              map[grp.Epoch]map[grp.ID]grp.Node
	maxEpochSeen       grp.Epoch
	cPromises          map[grp.Epoch][]*CPromise
	replicaProvider    nodeinit.ReplicaProvider
	dmx                net.Demuxer
	grpmgr             grp.GroupManager
	fd                 *liveness.Fd
	ld                 liveness.LeaderDetector
	fdChan             <-chan liveness.FdMsg
	repLdChan          <-chan grp.ID
	bcast              chan<- interface{}
	ucast              chan<- net.Packet
	reconfMsgChan      <-chan ReconfMsg
	cPromiseChan       <-chan CPromise
	activationChan     <-chan Activation
	runPaxosChan       chan bool
	cmdChan            chan ReconfCmd
	appStateReqChan    chan<- app.StateReq
	acceptor           paxos.Acceptor
	proposer           paxos.Proposer
	adu                *paxos.Adu
	stop               chan bool
	stopCheckIn        *sync.WaitGroup
}

// NewAReconfHandler returns a new areconfig handler.
func NewAReconfHandler(id grp.ID, conf *config.Config, appID string,
	grpmgr grp.GroupManager, fd *liveness.Fd, ld liveness.LeaderDetector,
	bcast chan<- interface{}, ucast chan<- net.Packet, dmx net.Demuxer,
	asrch chan<- app.StateReq, acceptor paxos.Acceptor, proposer paxos.Proposer, adu *paxos.Adu,
	stopCheckIn *sync.WaitGroup) *AReconfHandler {
	return &AReconfHandler{
		id:               id,
		rleader:          ld.ReplacementLeader(),
		config:           conf,
		appID:            appID,
		stateful:         id.Epoch == grp.Epoch(0), //Initial servers are statefull on start.
		valid:            true,
		acceptorRelease:  make(chan bool),
		reconfInProgress: false,
		confs:            make(map[grp.Epoch]map[grp.ID]grp.Node),
		maxEpochSeen:     id.Epoch,
		cPromises:        make(map[grp.Epoch][]*CPromise),
		replicaProvider:  nodeinit.GetReplicaProvider(0, conf),
		dmx:              dmx,
		grpmgr:           grpmgr,
		fd:               fd,
		ld:               ld,
		bcast:            bcast,
		ucast:            ucast,
		runPaxosChan:     make(chan bool),
		cmdChan:          make(chan ReconfCmd, 1),
		appStateReqChan:  asrch,
		acceptor:         acceptor,
		proposer:         proposer,
		adu:              adu,
		stop:             make(chan bool),
		stopCheckIn:      stopCheckIn,
	}
}

// Start starts the AReconfHandler by spawning a goroutine to handle
// incoming and outgoing messages.
func (rh *AReconfHandler) Start() {
	glog.V(1).Info("starting")
	rh.registerAndSubscribe()

	go func() {
		defer rh.stopCheckIn.Done()
		for {
			select {
			case fdmsg := <-rh.fdChan:
				rcmd := rh.handleFdMsg(&fdmsg)
				if rcmd != nil {
					rh.handleReconfCmd(rcmd)
				}
			case ldmsg := <-rh.repLdChan:
				rh.handleRepLdMsg(ldmsg)
			case rm := <-rh.reconfMsgChan:
				cp := rh.handleReconfMsg(&rm)
				if cp != nil {
					rh.sendCP(cp, &rm.OldIds)
				}
			case cp := <-rh.cPromiseChan:
				ac := rh.handleCPromise(cp)
				if ac != nil {
					rh.handleActivation(ac)
				}
			case ac := <-rh.activationChan:
				correct := rh.checkOutsideActivation(&ac)
				if correct {
					rh.handleActivation(&ac)
				}
			case arc := <-rh.cmdChan:
				rh.handleReconfCmd(&arc)
			case <-rh.stop:
				glog.V(1).Info("exiting")
				return
			}
		}
	}()
}

// Stop stops the AReconfHandler.
func (rh *AReconfHandler) Stop() {
	rh.stop <- true
}

// RequestReplacement request replacement of the replica specified
// by rcmd.
func (rh *AReconfHandler) RequestReconf(rcmd ReconfCmd) error {
	rh.cmdChan <- rcmd
	//err := <-rcmd.err
	return nil
}

// WaitForActivation blocks until a replica is activated by using the
// protocol. The method returns immediately if the replica
// is already activated. This method should be used by a replacer replica
// during startup.
func (rh *AReconfHandler) RunningPaxos() {
	if rh.stateful && rh.valid {
		return
	}

	<-rh.runPaxosChan
	glog.V(2).Infoln("valid and statefull activation")
	// Disabled for the current implementation using TCP
	//close(rh.cancelActivationTimer)
	//go rh.signalTerminateToReplacedNode()

	return
}

func (rh *AReconfHandler) registerAndSubscribe() {
	rmch := make(chan ReconfMsg, rh.grpmgr.Quorum())
	rh.dmx.RegisterChannel(rmch)
	rh.reconfMsgChan = rmch

	cpch := make(chan CPromise, rh.grpmgr.NrOfNodes()*2)
	rh.dmx.RegisterChannel(cpch)
	rh.cPromiseChan = cpch

	acch := make(chan Activation)
	rh.dmx.RegisterChannel(acch)
	rh.activationChan = acch

	rh.fdChan = rh.fd.SubscribeToFdMsgs("ar")
	rh.repLdChan = rh.ld.SubscribeToReplacementLdMsgs("ar")
}

func (rh *AReconfHandler) nodeIsReplacementLeader() bool {
	return rh.id == rh.rleader
}

func (rh *AReconfHandler) handleFdMsg(fdmsg *liveness.FdMsg) *ReconfCmd {
	glog.V(2).Infoln("received msg from fd:", fdmsg)

	if !rh.nodeIsReplacementLeader() {
		return nil
	}

	switch fdmsg.Event {
	case liveness.Suspect:
		glog.V(2).Infoln("requesting replacement of node", fdmsg.ID)
		return NewReconfCmdFromID(fdmsg.ID)
	case liveness.Restore:
		return nil
	}

	return nil
}

var (
	ErrInvalidID     = errors.New("provided id is not a part of the running cluster")
	ErrEpochMismatch = errors.New("inconsistency between epoch in id and epoch in epoch vector")
)

func (rh *AReconfHandler) handleReconfCmd(rcmd *ReconfCmd) {
	glog.V(2).Infoln("received command - ", rcmd)
	if !rh.valid || !rh.stateful {
		glog.V(2).Infoln("aborting reconf, since not valid and stateful", rcmd)
		return
	}

	if inProgress := rh.IsReconfigInProgress(); inProgress {
		glog.V(2).Infoln("aborting reconf, since other reconf in progress", rcmd)
		return
	}
	elog.Log(e.NewEvent(e.ARecStart))
	rh.SetReconfigInProgress(true)
	// Try also: go rh.prepareandSend(rcmd)

	rcMsg, newNodes, err := rh.reconfSetupLocal(rcmd)
	if err != nil {
		glog.Errorln("error preparing reconfiguration locally, aborting:", err)
		rh.SetReconfigInProgress(false)
		rh.cmdChan <- *rcmd
		return
	}

	appState := rh.getAppState()

	go rh.doRemoteSetupAndSend(rcMsg, newNodes, appState, rcmd)

	//rh.prepareandSend(rcmd)

}

func (rh *AReconfHandler) doRemoteSetupAndSend(rcMsg *ReconfMsg,
	newNodes map[grp.Node]config.TransferWrapper,
	appState *app.State, rcmd *ReconfCmd) {

	if err := reconfSetupGlobal(newNodes, *appState); err != nil {
		glog.Errorln("error while preparing remote nodes, aborting", rcmd)
		rh.SetReconfigInProgress(false)
		rh.cmdChan <- *rcmd
	} else {
		rh.bcast <- *rcMsg
		elog.Log(e.NewEvent(e.ARecRMSent))
		glog.V(2).Infoln("initalization done, broadcasting ReconfMsg")
	}
}

// SetReconfigInProgress is used to indicate that a reconfiguration is being prepared
func (rh *AReconfHandler) SetReconfigInProgress(inProgress bool) {
	glog.V(2).Infoln("setting reconfig in progress to", inProgress)
	rh.muReconfInProgress.Lock()
	defer rh.muReconfInProgress.Unlock()
	rh.reconfInProgress = inProgress
}

// IsReconfigInProgress returns the value uf reconfInProgress.
func (rh *AReconfHandler) IsReconfigInProgress() bool {
	rh.muReconfInProgress.Lock()
	defer rh.muReconfInProgress.Unlock()
	return rh.reconfInProgress
}

func (rh *AReconfHandler) handleRepLdMsg(repLdID grp.ID) {
	// Set the leader when method returns
	defer func() {
		rh.rleader = repLdID
	}()

	// If the replacement leader id is different from ours, do nothing
	if repLdID != rh.id {
		return
	}

	// We are not the current replacement leader, and the trust message is
	// for our id, handle it (if we are already for some reason are the
	// replacement leader, we do nothing).
	if rh.rleader != rh.id {
		glog.V(2).Infoln("I'm now replacement leader")
		// Let the FD resend every current suspected node in case the
		// failed replacement leader has failed to replace them.
		//rh.fd.ResendCurrentSuspected()
	}
}
