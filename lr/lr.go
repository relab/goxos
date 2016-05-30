package lr

import (
	"errors"
	"sync"

	"github.com/relab/goxos/config"
	"github.com/relab/goxos/grp"
	"github.com/relab/goxos/net"
	"github.com/relab/goxos/paxos"
	"github.com/relab/goxos/ringreplacer"

	"github.com/relab/goxos/elog"
	e "github.com/relab/goxos/elog/event"

	"github.com/relab/goxos/Godeps/_workspace/src/github.com/golang/glog"
)

// ReplacementHandler represents a Goxos submodule handling the core Live
// Replacement protocol. It is responsible for handling startup for replicas
// joining a Goxos cluster as the result of a replacement. The module is also
// responsible for initiating replacements based failure indications.
type ReplacementHandler struct {
	id                    grp.ID
	config                *config.Config
	replacer              bool
	creationSlotMarker    paxos.SlotID
	activated             bool
	activatedSignal       chan bool
	cancelActivationTimer chan bool
	epochPromises         []*EpochPromise
	getReplacerEpoch      func(grp.Epoch) grp.Epoch
	dmx                   net.Demuxer
	grpmgr                grp.GroupManager
	recMsgChan            chan ringreplacer.ReconfMsg
	bcast                 chan<- interface{}
	ucast                 chan<- net.Packet
	prepareEpochChan      chan PrepareEpoch
	epochPromiseChan      <-chan EpochPromise
	forwardedEpSetChan    <-chan ForwardedEpSet
	acceptor              paxos.Acceptor
	adu                   *paxos.Adu
	stop                  chan bool
	stopCheckIn           *sync.WaitGroup
}

// NewReplacementHandler returns a new replacement handler.
func NewReplacementHandler(id grp.ID, conf *config.Config,
	replacer bool, slotMarker paxos.SlotID,
	grpmgr grp.GroupManager, rrChan chan ringreplacer.ReconfMsg,
	bcast chan<- interface{}, ucast chan<- net.Packet, dmx net.Demuxer,
	acceptor paxos.Acceptor, adu *paxos.Adu,
	stopCheckIn *sync.WaitGroup) *ReplacementHandler {
	return &ReplacementHandler{
		id:                    id,
		config:                conf,
		replacer:              replacer,
		activated:             false,
		activatedSignal:       make(chan bool),
		cancelActivationTimer: make(chan bool),
		epochPromises:         make([]*EpochPromise, 0, grpmgr.NrOfNodes()),
		getReplacerEpoch:      epochGenerator(id),
		dmx:                   dmx,
		grpmgr:                grpmgr,
		recMsgChan:            rrChan,
		bcast:                 bcast,
		ucast:                 ucast,
		acceptor:              acceptor,
		adu:                   adu,
		stop:                  make(chan bool),
		stopCheckIn:           stopCheckIn,
	}
}

// Start starts the ReplacementHandler by spawning a goroutine to handle
// incoming and outgoing messages.
func (rh *ReplacementHandler) Start() {
	glog.V(1).Info("starting")
	rh.registerAndSubscribe()
	go func() {
		defer rh.stopCheckIn.Done()
		for {
			select {
			case rmsg := <-rh.recMsgChan:
				rh.handleRecMsg(rmsg)
			case pe := <-rh.prepareEpochChan:
				rh.handlePrepareEpoch(pe)
			case ep := <-rh.epochPromiseChan:
				rh.handleEpochPromise(ep)
			case <-rh.stop:
				glog.V(1).Info("exiting")
				return
			}
		}
	}()
}

// Stop stops the ReplacementHandler.
func (rh *ReplacementHandler) Stop() {
	rh.stop <- true
}

// WaitForActivation blocks until a replica is activated by using the
// Live Replacement protocol. The method returns immediately if the replica
// is already activated. This method should be used by a replacer replica
// during startup.
func (rh *ReplacementHandler) WaitForActivation() {
	if rh.activated {
		return
	}

	<-rh.activatedSignal
	rh.activated = true
}

func (rh *ReplacementHandler) registerAndSubscribe() {
	pech := make(chan PrepareEpoch, rh.grpmgr.Quorum())
	rh.dmx.RegisterChannel(pech)
	rh.prepareEpochChan = pech

	epch := make(chan EpochPromise, rh.grpmgr.NrOfNodes()*2)
	rh.dmx.RegisterChannel(epch)
	rh.epochPromiseChan = epch

	feps := make(chan ForwardedEpSet, 1)
	rh.dmx.RegisterChannel(feps)
	rh.forwardedEpSetChan = feps
}

func (rh *ReplacementHandler) handleRecMsg(rmsg ringreplacer.ReconfMsg) {
	elog.Log(e.NewEvent(e.LRStart))
	glog.V(2).Infoln("received reconfMsg - ", rmsg)
	ppEp, err := rh.checkandhandleRecMsg(rmsg)
	if err != nil {
		glog.Errorln("replacement aborted, reason:", err)
		return
	}

	glog.V(2).Infoln("broadcasting prepareEpoch:", ppEp)
	rh.bcast <- ppEp
	elog.Log(e.NewEvent(e.LRPrepareEpochSent))
	rh.prepareEpochChan <- *ppEp
}

func (rh *ReplacementHandler) checkandhandleRecMsg(rmsg ringreplacer.ReconfMsg) (ppEp *PrepareEpoch, err error) {
	if len(rmsg.NewIds) == 0 {
		glog.Errorln("replacement message without new Nodes")
		return nil, errors.New("Replacement message without new nodes")
	}

	for _, id := range rmsg.NewIds {
		if id.Epoch <= rh.grpmgr.Epochs()[id.PaxosID] {
			return nil, errors.New("New node with old epoch")
		}
	}

	//TODO: Suport multiple replacements in one Message:
	if len(rmsg.NewIds) > 1 {
		glog.Errorln("replacement with multiple new nodes. Only doing the first.")
	}

	if rh.config.GetInt("LRStrategy", config.DefLRStrategy) == 2 {
		rmsg.AduSlot = rh.adu.Value()
	}

	return &PrepareEpoch{rmsg.NewIds[0], rmsg.NodeMap[rmsg.NewIds[0]], rmsg.AduSlot}, nil
}

var (
	ErrInvalidID     = errors.New("provided id is not a part of the running cluster")
	ErrEpochMismatch = errors.New("inconsistency between epoch in id and epoch in epoch vector")
)

func (rh *ReplacementHandler) handleForwardedEpSet(feps *ForwardedEpSet) {
	glog.V(2).Infof("received set of forwarded epoch promises (size=%d)\n", len(feps.EpSet))
	for _, ep := range feps.EpSet {
		rh.handleEpochPromise(*ep)
	}
}
