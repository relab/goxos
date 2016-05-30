package parallelpaxos

import (
	"sync"

	"github.com/relab/goxos/config"
	"github.com/relab/goxos/grp"
	"github.com/relab/goxos/net"
	px "github.com/relab/goxos/paxos"

	"github.com/relab/goxos/Godeps/_workspace/src/github.com/golang/glog"
)

type acceptorStateRequest struct {
	afterSlot    px.SlotID
	responseChan chan<- *px.AcceptorSlotMap
	release      <-chan bool
}

type acceptorSlotRequest struct {
	responseChan chan<- *px.AcceptorSlotMap
	release      <-chan bool
}

type ParallelAcceptor struct {
	id          grp.ID
	started     bool
	startable   bool
	lowSlot     px.SlotID // The acceptor can't respond with learns for slots lower than this
	processors  int
	slots       *px.AcceptorSlotMap
	ucast       chan<- net.Packet
	bcast       chan<- interface{}
	prepareChan <-chan px.Prepare
	acceptChan  <-chan px.Accept

	// The round lock: Paxos requires some predictability
	// here. When the acceptor promises on a round number, it
	// should not learn accept messages with lower round
	// number. We cannot therefore do these tasks in parallel, but
	// for most of the time the round number is not changed and we
	// can handle accepts in parallel (read in parallel,
	// exclucively write)
	rndLock     sync.RWMutex
	getSlotLock sync.Mutex

	dmx          net.Demuxer
	grpmgr       grp.GroupManager
	stateReqChan chan acceptorStateRequest
	slotReqChan  chan acceptorSlotRequest
	stop         chan bool
	stopCheckIn  *sync.WaitGroup
}

// Create a new ParallelAcceptor with the help of a PaxosPack.
func NewParallelAcceptor(pp *px.Pack) *ParallelAcceptor {
	ma := &ParallelAcceptor{
		id:           pp.ID,
		startable:    (pp.RunAcc && pp.RunLrn),
		lowSlot:      pp.NextExpectedDcd,
		processors:   pp.Config.GetInt("parallelPaxosAcceptors", config.DefParallelPaxosAcceptors),
		slots:        px.NewAcceptorSlotMap(),
		ucast:        pp.Ucast,
		bcast:        pp.Bcast,
		dmx:          pp.Dmx,
		grpmgr:       pp.Gm,
		stateReqChan: make(chan acceptorStateRequest),
		slotReqChan:  make(chan acceptorSlotRequest),
		stop:         make(chan bool),
		stopCheckIn:  pp.StopCheckIn,
	}

	// Set lowSlot to maxSeen in the slot map
	ma.slots.MaxSeen = ma.lowSlot

	return ma
}

// Start listening for ParallelPaxos messages from other replicas. Registers
// interest for PrepareMsg and AcceptMsg.
func (a *ParallelAcceptor) Start() {
	if !a.startable || a.started {
		glog.Warning("ignoring start request")
		return
	}

	glog.V(1).Infof("starting %d processors", a.processors)
	a.started = true
	a.registerChannels()

	for i := 0; i < a.processors; i++ {
		go a.processor(i)
	}
}

// Stops all Acceptor processors.
func (a *ParallelAcceptor) Stop() {
	if a.started {
		a.started = false
		for i := 0; i < a.processors; i++ {
			a.stop <- true
		}
		a.stopCheckIn.Done()
	}
}

// Processor should be started as a new go routine. Will return when
// signaled on a.stop channel. So when having n processors, you'll
// have to send n signals on the stop channel to stop all processors.
func (a *ParallelAcceptor) processor(i int) {
	// Bookkeeping of the new goroutine:
	a.stopCheckIn.Add(1)
	defer a.stopCheckIn.Done()

	glog.V(1).Info("Started acceptor processor ", i)
	defer glog.V(1).Info("Stopped acceptor processor ", i)

	for {
		select {
		case prepare := <-a.prepareChan:
			prom, dest := a.handlePrepare(&prepare)
			if prom != nil {
				a.send(*prom, dest)
			}
		case accept := <-a.acceptChan:
			a.handleAccept(&accept)
		case <-a.stop:
			return
		}
	}
}

func (a *ParallelAcceptor) registerChannels() {
	acceptChan := make(chan px.Accept, 256)
	a.acceptChan = acceptChan
	a.dmx.RegisterChannel(acceptChan)

	prepareChan := make(chan px.Prepare, 16)
	a.prepareChan = prepareChan
	a.dmx.RegisterChannel(prepareChan)
}

func (a *ParallelAcceptor) handlePrepare(msg *px.Prepare) (*px.Promise, grp.ID) {
	glog.V(3).Infoln("got prepare from", msg.ID, "with crnd",
		msg.CRnd, "and slot id", msg.Slot)

	// Write lock:
	a.rndLock.Lock()
	defer a.rndLock.Unlock()

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

func (a *ParallelAcceptor) handleAccept(msg *px.Accept) {
	if msg.Slot < a.lowSlot {
		glog.V(3).Infoln("Got a accept for lower than a.lowSlot.")
		return
	}

	a.rndLock.RLock()
	defer a.rndLock.RUnlock()

	a.getSlotLock.Lock()
	slot := a.slots.GetSlot(msg.Slot)
	a.getSlotLock.Unlock()

	// If the round number for the Accept is lower or equal to the highest
	// one in which we have participated, or the message round is eqaual to
	// vrnd: ignore.
	if a.slots.Rnd.Compare(msg.Rnd) == 1 {
		glog.V(3).Infoln("Got a accept for a lower slot than a.slots.Rnd")
		return
	}
	if msg.Rnd.Compare(slot.VRnd) == 0 {
		glog.V(3).Infoln("Got a new accept with same round number...")
		return
	}

	// If the round number for the Accept for some reason is higher than
	// the highest one in which we have participated: update our round
	// variable to this value.
	if a.slots.Rnd.Compare(msg.Rnd) == -1 {
		glog.V(3).Infoln("Slot number increased for some unexcepted reason")
		a.slots.Rnd = msg.Rnd
	}

	slot.VRnd = msg.Rnd
	slot.VVal = msg.Val

	glog.V(3).Infof("Going to broadcast a learn for slot %d", slot.ID)
	a.broadcast(px.Learn{
		Slot: slot.ID,
		ID:   a.id,
		Rnd:  a.slots.Rnd,
		Val:  msg.Val,
	})
	glog.V(3).Infof("Broadcasted a learn for slot %d", slot.ID)
}

// -----------------------------------------------------------------------
// Communication utilities

func (a *ParallelAcceptor) send(msg interface{}, id grp.ID) {
	a.ucast <- net.Packet{DestID: id, Data: msg}
}

func (a *ParallelAcceptor) broadcast(msg interface{}) {
	a.bcast <- msg
}

// -----------------------------------------------------------------------
// Acceptor state

// Set the state of the acceptor
func (a *ParallelAcceptor) SetState(slots *px.AcceptorSlotMap) {
	// a.slots = slots
}

// Request the state of the acceptor
func (a *ParallelAcceptor) GetState(afterSlot px.SlotID,
	release <-chan bool) *px.AcceptorSlotMap {
	responseChan := make(chan *px.AcceptorSlotMap)
	a.stateReqChan <- acceptorStateRequest{afterSlot, responseChan, release}
	return <-responseChan
}

func (a *ParallelAcceptor) GetMaxSlot(release <-chan bool) *px.AcceptorSlotMap {
	responseChan := make(chan *px.AcceptorSlotMap)
	a.slotReqChan <- acceptorSlotRequest{responseChan, release}
	return <-responseChan
}

func (a *ParallelAcceptor) SetLowSlot(slot px.SlotID) {
	a.lowSlot = slot
}
