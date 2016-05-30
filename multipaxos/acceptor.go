package multipaxos

import (
	"sync"

	"github.com/relab/goxos/elog"
	e "github.com/relab/goxos/elog/event"
	"github.com/relab/goxos/grp"
	"github.com/relab/goxos/net"
	px "github.com/relab/goxos/paxos"

	"github.com/relab/goxos/Godeps/_workspace/src/github.com/golang/glog"
)

// A MultiAcceptor holds all of the state for an acceptor in MultiPaxos.
type MultiAcceptor struct {
	id            grp.ID
	started       bool
	startable     bool
	processing    bool
	leader        grp.ID
	lowSlot       px.SlotID // The acceptor can't respond with learns for slots lower than this
	slots         *px.AcceptorSlotMap
	ucast         chan<- net.Packet
	bcast         chan<- interface{}
	trust         <-chan grp.ID
	prepareChan   <-chan px.Prepare
	acceptChan    <-chan px.Accept
	handlePrepare func(*px.Prepare) (*px.Promise, grp.ID)
	handleAccept  func(*px.Accept) *px.Learn
	dmx           net.Demuxer
	grpmgr        grp.GroupManager
	stateReqChan  chan acceptorStateRequest
	slotReqChan   chan acceptorSlotRequest
	stop          chan bool
	stopCheckIn   *sync.WaitGroup
}

// NewMultiAcceptor returns a new acceptor based on the state in pp.
func NewMultiAcceptor(pp *px.Pack) *MultiAcceptor {
	ma := &MultiAcceptor{
		id:           pp.ID,
		startable:    pp.RunAcc,
		processing:   false,
		leader:       pp.Ld.PaxosLeader(),
		lowSlot:      pp.NextExpectedDcd,
		slots:        px.NewAcceptorSlotMap(),
		ucast:        pp.Ucast,
		bcast:        pp.Bcast,
		trust:        pp.Ld.SubscribeToPaxosLdMsgs("acceptor"),
		dmx:          pp.Dmx,
		grpmgr:       pp.Gm,
		stateReqChan: make(chan acceptorStateRequest),
		slotReqChan:  make(chan acceptorSlotRequest),
		stop:         make(chan bool),
		stopCheckIn:  pp.StopCheckIn,
	}

	// Set maxSeen to lowSlot in slot map
	ma.slots.MaxSeen = ma.lowSlot

	if ma.grpmgr.LrEnabled() {
		ma.handleAccept = ma.handleAccLr
		ma.handlePrepare = ma.handlePrepLr
	} else if ma.grpmgr.ArEnabled() {
		ma.handlePrepare = ma.handlePrepAr
		ma.handleAccept = ma.handleAccAr
	} else {
		ma.handlePrepare = ma.handlePrep
		ma.handleAccept = ma.handleAcc
	}

	return ma
}

// Start starts the acceptor.
func (a *MultiAcceptor) Start() {
	if !a.startable || a.started {
		glog.Warning("ignoring start request")
		return
	}

	glog.V(1).Info("starting")
	a.started = true
	a.registerChannels()

	go func() {
		defer a.stopCheckIn.Done()
		for {
			select {
			case prepare := <-a.prepareChan:
				promise, dest := a.handlePrepare(&prepare)
				if promise != nil {
					a.send(*promise, dest)
				}
			case accept := <-a.acceptChan:
				learn := a.handleAccept(&accept)
				if learn != nil {
					a.broadcast(*learn)
				}
			case trustID := <-a.trust:
				a.leader = trustID
			case asr := <-a.stateReqChan:
				a.handleStateRequest(asr)
			case sr := <-a.slotReqChan:
				a.handleSlotRequest(sr)
			case <-a.stop:
				a.started = false
				glog.V(1).Info("exiting")
				return
			}
		}
	}()
}

// Stop stops the acceptor.
func (a *MultiAcceptor) Stop() {
	if a.started {
		a.stop <- true
	}
}

func (a *MultiAcceptor) registerChannels() {
	acceptChan := make(chan px.Accept, 64)
	a.acceptChan = acceptChan
	a.dmx.RegisterChannel(acceptChan)

	prepareChan := make(chan px.Prepare, 8)
	a.prepareChan = prepareChan
	a.dmx.RegisterChannel(prepareChan)
}

func (a *MultiAcceptor) handlePrep(msg *px.Prepare) (*px.Promise, grp.ID) {
	if glog.V(3) {
		glog.Infoln(
			"got prepare from", msg.ID, "with crnd",
			msg.CRnd, "and slot id", msg.Slot)
	}

	// If the round number for the Promise is lower or equal
	// to the highest one in which we have participated: ignore.
	// Future work: Reply with some kind of NACK.
	if a.slots.Rnd.Compare(msg.CRnd) >= 0 {
		return nil, grp.UndefinedID()
	}

	a.slots.Rnd = msg.CRnd
	max := a.slots.MaxSeen
	var accslots []px.AcceptorSlot
	if int(msg.Slot-max) >= 0 {
		accslots = make([]px.AcceptorSlot, msg.Slot-max)
	} else {
		accslots = make([]px.AcceptorSlot, 0)
	}

	// For every slot in range [msg.Slot, max]:
	// Add to []PromiseMsg if VRnd != ⊥
	for i := msg.Slot; i <= max; i++ {
		slot := a.slots.GetSlot(i)
		if slot.VRnd.Compare(px.ZeroRound) > 0 {
			accslots = append(accslots, *slot)
		}
	}

	return &px.Promise{
		ID:       a.id,
		Rnd:      a.slots.Rnd,
		AccSlots: accslots,
	}, msg.ID
}

func (a *MultiAcceptor) handlePrepAr(msg *px.Prepare) (*px.Promise, grp.ID) {
	if glog.V(3) {
		glog.Infoln(
			"got prepare from", msg.ID, "with crnd",
			msg.CRnd, "and slot id", msg.Slot,
		)
	}

	//If from other epoch (configuration), ignore!
	if msg.ID.Epoch != a.id.Epoch {
		return nil, grp.UndefinedID()
	}

	// If the round number for the Promise is lower or equal
	// to the highest one in which we have participated: ignore.
	// Future work: Reply with some kind of NACK.
	if a.slots.Rnd.Compare(msg.CRnd) >= 0 {
		return nil, grp.UndefinedID()
	}

	a.slots.Rnd = msg.CRnd
	max := a.slots.MaxSeen
	var accslots []px.AcceptorSlot
	if int(msg.Slot-max) >= 0 {
		accslots = make([]px.AcceptorSlot, msg.Slot-max)
	} else {
		accslots = make([]px.AcceptorSlot, 0)
	}

	// For every slot in range [msg.Slot, max]:
	// Add to []PromiseMsg if VRnd != ⊥
	for i := msg.Slot; i <= max; i++ {
		slot := a.slots.GetSlot(i)
		if slot.VRnd.Compare(px.ZeroRound) > 0 {
			accslots = append(accslots, *slot)
		}
	}

	return &px.Promise{
		ID:       a.id,
		Rnd:      a.slots.Rnd,
		AccSlots: accslots,
	}, msg.ID
}

func (a *MultiAcceptor) handlePrepLr(msg *px.Prepare) (*px.Promise, grp.ID) {
	if glog.V(3) {
		glog.Infoln(
			"got prepare from", msg.ID, "with crnd",
			msg.CRnd, "and slot id", msg.Slot,
		)
	}

	// If the round number for the Promise is lower or equal
	// to the highest one in which we have participated: ignore.
	// Future work: Reply with some kind of NACK.
	if a.slots.Rnd.Compare(msg.CRnd) >= 0 {
		return nil, grp.UndefinedID()
	}

	a.slots.Rnd = msg.CRnd
	max := a.slots.MaxSeen
	var accslots []px.AcceptorSlot
	if int(msg.Slot-max) >= 0 {
		accslots = make([]px.AcceptorSlot, msg.Slot-max)
	} else {
		accslots = make([]px.AcceptorSlot, 0)
	}

	// For every slot in range [msg.Slot, max]:
	// Add to []PromiseMsg if VRnd != ⊥
	for i := msg.Slot; i <= max; i++ {
		slot := a.slots.GetSlot(i)
		if slot.VRnd.Compare(px.ZeroRound) > 0 {
			accslots = append(accslots, *slot)
		}
	}

	return &px.Promise{
		ID:          a.id,
		Rnd:         a.slots.Rnd,
		AccSlots:    accslots,
		EpochVector: a.grpmgr.Epochs(),
	}, msg.ID
}

func (a *MultiAcceptor) handleAcc(msg *px.Accept) *px.Learn {
	if !a.processing {
		a.processing = true
		elog.Log(e.NewEvent(e.Processing))
	}

	if msg.Slot < a.lowSlot {
		return nil
	}

	slot := a.slots.GetSlot(msg.Slot)

	// If the round number for the Accept is lower or equal to the highest
	// one in which we have participated, or the message round is eqaual to
	// vrnd: ignore.
	if a.slots.Rnd.Compare(msg.Rnd) == 1 || msg.Rnd.Compare(slot.VRnd) == 0 {
		return nil
	}

	// If the round number for the Accept for some reason is higher than
	// the highest one in which we have participated: update our round
	// variable to this value.
	if a.slots.Rnd.Compare(msg.Rnd) == -1 {
		a.slots.Rnd = msg.Rnd
	}

	slot.VRnd = msg.Rnd
	slot.VVal = msg.Val

	return &px.Learn{
		Slot: slot.ID,
		ID:   a.id,
		Rnd:  a.slots.Rnd,
		Val:  slot.VVal,
	}
}

func (a *MultiAcceptor) handleAccAr(msg *px.Accept) *px.Learn {
	if !a.processing {
		a.processing = true
		elog.Log(e.NewEvent(e.Processing))
	}

	if msg.Slot < a.lowSlot {
		return nil
	}

	if msg.ID.Epoch != a.id.Epoch {
		return nil
	}

	slot := a.slots.GetSlot(msg.Slot)

	// If the round number for the Accept is lower or equal to the highest
	// one in which we have participated, or the message round is eqaual to
	// vrnd: ignore.
	if a.slots.Rnd.Compare(msg.Rnd) == 1 || msg.Rnd.Compare(slot.VRnd) == 0 {
		return nil
	}

	// If the round number for the Accept for some reason is higher than
	// the highest one in which we have participated: update our round
	// variable to this value.
	if a.slots.Rnd.Compare(msg.Rnd) == -1 {
		a.slots.Rnd = msg.Rnd
	}

	slot.VRnd = msg.Rnd
	slot.VVal = msg.Val

	return &px.Learn{
		Slot: slot.ID,
		ID:   a.id,
		Rnd:  a.slots.Rnd,
		Val:  slot.VVal,
	}
}

func (a *MultiAcceptor) handleAccLr(msg *px.Accept) *px.Learn {
	if !a.processing {
		a.processing = true
		elog.Log(e.NewEvent(e.Processing))
	}

	if msg.Slot < a.lowSlot {
		return nil
	}

	slot := a.slots.GetSlot(msg.Slot)

	// If the round number for the Accept is lower or equal to the highest
	// one in which we have participated, or the message round is eqaual to
	// vrnd: ignore.
	if a.slots.Rnd.Compare(msg.Rnd) == 1 || msg.Rnd.Compare(slot.VRnd) == 0 {
		return nil
	}

	// If the round number for the Accept for some reason is higher than
	// the highest one in which we have participated: update our round
	// variable to this value.
	if a.slots.Rnd.Compare(msg.Rnd) == -1 {
		a.slots.Rnd = msg.Rnd
	}

	slot.VRnd = msg.Rnd
	slot.VVal = msg.Val

	return &px.Learn{
		Slot:        slot.ID,
		ID:          a.id,
		Rnd:         a.slots.Rnd,
		Val:         slot.VVal,
		EpochVector: a.grpmgr.Epochs(),
	}
}

// -----------------------------------------------------------------------
// Communication utilities

func (a *MultiAcceptor) send(msg interface{}, id grp.ID) {
	a.ucast <- net.Packet{DestID: id, Data: msg}
}

func (a *MultiAcceptor) broadcast(msg interface{}) {
	a.bcast <- msg
}

// -----------------------------------------------------------------------
// Acceptor state

// SetState sets the state of the accetor to the slot map contained in slots.
func (a *MultiAcceptor) SetState(slots *px.AcceptorSlotMap) {
	a.slots = slots
}

// GetState returns a new slot map with all acceptor state over afterSlot. The
// acceptor will pause running until the provided release channel is closed.
func (a *MultiAcceptor) GetState(afterSlot px.SlotID,
	release <-chan bool) *px.AcceptorSlotMap {
	responseChan := make(chan *px.AcceptorSlotMap)
	a.stateReqChan <- acceptorStateRequest{afterSlot, responseChan, release}
	return <-responseChan
}

type acceptorStateRequest struct {
	afterSlot    px.SlotID
	responseChan chan<- *px.AcceptorSlotMap
	release      <-chan bool
}

func (a *MultiAcceptor) handleStateRequest(asr acceptorStateRequest) {
	slotMap := px.CopyAccSlotMapRange(asr.afterSlot, a.slots)
	asr.responseChan <- slotMap
	<-asr.release
	a.id = a.grpmgr.GetID()
}

// SetLowSlot sets the low slot of the acceptor. The acceptor will not respond
// to learns for slots below this slot.
func (a *MultiAcceptor) SetLowSlot(slot px.SlotID) {
	a.lowSlot = slot
}

// SlotRequest: Used for LR fast version
type acceptorSlotRequest struct {
	responseChan chan<- *px.AcceptorSlotMap
	release      <-chan bool
}

// GetMaxSlot returns a new empty slot map. The current round in the slot map
// is set to the acceptor's current round. The highest seen slot in the slot
// map is set to the acceptor's highest seen slot.  The acceptor will pause
// running until the provided release channel is closed.
//
// TODO: This method should be renamed to better indicate what it does.
func (a *MultiAcceptor) GetMaxSlot(release <-chan bool) *px.AcceptorSlotMap {
	responseChan := make(chan *px.AcceptorSlotMap)
	a.slotReqChan <- acceptorSlotRequest{responseChan, release}
	return <-responseChan
}

func (a *MultiAcceptor) handleSlotRequest(asr acceptorSlotRequest) {
	emptyMap := px.NewAcceptorSlotMap()
	emptyMap.MaxSeen = a.slots.MaxSeen
	emptyMap.Rnd = a.slots.Rnd
	asr.responseChan <- emptyMap
	<-asr.release
}
