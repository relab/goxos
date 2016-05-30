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

// A MultiLearner holds all of the state for a learner in MultiPaxos.
type MultiLearner struct {
	id                grp.ID
	startable         bool
	started           bool
	leader            grp.ID
	slots             *px.LearnerSlotMap
	slotsLr           *px.LearnerLrSlotMap
	dmx               net.Demuxer
	grpmgr            grp.GroupManager
	grpSubscriber     grp.Subscriber
	next              px.SlotID // Next expected decided slot
	ucast             chan<- net.Packet
	bcast             chan<- interface{}
	trust             <-chan grp.ID
	learnChan         <-chan px.Learn
	creqChan          <-chan px.CatchUpRequest
	crespChan         <-chan px.CatchUpResponse
	catchUpInProgress bool
	handleLearn       func(*px.Learn) (*px.Value, px.SlotID)
	learnValue        func(*px.Value, px.SlotID) (bool, bool, px.SlotID)
	advance           func() (*px.Value, px.SlotID)
	dcdChan           chan<- *px.Value
	getUndecidedSlots func(slot px.SlotID) []px.RangeTuple
	handleCatchUpReq  func(msg *px.CatchUpRequest) (*px.CatchUpResponse, grp.ID)
	handleCatchUpResp func(msg *px.CatchUpResponse)
	stop              chan bool
	stopCheckIn       *sync.WaitGroup
}

// NewMultiLearner returns a new learner based on the state in pp.
func NewMultiLearner(pp *px.Pack) *MultiLearner {
	ml := &MultiLearner{
		id:          pp.ID,
		startable:   pp.RunLrn,
		leader:      pp.Ld.PaxosLeader(),
		dmx:         pp.Dmx,
		grpmgr:      pp.Gm,
		next:        pp.NextExpectedDcd,
		ucast:       pp.Ucast,
		bcast:       pp.Bcast,
		trust:       pp.Ld.SubscribeToPaxosLdMsgs("learner"),
		dcdChan:     pp.DcdChan,
		stop:        make(chan bool),
		stopCheckIn: pp.StopCheckIn,
	}

	if !ml.grpmgr.LrEnabled() {
		ml.handleLearn = ml.handleLrn
		ml.learnValue = ml.learnVal
		ml.advance = ml.adv
		ml.getUndecidedSlots = ml.getUndecSlots
		ml.handleCatchUpReq = ml.handleCUReq
		ml.handleCatchUpResp = ml.handleCUResp
		ml.slots = px.NewLearnerSlotMap()
	} else {
		ml.handleLearn = ml.handleLrnLr
		ml.learnValue = ml.learnValLr
		ml.advance = ml.advLr
		ml.getUndecidedSlots = ml.getUndecSlotsLr
		ml.handleCatchUpReq = ml.handleCUReqLr
		ml.handleCatchUpResp = ml.handleCURespLr
		ml.slotsLr = px.NewLearnerLrSlotMap()
	}

	return ml
}

// Start starts the learner.
func (l *MultiLearner) Start() {
	if !l.startable || l.started {
		glog.Warning("ignoring start request")
		return
	}

	glog.V(1).Info("starting")
	l.started = true
	l.registerChannels()

	go func() {
		defer l.stopCheckIn.Done()
		for {
			select {
			case learn := <-l.learnChan:
				value, slotID := l.handleLearn(&learn)
				advance, startcu, cuslot := l.learnValue(value, slotID)
				switch {
				case advance:
					for dcdVal, slotID := l.advance(); dcdVal != nil; dcdVal, slotID = l.advance() {
						l.dcdChan <- dcdVal
						l.next = slotID + 1
					}
				case startcu:
					if l.catchUpInProgress || l.id == l.leader {
						break
					}
					l.catchUpInProgress = true
					elog.Log(e.NewEvent(e.CatchUpMakeReq))
					creq, dest := l.genCatchUpReq(cuslot)
					l.send(creq, dest)
					elog.Log(e.NewEvent(e.CatchUpSentReq))
				}
			case creq := <-l.creqChan:
				elog.Log(e.NewEvent(e.CatchUpRecvReq))
				cresp, dest := l.handleCatchUpReq(&creq)
				elog.Log(e.NewEvent(e.CatchUpSentResp))
				l.send(cresp, dest)
			case cresp := <-l.crespChan:
				elog.Log(e.NewEvent(e.CatchUpRecvResp))
				l.handleCatchUpResp(&cresp)
				elog.Log(e.NewEvent(e.CatchUpDoneHandlingResp))
				l.catchUpInProgress = false
				for dcdVal, slotID := l.advance(); dcdVal != nil; dcdVal, slotID = l.advance() {
					l.dcdChan <- dcdVal
					l.next = slotID + 1
				}
			case trustID := <-l.trust:
				l.leader = trustID
			case grpPrepare := <-l.grpSubscriber.PrepareChan():
				l.handleGrpHold(grpPrepare)
			case <-l.stop:
				l.started = false
				glog.V(1).Info("exiting")
				return
			}
		}
	}()
}

// Stop stops the learner.
func (l *MultiLearner) Stop() {
	if l.started {
		l.stop <- true
	}
}

func (l *MultiLearner) registerChannels() {
	learnChan := make(chan px.Learn, 64)
	l.learnChan = learnChan
	l.dmx.RegisterChannel(learnChan)
	creqChan := make(chan px.CatchUpRequest, 8)
	l.creqChan = creqChan
	l.dmx.RegisterChannel(creqChan)
	crespChan := make(chan px.CatchUpResponse, 8)
	l.crespChan = crespChan
	l.dmx.RegisterChannel(crespChan)
	if l.grpmgr.ArEnabled() {
		l.grpSubscriber = l.grpmgr.SubscribeToHold("Learner")
	}
}

func (l *MultiLearner) handleLrn(msg *px.Learn) (*px.Value, px.SlotID) {
	if glog.V(3) {
		glog.Infof("got learn from %v for slot %d", msg.ID, msg.Slot)
	}

	slot := l.slots.GetSlot(msg.Slot)

	if slot.Learned {
		return nil, 0
	}

	switch {
	case slot.Rnd.Compare(msg.Rnd) == 1:
		// Round in learn is lower, ignore
		return nil, 0
	case slot.Rnd.Compare(msg.Rnd) == -1:
		// Round in learn is bigger, reset and initialize, then
		// fallthrough. Note that this case always will be true for the
		// first learn received for a new slot.
		slot.Rnd = msg.Rnd
		slot.Learns = make([]*px.Learn, l.grpmgr.NrOfNodes())
		slot.Votes = 0
		fallthrough
	case slot.Rnd.Compare(msg.Rnd) == 0:
		if slot.Learns[msg.ID.PxInt()] != nil {
			// If we already have a learn with same round from the
			// replica: ignore.
			return nil, 0
		}

		slot.Votes++
		slot.Learns[msg.ID.PxInt()] = msg

		// Quorum?
		if slot.Votes >= l.grpmgr.Quorum() {
			return &msg.Val, slot.ID
		}
	}

	return nil, 0
}

func (l *MultiLearner) handleLrnLr(msg *px.Learn) (*px.Value, px.SlotID) {
	if glog.V(3) {
		glog.Infof("got learn from %v for slot %d", msg.ID, msg.Slot)
	}

	slot := l.slotsLr.GetSlot(msg.Slot)

	if slot.Learned {
		return nil, 0
	}

	// Only check the epoch vectors if valid quorum chekcing is not already
	// enabled for this slot
	if !slot.CheckVQ {
		// If we see a EpochVector different from ours, turn on valid
		// quorum checking for this slot.
		if !grp.EpochSlicesEqual(msg.EpochVector, l.grpmgr.Epochs()) {
			if glog.V(3) {
				glog.Infoln(
					"epoch vectors differ:",
					msg.EpochVector, l.grpmgr.Epochs(),
				)
			}
			slot.CheckVQ = true
		}
	}

	switch {

	case slot.Rnd.Compare(msg.Rnd) == 1:
		// Round in learn is lower, ignore
		return nil, 0
	case slot.Rnd.Compare(msg.Rnd) == -1:
		// Round in learn is bigger, reset and initalize, then
		// fallthrough. Note that this case will be true for the first
		// learn received for a specific slot.
		slot.Rnd = msg.Rnd
		slot.Learns = make([]*px.Learn, 0, l.grpmgr.NrOfNodes())
		slot.Votes = 0
		slot.Voted = make(map[grp.ID]bool)
		fallthrough
	case slot.Rnd.Compare(msg.Rnd) == 0:
		if slot.Voted[msg.ID] {
			// We already have a learn with same round from the
			// replica: ignore.
			return nil, 0
		}

		slot.Votes++
		slot.Voted[msg.ID] = true
		slot.Learns = append(slot.Learns, msg)

		// May have a quorum
		if slot.Votes < l.grpmgr.Quorum() {
			return nil, 0
		}

		if !slot.CheckVQ {
			// Fast path: we have learned this slot
			return &msg.Val, slot.ID
		}

		if glog.V(3) {
			glog.Info("vq-check enabled for this slot")
		}
		vq, vqmsgs := extractVqLearns(slot.Learns, l.grpmgr.Quorum())
		if vq {
			// Just pick the first valid one
			msg := slot.Learns[vqmsgs[0]]
			return &msg.Val, slot.ID
		}
	}

	return nil, 0
}

func (l *MultiLearner) learnVal(val *px.Value, slotID px.SlotID) (
	advance, startcu bool, cuslot px.SlotID) {
	if val == nil {
		return false, false, 0
	}
	if glog.V(3) {
		glog.Infof("learned slot %d", slotID)
	}
	slot := l.slots.GetSlot(slotID)
	slot.Learned = true
	slot.LearnedVal = *val
	if slot.ID == l.next {
		return true, false, 0
	} else if slot.ID > l.next {
		// Gap in range of learned messages
		return false, true, slot.ID
	} else {
		panic("unhandled case in learnValue")
	}
}

func (l *MultiLearner) learnValLr(val *px.Value, slotID px.SlotID) (
	advance, startcu bool, cuslot px.SlotID) {
	if val == nil {
		return false, false, 0
	}
	if glog.V(3) {
		glog.Infof("learned slot %d", slotID)
	}
	slot := l.slotsLr.GetSlot(slotID)
	slot.Learned = true
	slot.LearnedVal = *val
	if slot.ID == l.next {
		return true, false, 0
	} else if slot.ID > l.next {
		// Gap in range of learned messages
		return false, true, slot.ID
	} else {
		panic("unhandled case in learnValueLr")
	}
}

func (l *MultiLearner) adv() (*px.Value, px.SlotID) {
	slot := l.slots.GetSlot(l.next)
	if slot.Learned && !slot.Decided {
		// If we've received quorum of learns, but haven't yet sent it
		// to the application.
		if glog.V(3) {
			glog.Info("advancing next expected slot")
		}
		slot.Decided = true
		return &slot.LearnedVal, l.next
	}

	// Next is undecided
	return nil, 0
}

func (l *MultiLearner) advLr() (*px.Value, px.SlotID) {
	if glog.V(3) {
		glog.Info("advancing next expected slot")
	}
	slot := l.slotsLr.GetSlot(l.next)
	if slot.Learned && !slot.Decided {
		// If we've received quorum of learns, but haven't yet sent it
		// to the application.
		slot.Decided = true
		return &slot.LearnedVal, l.next
	}

	// Next is undecided
	return nil, 0
}

// -----------------------------------------------------------------------
// Catch-up (Common)

func (l *MultiLearner) genCatchUpReq(slot px.SlotID) (*px.CatchUpRequest, grp.ID) {
	glog.V(2).Info("generating catch up request")

	// List of undecided slots as ranges (from, to)
	rngs := l.getUndecidedSlots(slot)

	glog.V(2).Info("catch up ranges are:")
	for _, rng := range rngs {
		glog.V(2).Infof("[%d, %d]", rng.From, rng.To)
	}

	glog.V(2).Infoln("catch up request destination is set to:", l.leader)
	return &px.CatchUpRequest{ID: l.id, Ranges: rngs}, l.leader
}

// -----------------------------------------------------------------------
// Catch-up (Reg)

func (l *MultiLearner) getUndecSlots(slot px.SlotID) []px.RangeTuple {
	var ts []px.RangeTuple
	t := px.RangeTuple{}
	start := false

	for i := l.next; i < slot; i++ {
		slot := l.slots.GetSlot(i)
		if !slot.Decided {
			if !start {
				start = true
				t.From = i
			}
		} else {
			if start {
				t.To = i - 1
				ts = append(ts, t)
				t = px.RangeTuple{}
				start = false
			}
		}
	}

	if start {
		// Handle case where we ended in undecided slots
		t.To = slot - 1
		ts = append(ts, t)
	}

	return ts
}

func (l *MultiLearner) handleCUReq(msg *px.CatchUpRequest) (*px.CatchUpResponse, grp.ID) {
	glog.V(2).Infoln("got catch up request from", msg.ID)
	for _, rng := range msg.Ranges {
		glog.V(2).Infof("range [%d, %d]", rng.From, rng.To)
	}

	var ts []px.ResponseTuple

	for _, rng := range msg.Ranges {
		for i := rng.From; i < rng.To+1; i++ {
			slot := l.slots.GetSlot(i)
			if slot.Decided {
				ts = append(ts, px.ResponseTuple{
					Slot: i,
					Val:  slot.LearnedVal})
			}
		}
	}

	return &px.CatchUpResponse{ID: l.id, Vals: ts}, msg.ID
}

func (l *MultiLearner) handleCUResp(msg *px.CatchUpResponse) {
	if len(msg.Vals) == 0 {
		glog.V(2).Infoln("catch up response from", msg.ID, "was empty")
		return
	}
	glog.V(2).Infoln(
		"got catch up response from", msg.ID, "with", len(msg.Vals),
		"decided slots, range:", msg.Vals[0].Slot, "-",
		msg.Vals[len(msg.Vals)-1].Slot,
	)
	for _, dec := range msg.Vals {
		slot := l.slots.GetSlot(dec.Slot)
		// If the slot hasn't already been learned, update it
		if !slot.Learned {
			slot.Learned = true
			slot.LearnedVal = dec.Val
		}
	}
}

// -----------------------------------------------------------------------
// Catch-up (LR)

func (l *MultiLearner) getUndecSlotsLr(slot px.SlotID) []px.RangeTuple {
	var ts []px.RangeTuple
	t := px.RangeTuple{}
	start := false

	for i := l.next; i < slot; i++ {
		slot := l.slotsLr.GetSlot(i)
		if !slot.Decided {
			if !start {
				start = true
				t.From = i
			}
		} else {
			if start {
				t.To = i - 1
				ts = append(ts, t)
				t = px.RangeTuple{}
				start = false
			}
		}
	}

	if start {
		// Handle case where we ended in undecided slots
		t.To = slot - 1
		ts = append(ts, t)
	}

	return ts
}

func (l *MultiLearner) handleCUReqLr(msg *px.CatchUpRequest) (*px.CatchUpResponse, grp.ID) {
	glog.V(2).Infoln("got catch up request from", msg.ID)
	for _, rng := range msg.Ranges {
		glog.V(2).Infof("range [%d, %d]", rng.From, rng.To)
	}

	var ts []px.ResponseTuple

	for _, rng := range msg.Ranges {
		for i := rng.From; i < rng.To+1; i++ {
			slot := l.slotsLr.GetSlot(i)
			if slot.Decided {
				ts = append(ts, px.ResponseTuple{
					Slot: i,
					Val:  slot.LearnedVal})
			}
		}
	}

	return &px.CatchUpResponse{ID: l.id, Vals: ts}, msg.ID
}

func (l *MultiLearner) handleCURespLr(msg *px.CatchUpResponse) {
	if len(msg.Vals) == 0 {
		glog.V(2).Infoln("catch up response from", msg.ID, "was empty")
		return
	}
	glog.V(2).Infoln(
		"got catch up response from", msg.ID, "with", len(msg.Vals),
		"decided slots, range:", msg.Vals[0].Slot, "-",
		msg.Vals[len(msg.Vals)-1].Slot,
	)
	for _, dec := range msg.Vals {
		slot := l.slotsLr.GetSlot(dec.Slot)
		// If the slot hasn't already been learned, update it
		if !slot.Learned {
			slot.Learned = true
			slot.LearnedVal = dec.Val
		}
	}
}

// -----------------------------------------------------------------------
// Communication utilities

func (l *MultiLearner) send(msg interface{}, id grp.ID) {
	l.ucast <- net.Packet{DestID: id, Data: msg}
}

// -----------------------------------------------------------------------
// Group Manager Hold

func (l *MultiLearner) handleGrpHold(gp *sync.WaitGroup) {
	glog.V(2).Info("grpmgr hold request")
	gp.Done()
	<-l.grpSubscriber.ReleaseChan()
	glog.V(2).Info("grpmgr release")
	l.id = l.grpmgr.GetID()
}
