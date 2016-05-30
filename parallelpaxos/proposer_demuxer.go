package parallelpaxos

import (
	"sync"

	"github.com/relab/goxos/grp"
	px "github.com/relab/goxos/paxos"

	"github.com/relab/goxos/Godeps/_workspace/src/github.com/golang/glog"
)

// We recieved a signal about a desicion on a new leader. If this is
// us, we'll need to start proposing.
func (p *ParallelProposer) demuxerHandleTrust(id grp.ID) {
	if p.leader == p.id && p.leader == id {
		glog.V(2).Info("I was reelected as leader")
		return
	}

	p.leader = id

	if p.leader == p.id {
		glog.V(2).Info("I'm elected as leader")
		p.demuxerStartPhase1()
	}
}

// Resets state and sends prepare
func (p *ParallelProposer) demuxerStartPhase1() {
	p.phase1done = false
	p.phase1numPromises = 0
	p.phase1promises = make([]*px.Promise, p.grpmgr.NrOfNodes())

	p.crnd.Next()
	prepare := &px.Prepare{
		ID:   p.id,
		CRnd: *p.crnd,
		Slot: px.SlotID(p.adu),
	}
	p.bcast <- *prepare
}

func (p *ParallelProposer) demuxerHandlePromise(promise *px.Promise) {
	if p.phase1done {
		glog.V(3).Info("Ignoring a promise. Already in phase 2")
		return
	}

	if p.crnd.Compare(promise.Rnd) != 0 {
		glog.V(3).Info("Ignoring a promise to a lower round number")
		return
	}

	if p.phase1promises[promise.ID.PxInt()] != nil {
		glog.V(3).Info("Ignoring a promise. Already got one from that client.")
		return
	}

	glog.V(3).Infof("Got promise with data for %d slots from %v",
		len(promise.AccSlots), promise.ID)

	p.phase1promises[promise.ID.PxInt()] = promise
	p.phase1numPromises++

	if p.phase1numPromises >= p.grpmgr.Quorum() {
		glog.V(2).Info("Got quorum of promises")
		p.demuxerFinishPhase1()
	}
}

// Copies slots from promises to our slot map and finishes phase 1
func (p *ParallelProposer) demuxerFinishPhase1() {
	var wg sync.WaitGroup
	promisesMsg := &PromisesMsg{
		promises: p.phase1promises,
		wg:       &wg,
	}

	for _, ch := range p.processPromises {
		wg.Add(1)
		ch <- promisesMsg
	}

	wg.Wait()

	p.phase1done = true
	p.demuxerSignalToSendAccepts()
}

// If in phase 1: Start over again with increased round number
func (p *ParallelProposer) demuxerHandlePhaseTick() {
	if p.leader != p.id || p.phase1done {
		return
	}

	glog.V(3).Info("timeout: phase 1 not complete, retrying")
	p.demuxerStartPhase1()
}

func (p *ParallelProposer) demuxerHandleNewRequest(value *px.Value) {
	if p.leader != p.id {
		return
	}

	select {
	case p.reqQueue <- value:
		p.reqQueueLength++
	default:
		glog.Warning("Dropping a request. Queue in proposer out of space.")
		return
	}

	if p.leader == p.id && p.phase1done {
		p.demuxerSignalToSendAccepts()
	}
}

// Send the decided message to correct processor
func (p *ParallelProposer) demuxerDemuxDecided(slotID uint) {
	p.processDecided[slotID%p.numProcessors] <- slotID
}

// Handle message about a processor having increased its adu
func (p *ParallelProposer) demuxerHandleAdvance(msg *AduMsg) {
	// Was this processor the processor with the lowest adu?
	aduIncreased := true
	for _, adu := range p.aduInProcessor {
		if adu < p.aduInProcessor[msg.processor] {
			aduIncreased = false
			break
		}
	}

	// Anyway: store the new processor adu
	p.aduInProcessor[msg.processor] = msg.adu

	if aduIncreased {
		p.adu = msg.adu
		glog.V(3).Info("Advanced adu to ", p.adu)

		if p.leader == p.id && p.phase1done {
			p.demuxerSignalToSendAccepts()
		}
	}
}

// Signal the processors to start sending accepts
func (p *ParallelProposer) demuxerSignalToSendAccepts() {
	msg := &SendAcceptsMsg{
		crnd: *p.crnd,
	}

	if (p.adu + p.alpha) < (p.next + p.reqQueueLength - 1) {
		msg.upTo = p.adu + p.alpha
	} else {
		msg.upTo = p.next + p.reqQueueLength - 1
	}

	glog.V(3).Infof("Signaling to send accepts up to %d...", msg.upTo)

	for _, ch := range p.sendAccepts {
		go func(dest chan<- *SendAcceptsMsg) {
			dest <- msg
		}(ch)
	}

	p.reqQueueLength -= msg.upTo - p.next
	p.next += (msg.upTo - p.next)
}
