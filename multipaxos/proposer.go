package multipaxos

import (
	"container/list"
	"fmt"
	"sync"
	"time"

	"github.com/relab/goxos/config"
	"github.com/relab/goxos/grp"
	"github.com/relab/goxos/net"
	px "github.com/relab/goxos/paxos"

	"github.com/relab/goxos/Godeps/_workspace/src/github.com/golang/glog"
)

const (
	phaseOneTimeout                 = 500 * time.Millisecond
	phaseTwoTimeout                 = 50 * time.Millisecond
	resendsBeforeRestartingPhaseOne = 10
)

// A MultiProposer contains all the state for a proposer in MultiPaxos.
type MultiProposer struct {
	id               grp.ID
	startable        bool
	started          bool
	leader           grp.ID
	dmx              net.Demuxer
	grpmgr           grp.GroupManager
	grpSubscriber    grp.Subscriber
	alpha            uint
	crnd             *px.ProposerRound // Current round, spans across slots
	nextSlot         px.SlotID         // The next slot for an accept
	aru              *px.Adu
	adu              px.SlotID              // All-decided-up-to: highest decided slot
	slots            *px.ProposerSlotMap    // Proposer slot map
	reqQueue         *list.List             // Client request queue
	handlePromise    func(*px.Promise) bool // Method for handling promises (dependent on LR on/off)
	phaseOneVQCheck  bool                   // VQ-check for Phase One if LR is enabled
	phaseOnePromises []*px.Promise          // Stored promise messages
	phaseOneRecvFrom map[grp.ID]bool        // Has replica sent promise (LR)
	phaseOneCount    uint                   // Number of promises received for current crnd
	phaseOneDone     bool                   // Phase 1 completed?
	phaseOneTimer    *time.Timer            // Phase One progress check timer
	phaseTwoTimer    *time.Timer            // Phase Two progress check timer
	ucast            chan<- net.Packet      // Unicast channel
	bcast            chan<- interface{}     // Broadcast channel
	trust            <-chan grp.ID
	promiseChan      <-chan px.Promise
	newDcdChan       <-chan bool
	propChan         <-chan *px.Value
	stopCheckIn      *sync.WaitGroup
	stop             chan bool
}

// NewMultiProposer returns a new proposer based on the state in pp.
func NewMultiProposer(pp *px.Pack) *MultiProposer {
	mp := &MultiProposer{
		id:          pp.ID,
		startable:   pp.RunProp,
		leader:      pp.Ld.PaxosLeader(),
		dmx:         pp.Dmx,
		grpmgr:      pp.Gm,
		alpha:       uint(pp.Config.GetInt("alpha", config.DefAlpha)),
		crnd:        px.NewProposerRound(pp.ID),
		nextSlot:    pp.FirstSlot,
		aru:         pp.LocalAdu,
		adu:         pp.FirstSlot - 1,
		slots:       px.NewProposerSlotMap(),
		reqQueue:    list.New(),
		ucast:       pp.Ucast,
		bcast:       pp.Bcast,
		trust:       pp.Ld.SubscribeToPaxosLdMsgs("proposer"),
		newDcdChan:  pp.NewDcdChan,
		propChan:    pp.PropChan,
		stopCheckIn: pp.StopCheckIn,
		stop:        make(chan bool),
	}

	if !mp.grpmgr.LrEnabled() {
		mp.handlePromise = mp.handleProm
	} else {
		mp.handlePromise = mp.handlePromLr
	}

	return mp
}

// Start starts the proposer.
func (p *MultiProposer) Start() {
	if !p.startable || p.started {
		glog.Warning("ignoring start request")
		return
	}

	glog.V(1).Info("starting")
	glog.V(1).Infoln("status:", p.getStatus())

	p.registerChannels()

	p.phaseOneTimer = time.NewTimer(phaseOneTimeout)
	p.phaseTwoTimer = time.NewTimer(phaseTwoTimeout)

	if p.grpmgr.ArEnabled() {
		p.grpSubscriber = p.grpmgr.SubscribeToHold("proposer")
	}

	go func() {
		defer p.stopCheckIn.Done()
		for {
			select {
			// Decided progress from Server
			case <-p.newDcdChan:
				// Reset the resend accept timer, we only check
				// progress after periods of inactivity.
				p.phaseTwoTimer.Reset(phaseTwoTimeout)
				p.advanceAdu()
				if !p.isLeaderAndPhaseOneComplete() {
					break
				}
				p.sendAccept()
			// Values received from clients
			case val := <-p.propChan:
				// If we're not leader, drop request
				if p.leader != p.id {
					break
				}
				p.reqQueue.PushBack(val)
				if p.phaseOneDone {
					p.sendAccept()
				}
			// Trust messages from leader detector
			case trustID := <-p.trust:
				// The check below will prevent us from
				// starting a new round if we receive a trust
				// message for ourselves while we already see
				// ourselves as the leader.
				if p.leader == trustID {
					glog.V(2).Infof(
						"trust message for ",
						"current leader (%v), ",
						"ignoring...", trustID,
					)
					break
				}
				p.leader = trustID
				if p.leader != p.id {
					break
				}
				glog.V(2).Info(
					"we're elected leader ",
					"starting Phase 1...",
				)
				p.startPhaseOne(true)
			// Promise messages from peers
			case promise := <-p.promiseChan:
				quorum := p.handlePromise(&promise)
				if !quorum {
					break
				}
				p.setStateFromPromises()
				p.phaseOneDone = true
				glog.V(2).Info("phase 1 complete")
				p.nextSlot = p.adu + 1
				p.sendAccept()
			// Phase 1 progress timeout
			case <-p.phaseOneTimer.C:
				if p.leader != p.id || p.phaseOneDone {
					break
				}
				glog.V(2).Info(
					"timeout: phase 1 not complete, ",
					"retrying...",
				)
				p.startPhaseOne(true)
			// Resend accept timeout
			case <-p.phaseTwoTimer.C:
				p.resendAcceptsIfNecessary()
			// Group manager hold
			case grpPrepare := <-p.grpSubscriber.PrepareChan():
				p.handleGrpHold(grpPrepare)
			// Stop signal
			case <-p.stop:
				p.started = false
				glog.V(1).Info("exiting")
				return
			}
		}
	}()
}

// Stop stops the proposer.
func (p *MultiProposer) Stop() {
	if p.startable {
		p.stop <- true
	}
}

func (p *MultiProposer) registerChannels() {
	promiseChan := make(chan px.Promise, p.grpmgr.NrOfNodes())
	p.promiseChan = promiseChan
	p.dmx.RegisterChannel(promiseChan)
}

// -----------------------------------------------------------------------
// Phase 1: Common methods for both regular operation and Live Replacement

func (p *MultiProposer) startPhaseOne(clearRequestQueue bool) {
	p.updateStatePrePhaseOne(clearRequestQueue)
	p.broadcast(px.Prepare{
		ID:   p.id,
		CRnd: *p.crnd,
		Slot: p.adu,
	})
	p.phaseOneTimer.Reset(phaseOneTimeout)
}

func (p *MultiProposer) updateStatePrePhaseOne(clearRequestQueue bool) {
	p.resetPhaseOneData()
	p.resetSentCountersInAlphaWindow()
	p.crnd.Next()
	if clearRequestQueue {
		p.reqQueue.Init()
	}
}

func (p *MultiProposer) resetPhaseOneData() {
	p.phaseOneDone = false
	p.phaseOneCount = 0
	if p.grpmgr.LrEnabled() {
		p.phaseOneVQCheck = false
		p.phaseOnePromises = make([]*px.Promise, 0)
		p.phaseOneRecvFrom = make(map[grp.ID]bool)
	} else {
		p.phaseOnePromises = make([]*px.Promise, p.grpmgr.NrOfNodes())
	}
}

func (p *MultiProposer) setStateFromPromises() {
	for _, promise := range p.phaseOnePromises {
		if promise != nil {
			p.setStateFromPromise(promise)
		}
	}
}

func (p *MultiProposer) setStateFromPromise(promise *px.Promise) {
	for _, accSlot := range promise.AccSlots {
		// Should not happen, but lets check it anyway
		if accSlot.ID <= p.adu {
			continue
		}

		slot := p.slots.GetSlot(accSlot.ID)
		if slot.Proposal == nil {
			// We can set it directly
			slot.Proposal = &px.Accept{
				Rnd: accSlot.VRnd,
				Val: accSlot.VVal,
			}
			continue
		}

		// We must compare round
		if slot.Proposal.Rnd.Compare(accSlot.VRnd) >= 0 {
			// Lower: ignore
			continue
		}
		// Higher: set new value
		slot.Proposal = &px.Accept{
			Rnd: accSlot.VRnd,
			Val: accSlot.VVal,
		}
	}
}

// -----------------------------------------------------------------------
// Phase 1: Sending prepares and handling promises

func (p *MultiProposer) handleProm(msg *px.Promise) (quorum bool) {
	/* Return early if:
	1. Phase 1 is done
	2. CRnd does not match our current one
	3. We have already received a promise from the replica with equal CRnd
	*/
	if p.phaseOneDone {
		return false
	}
	if p.crnd.Compare(msg.Rnd) != 0 {
		return false
	}
	if p.phaseOnePromises[msg.ID.PxInt()] != nil {
		return false
	}

	glog.V(3).Infof(
		"got composite promise (size=%d) from %v",
		len(msg.AccSlots),
		msg.ID,
	)

	p.phaseOnePromises[msg.ID.PxInt()] = msg
	p.phaseOneCount++

	if p.phaseOneCount < p.grpmgr.Quorum() {
		return false
	}

	glog.V(3).Infof("received quorum of %d promises", p.phaseOneCount)

	return true
}

// -----------------------------------------------------------------------
// Phase 1: Sending prepares and handling promises with Live Replacement
// enabled

func (p *MultiProposer) handlePromLr(msg *px.Promise) (quorum bool) {
	/* Return early if:
	1. Phase 1 is done
	2. CRnd does not match our current one
	3. We already received a promise from the replica with equal CRnd
	*/
	if p.phaseOneDone {
		return false
	}
	if p.crnd.Compare(msg.Rnd) != 0 {
		return false
	}
	if p.phaseOneRecvFrom[msg.ID] {
		return false
	}

	glog.V(3).Infof(
		"got composite promise (size=%d) from %v",
		len(msg.AccSlots),
		msg.ID,
	)

	p.phaseOnePromises = append(p.phaseOnePromises, msg)
	p.phaseOneCount++

	// Only check the epoch vectors if valid quorum checking is not already
	// enabled for this slot
	if !p.phaseOneVQCheck {
		// If we see a EpochVector different from ours, turn on valid
		// quorum checking for phase 1
		if !grp.EpochSlicesEqual(msg.EpochVector, p.grpmgr.Epochs()) {
			if glog.V(3) {
				glog.Infoln("epoch vectors differ:",
					msg.EpochVector,
					p.grpmgr.Epochs(),
				)
			}
			p.phaseOneVQCheck = true
		}
	}

	// We can't have a valid quorum
	if p.phaseOneCount < p.grpmgr.Quorum() {
		return false
	}

	if !p.phaseOneVQCheck {
		// Fast path: we have a valid quorum of promises
		glog.V(3).Infoln(
			"received valid quorum of",
			p.phaseOneCount,
			"promises",
		)
		return true
	}

	// We must check for a valid quorum
	glog.V(3).Info("vq-check is enabled for phase 1")

	vq, vqmsgsidx := extractVqPromises(
		p.phaseOnePromises,
		p.grpmgr.Quorum(),
	)
	if !vq {
		return false
	}

	if len(vqmsgsidx) == len(p.phaseOnePromises) {
		// Every messages forms a valid quorum
		glog.V(3).Infoln(
			"received valid quorum of",
			vqmsgsidx,
			"promises",
		)
		return true
	}

	// Every promise is not a part of the valid quorum, we need to extract
	// the valid ones.
	onlyValidPromises := make([]*px.Promise, len(vqmsgsidx))
	for i, j := range vqmsgsidx {
		onlyValidPromises[i] = p.phaseOnePromises[j]
	}
	p.phaseOnePromises = onlyValidPromises

	glog.V(3).Infoln("received valid quorum of", vqmsgsidx, "promises")

	return true
}

// -----------------------------------------------------------------------
// Phase 2: Sending accepts

func (p *MultiProposer) sendAccept() {
	for {
		// Check that we're not restricted by alpha
		if !inAlphaRange(p.nextSlot, p.adu, p.alpha) {
			if glog.V(3) {
				glog.Info("sendAccept: we are restricted by alpha/adu/nextSlot, breaking")
			}
			break
		}

		ns := p.slots.GetSlot(p.nextSlot)
		acc := p.genAccept(ns)
		if acc != nil {
			p.broadcast(*acc)
			if glog.V(3) {
				glog.Info("sendAccept: accept broadcasted for slot: ", acc.Slot)
			}
			p.nextSlot++
			p.phaseTwoTimer.Reset(phaseTwoTimeout)
		} else {
			break
		}
	}
}

func (p *MultiProposer) genAccept(nextSlot *px.ProposerSlot) *px.Accept {
	var val *px.Value
	if nextSlot.Proposal != nil {
		val = &nextSlot.Proposal.Val
	} else if p.reqQueue.Len() == 0 {
		return nil
	} else {
		val = p.reqQueue.Front().Value.(*px.Value)
		p.reqQueue.Remove(p.reqQueue.Front())
	}

	accept := &px.Accept{
		ID:   p.id,
		Rnd:  *p.crnd,
		Slot: nextSlot.ID,
		Val:  *val,
	}

	nextSlot.Proposal = accept

	return accept
}

func (p *MultiProposer) advanceAdu() {
	p.adu++
	if glog.V(3) {
		glog.Infoln("received decided slot id, advanced adu to", p.adu)
	}
}

// -----------------------------------------------------------------------
// Resend accept methods

func (p *MultiProposer) resendAcceptsIfNecessary() {
	if !p.isLeaderAndPhaseOneComplete() {
		return
	}
	for i := p.adu + 1; i < p.nextSlot; i++ {
		slot := p.slots.GetSlot(i)
		p.resendAccept(slot.Proposal)
	}
}

func (p *MultiProposer) resendAccept(acc *px.Accept) {
	if acc != nil {
		if p.hasReachedResendThreshold(acc.Slot) {
			// Restart Phase One. False as argument because we
			// don't want to reset the client queue in this case.
			glog.V(2).Infoln(
				"slot", acc.Slot, "has reached resend limit, ",
				"starting phase 1...",
			)
			p.startPhaseOne(false)
			return
		}
		p.broadcast(acc)
		p.incrementSentCountFor(acc.Slot)
		p.phaseTwoTimer.Reset(phaseTwoTimeout)
		glog.V(3).Infoln("resending accept for slot: ", acc.Slot)
	}
}

func (p *MultiProposer) incrementSentCountFor(slot px.SlotID) {
	s := p.slots.GetSlot(slot)
	s.SentCount++
}

func (p *MultiProposer) hasReachedResendThreshold(slot px.SlotID) bool {
	s := p.slots.GetSlot(slot)
	return s.SentCount >= resendsBeforeRestartingPhaseOne
}

func (p *MultiProposer) resetSentCountersInAlphaWindow() {
	for i := p.adu + 1; i < p.nextSlot; i++ {
		slot := p.slots.GetSlot(i)
		slot.SentCount = 0
	}
}

// -----------------------------------------------------------------------
// Utility methods

func (p *MultiProposer) broadcast(msg interface{}) {
	p.bcast <- msg
}

func (p *MultiProposer) isLeaderAndPhaseOneComplete() bool {
	return (p.leader == p.id) && p.phaseOneDone
}

func (p *MultiProposer) getStatus() string {
	return fmt.Sprintf(
		"adu: %d, next slot: %d, alpha: %d",
		p.adu, p.nextSlot, p.alpha,
	)
}

func inAlphaRange(nextSlot, adu px.SlotID, alpha uint) bool {
	return uint(nextSlot) <= uint(adu)+alpha
}

// SetNextSlot sets the internal next slot variable in the proposer. Should not
// be used while running.
func (p *MultiProposer) SetNextSlot(ns px.SlotID) {
	if p.nextSlot > ns+px.SlotID(p.alpha) {
		ns = p.nextSlot - px.SlotID(p.alpha)
	}
	glog.V(2).Infof("moving next slot from %d to %d", p.nextSlot, ns)
	p.nextSlot = ns
}

func (p *MultiProposer) handleGrpHold(gp *sync.WaitGroup) {
	gp.Done()
	glog.V(2).Info("grpmgr hold request")
	<-p.grpSubscriber.ReleaseChan()
	glog.V(2).Info("grpmgr release")
	// ARec specific
	if p.leader == p.id {
		p.id = p.grpmgr.GetID()
		p.leader = p.id
	} else {
		p.id = p.grpmgr.GetID()
	}
	p.crnd.SetOwner(p.id)
}
