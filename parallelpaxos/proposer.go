package parallelpaxos

import (
	"fmt"
	"sync"
	"time"

	"github.com/relab/goxos/config"
	"github.com/relab/goxos/grp"
	"github.com/relab/goxos/net"
	px "github.com/relab/goxos/paxos"

	"github.com/relab/goxos/Godeps/_workspace/src/github.com/golang/glog"
)

// Time between proposal attemts:
const initialPhase1timeout = 500 * time.Millisecond

type ParallelProposer struct {
	//--------------------------
	// Paxos spesifics:
	//

	leader            grp.ID
	numProcessors     uint                   // the number of processors (parallel goroutines) we want in the proposer
	processDecided    []chan uint            // Signal from demuxer to processor
	sendAccepts       []chan *SendAcceptsMsg // Signal from demuxer to processor
	processPromises   []chan *PromisesMsg    // Signal from demuxer to processor to process phase1promises
	crnd              *px.ProposerRound      // current round number
	alpha             uint
	reqQueue          chan *px.Value // queue of client requests that we should process on phase one we'll have to store the promises we get
	reqQueueLength    uint
	phase1promises    []*px.Promise
	phase1numPromises uint
	phase1timeout     time.Duration // progress check interval
	phase1ticker      *time.Ticker  // and ticker for interval check
	phase1done        bool
	adu               uint // All decided up to
	aduInProcessor    []uint
	advanceChan       chan *AduMsg
	next              uint

	//--------------------------
	// Goxos spesifics:
	//

	id          grp.ID // unique id of this actor in the paxos network.
	startable   bool   // some replicas may be set up without a proposer, and it is then not startable
	started     bool
	dmx         net.Demuxer      // by registering channels with the demuxer we'll get all messages we're interested in
	grpmgr      grp.GroupManager // manages the group of replicas
	stopCheckIn *sync.WaitGroup  // Bookkeeping of goroutines
	stop        chan bool        // signal all goroutines to stop

	// Outgoing channels:
	ucast chan<- net.Packet
	bcast chan<- interface{}

	// Incoming channels
	trust       <-chan grp.ID
	promiseChan chan px.Promise
	dcdChan     <-chan px.SlotID
	propChan    <-chan *px.Value // incoming request from client
}

// Used to send Adu update messages from processors to demuxer
type AduMsg struct {
	processor uint
	adu       uint
}

// Used to command processors to process these promises
type PromisesMsg struct {
	promises []*px.Promise
	wg       *sync.WaitGroup
}

// Used to command processors to start sending accept messages up to a
// given slot id.
type SendAcceptsMsg struct {
	upTo uint
	crnd px.ProposerRound
}

func NewParallelProposer(pp *px.Pack) *ParallelProposer {
	proposer := &ParallelProposer{
		leader:         pp.Ld.PaxosLeader(),
		numProcessors:  uint(pp.Config.GetInt("parallelPaxosProposers", config.DefParallelPaxosProposers)),
		crnd:           px.NewProposerRound(pp.ID),
		alpha:          uint(pp.Config.GetInt("alpha", config.DefAlpha)),
		reqQueue:       make(chan *px.Value, pp.Config.GetInt("alpha", config.DefAlpha)*128),
		reqQueueLength: 0,
		phase1timeout:  initialPhase1timeout,
		phase1done:     false,
		next:           1,

		id:          pp.ID,
		startable:   pp.RunProp,
		dmx:         pp.Dmx,
		grpmgr:      pp.Gm,
		ucast:       pp.Ucast,
		bcast:       pp.Bcast,
		trust:       pp.Ld.SubscribeToPaxosLdMsgs("proposer"),
		promiseChan: make(chan px.Promise, pp.Gm.NrOfNodes()),
		dcdChan:     pp.DcdSlotIDToProp,
		propChan:    pp.PropChan,
		stopCheckIn: pp.StopCheckIn,
		stop:        make(chan bool),
	}

	proposer.dmx.RegisterChannel(proposer.promiseChan)
	proposer.advanceChan = make(chan *AduMsg)
	proposer.processDecided = make([]chan uint, proposer.numProcessors)
	proposer.sendAccepts = make([]chan *SendAcceptsMsg, proposer.numProcessors)
	proposer.processPromises = make([]chan *PromisesMsg, proposer.numProcessors)
	proposer.aduInProcessor = make([]uint, proposer.numProcessors)

	return proposer
}

// Starts the demuxer and the processors
func (p *ParallelProposer) Start() {
	if !p.startable || p.started {
		glog.V(1).Info("ignoring start request")
	}
	p.started = true

	glog.V(1).Info("starting")
	glog.V(1).Info("status: ", p.getStatus())

	p.phase1ticker = time.NewTicker(p.phase1timeout)

	// Start all processors
	for i := uint(0); i < p.numProcessors; i++ {
		p.processDecided[i] = make(chan uint)
		p.sendAccepts[i] = make(chan *SendAcceptsMsg)
		p.processPromises[i] = make(chan *PromisesMsg)
		p.startProcessor(i)
	}

	go p.startDemuxer()
}

// The demuxer takes totally care of phase 1 stuff (proposals,
// promises, timeouts etc.) And it demuxes all other events to the
// correct processor.
func (p *ParallelProposer) startDemuxer() {
	p.stopCheckIn.Add(1)
	defer p.stopCheckIn.Done()

	glog.V(1).Info("started proposer demuxer")
	defer glog.V(1).Info("stopped proposer demuxer")

	for {
		select {
		case trustID := <-p.trust:
			p.demuxerHandleTrust(trustID)
		case promise := <-p.promiseChan:
			p.demuxerHandlePromise(&promise)
		case <-p.phase1ticker.C:
			p.demuxerHandlePhaseTick()
		case val := <-p.propChan:
			p.demuxerHandleNewRequest(val)
		case dcd := <-p.dcdChan:
			p.demuxerDemuxDecided(uint(dcd))
		case aduMsg := <-p.advanceChan:
			p.demuxerHandleAdvance(aduMsg)
		case <-p.stop:
			return
		}
	}
}

// We can have multiple processors. Each is responsible for its part
// of the slotmap.
func (p *ParallelProposer) startProcessor(id uint) {
	p.stopCheckIn.Add(1)

	// TODO: Find a better way to solve the processors adu:
	var adu uint
	for ; adu <= p.adu; adu += p.numProcessors {
	}
	adu -= p.numProcessors

	next := id
	if (adu != 0) || (adu == 0 && id == 0) {
		next = adu + p.numProcessors
	}

	processor := ProposerProcessor{
		proposerID:    p.id,
		numProcessors: p.numProcessors,
		slots:         make(map[uint]*ProposerSlot),
		bcast:         p.bcast,
		reqQueue:      p.reqQueue,
		id:            id,
		adu:           adu,
		next:          next,
	}

	go func(pid uint) {
		glog.V(1).Info("started proposer processor ", pid)
		defer glog.V(1).Info("stopped proposer processor ", pid)
		defer p.stopCheckIn.Done()

		for {
			select {
			case dcd := <-p.processDecided[pid]:
				processor.HandleDecided(dcd, p.advanceChan)
			case msg := <-p.sendAccepts[pid]:
				processor.SendAccepts(msg)
			case msg := <-p.processPromises[pid]:
				processor.ProcessPromises(msg)
			case <-p.stop:
				return
			}
		}
	}(id)
}

func (p *ParallelProposer) Stop() {
	if p.started {
		glog.V(1).Info("signaling processors and demuxer to stop")

		for i := uint(0); i < (p.numProcessors + 1); i++ {
			p.stop <- true
		}
		p.stopCheckIn.Done()
		p.started = false
	}
}

func (p *ParallelProposer) getStatus() string {
	return fmt.Sprintf("adu: %d, alpha: %d",
		p.adu, p.alpha)
}

// We need to implement this, but this is a Asynchronous
// Reconfiguration (goxos/arec) feature that is not supported anyway.
func (p *ParallelProposer) SetNextSlot(ns px.SlotID) {
	panic("Arec not supported by parallelpaxos.")
}
