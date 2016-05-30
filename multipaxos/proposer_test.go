package multipaxos

import (
	"github.com/relab/goxos/grp"
	px "github.com/relab/goxos/paxos"

	gc "github.com/relab/goxos/Godeps/_workspace/src/gopkg.in/check.v1"
)

// -----------------------------------------------------------------------
// Initialization

type propSuite struct {
	psm1 *px.ProposerSlotMap
	psm2 *px.ProposerSlotMap
}

var _ = gc.Suite(&propSuite{})

// -----------------------------------------------------------------------
// Test: Sending prepare

func (*propSuite) TestResetPhaseOneData(c *gc.C) {
	// New proposer with next slot 1 and rnd (0,1)
	proposer := NewMultiProposer(ppThreeNodesNonLr)

	// Simulate completed Phase 1
	proposer.phaseOneDone = true
	proposer.phaseOneCount = 2
	proposer.phaseOnePromises = make(
		[]*px.Promise,
		proposer.grpmgr.NrOfNodes(),
	)
	proposer.phaseOnePromises[1] = &px.Promise{ID: r1id}
	proposer.phaseOnePromises[2] = &px.Promise{ID: r2id}

	// Reset Phase 1 data
	proposer.resetPhaseOneData()

	// Check that Phase 1 data is reset
	c.Assert(proposer.phaseOneDone, gc.Equals, false)
	c.Assert(proposer.phaseOneCount, gc.Equals, uint(0))
	c.Assert(proposer.phaseOnePromises, gc.HasLen, 3)
	c.Assert(proposer.phaseOnePromises[1], gc.IsNil)
	c.Assert(proposer.phaseOnePromises[2], gc.IsNil)

}

func (*propSuite) TestResetPhaseOneDataLr(c *gc.C) {
	// New proposer with next slot 1 and rnd (0,1) and Live Replacement
	// enabled
	proposer := NewMultiProposer(ppThreeNodesLr)

	// Simulate completed Phase 1
	proposer.phaseOneDone = true
	proposer.phaseOneVQCheck = true
	proposer.phaseOneCount = 2
	proposer.phaseOneRecvFrom = make(map[grp.ID]bool)
	proposer.phaseOneRecvFrom[r1id] = true
	proposer.phaseOneRecvFrom[r2id] = true
	proposer.phaseOnePromises = make([]*px.Promise, 0)
	proposer.phaseOnePromises = append(
		proposer.phaseOnePromises,
		&px.Promise{ID: r1id},
	)
	proposer.phaseOnePromises = append(
		proposer.phaseOnePromises,
		&px.Promise{ID: r2id},
	)

	// Reset Phase 1 data
	proposer.resetPhaseOneData()

	// Check that Phase 1 data is reset
	c.Assert(proposer.phaseOneDone, gc.Equals, false)
	c.Assert(proposer.phaseOneVQCheck, gc.Equals, false)
	c.Assert(proposer.phaseOneCount, gc.Equals, uint(0))
	c.Assert(proposer.phaseOnePromises, gc.HasLen, 0)
	c.Assert(proposer.phaseOneRecvFrom, gc.HasLen, 0)
}

func (*propSuite) TestStartPhaseOne(c *gc.C) {
	// New proposer with next slot 1 and rnd (0,1)
	proposer := NewMultiProposer(ppThreeNodesNonLr)

	// Simulate single client request in queue
	proposer.reqQueue.PushBack(valFoo)

	// Update state pre Phase 1 and reset client request queue
	proposer.updateStatePrePhaseOne(true)

	// Check that client request queue is reset
	c.Assert(proposer.reqQueue.Len(), gc.Equals, 0)

	// Check that state used to generate prepare is correct:
	// ID, current round and ADU (used as slot in prepare).
	c.Assert(proposer.id, gc.Equals, r0id)
	c.Assert(*proposer.crnd, gc.Equals, rnd02)
	c.Assert(proposer.adu, gc.Equals, sid0)

	// Simulate single client request in queue
	proposer.reqQueue.PushBack(valBar)

	// Update state pre Phase 1 and do not reset client request queue
	proposer.updateStatePrePhaseOne(false)

	// Check that client request queue is not reset
	c.Assert(proposer.reqQueue.Len(), gc.Equals, 1)

	// Check that state used to generate prepare is correct:
	// ID, current round and ADU (used as slot in prepare).
	c.Assert(proposer.id, gc.Equals, r0id)
	c.Assert(*proposer.crnd, gc.Equals, rnd03)
	c.Assert(proposer.adu, gc.Equals, sid0)
}

// -----------------------------------------------------------------------
// Tests: Handling promises

func (*propSuite) TestIgnorePromiseIfPhaseOneDOne(c *gc.C) {
	// New proposer with next slot 1 and rnd (0,1)
	proposer := NewMultiProposer(ppThreeNodesNonLr)
	proposer.resetPhaseOneData()

	// Simulate phase 1 done
	proposer.phaseOneDone = true

	// Should not be stored or report a quorum
	quorum := proposer.handlePromise(&px.Promise{
		ID:       r1id,
		Rnd:      rnd01,
		AccSlots: []px.AcceptorSlot{},
	})

	c.Assert(quorum, gc.Equals, false)
	c.Assert(proposer.phaseOnePromises[r1id.PxInt()], gc.IsNil)
}

func (*propSuite) TestIgnorePromiseIfDifferentRound(c *gc.C) {
	// New proposer with next slot 1 and rnd (0,1)
	proposer := NewMultiProposer(ppThreeNodesNonLr)
	proposer.resetPhaseOneData()

	// Should not be stored or report a quorum due to differing round
	quorum := proposer.handlePromise(&px.Promise{
		ID:       r1id,
		Rnd:      px.ProposerRound{ID: r0id, Rnd: 42},
		AccSlots: []px.AcceptorSlot{},
	})

	c.Assert(quorum, gc.Equals, false)
	c.Assert(proposer.phaseOnePromises[r1id.PxInt()], gc.IsNil)
}

func (*propSuite) TestAddPromiseIfValid(c *gc.C) {
	// New proposer with next slot 1 and rnd (0,1)
	proposer := NewMultiProposer(ppThreeNodesNonLr)
	proposer.resetPhaseOneData()

	// Should be stored but not report a quorum
	promise := &px.Promise{
		ID:       r1id,
		Rnd:      rnd01,
		AccSlots: []px.AcceptorSlot{},
	}
	quorum := proposer.handlePromise(promise)

	c.Assert(quorum, gc.Equals, false)
	c.Assert(proposer.phaseOnePromises[r1id.PxInt()], gc.NotNil)
	c.Assert(proposer.phaseOnePromises[r1id.PxInt()], gc.DeepEquals, promise)
}

func (*propSuite) TestIgnorePromiseIfCorrectRoundButDuplicate(c *gc.C) {
	// New proposer with next slot 1 and rnd (0,1)
	proposer := NewMultiProposer(ppThreeNodesNonLr)
	proposer.resetPhaseOneData()

	// Should be stored but not report a quorum
	promise := &px.Promise{
		ID:       r1id,
		Rnd:      rnd01,
		AccSlots: []px.AcceptorSlot{},
	}
	quorum := proposer.handlePromise(promise)

	c.Assert(quorum, gc.Equals, false)
	c.Assert(proposer.phaseOnePromises[r1id.PxInt()], gc.NotNil)
	c.Assert(proposer.phaseOnePromises[r1id.PxInt()], gc.DeepEquals, promise)

	// Should not be stored or report a quorum since it is duplicate
	// from the same replica
	quorum = proposer.handlePromise(promise)

	c.Assert(quorum, gc.Equals, false)
	c.Assert(proposer.phaseOnePromises[r1id.PxInt()], gc.NotNil)
	c.Assert(proposer.phaseOnePromises[r1id.PxInt()], gc.DeepEquals, promise)
}

func (*propSuite) TestAddQuorumOfPromises(c *gc.C) {
	// New proposer with next slot 1 and rnd (0,1)
	proposer := NewMultiProposer(ppThreeNodesNonLr)
	proposer.resetPhaseOneData()

	// Should be stored but not report a quorum
	promise1 := &px.Promise{
		ID:       r1id,
		Rnd:      rnd01,
		AccSlots: []px.AcceptorSlot{},
	}
	quorum := proposer.handlePromise(promise1)

	c.Assert(quorum, gc.Equals, false)
	c.Assert(proposer.phaseOnePromises[r1id.PxInt()], gc.NotNil)
	c.Assert(proposer.phaseOnePromises[r1id.PxInt()], gc.DeepEquals, promise1)

	// Should be stored and report a quorum
	promise2 := &px.Promise{
		ID:       r2id,
		Rnd:      rnd01,
		AccSlots: []px.AcceptorSlot{},
	}
	quorum = proposer.handlePromise(promise2)

	c.Assert(quorum, gc.Equals, true)
	c.Assert(proposer.phaseOnePromises[r2id.PxInt()], gc.NotNil)
	c.Assert(proposer.phaseOnePromises[r2id.PxInt()], gc.DeepEquals, promise2)
}

func (*propSuite) TestSetStateNoValuesReported(c *gc.C) {
	// New proposer with next slot 1, adu 0 and rnd (0,1)
	proposer := NewMultiProposer(ppThreeNodesNonLr)
	proposer.resetPhaseOneData()

	// Add a quorum of promises
	_ = proposer.handlePromise(&px.Promise{
		ID:       r1id,
		Rnd:      rnd01,
		AccSlots: []px.AcceptorSlot{},
	})
	_ = proposer.handlePromise(&px.Promise{
		ID:       r2id,
		Rnd:      rnd01,
		AccSlots: []px.AcceptorSlot{},
	})

	proposer.setStateFromPromises()

	// Slot 1 should not contain propsal
	s1 := proposer.slots.GetSlot(1)
	c.Assert(s1.Proposal, gc.IsNil)
}

func (*propSuite) TestSetStateSingleValueReported(c *gc.C) {
	// New proposer with next slot 1, adu 0 and rnd (0,1)
	proposer := NewMultiProposer(ppThreeNodesNonLr)
	proposer.resetPhaseOneData()

	// Increase crnd to (0,2)
	proposer.crnd.Next()

	// Add a quorum of promises, where r2 reports a value for slot 1
	_ = proposer.handlePromise(&px.Promise{
		ID:       r1id,
		Rnd:      rnd02,
		AccSlots: []px.AcceptorSlot{},
	})
	_ = proposer.handlePromise(&px.Promise{
		ID:  r2id,
		Rnd: rnd02,
		AccSlots: []px.AcceptorSlot{
			{
				ID:   1,
				VRnd: rnd11,
				VVal: valFoo,
			},
		},
	})

	proposer.setStateFromPromises()

	// Slot 1 should a contain propsal but not be decided
	s1 := proposer.slots.GetSlot(1)
	c.Assert(s1.Proposal, gc.NotNil)
	c.Assert(s1.Proposal.Rnd, gc.DeepEquals, rnd11)
	c.Assert(s1.Proposal.Val, gc.DeepEquals, valFoo)
}

func (*propSuite) TestSetStateIgnoreBelowAdu(c *gc.C) {
	// New proposer with next slot 1, adu 0 and rnd (0,1)
	proposer := NewMultiProposer(ppThreeNodesNonLr)
	proposer.resetPhaseOneData()

	// Set next slot to 2 and adu to 1
	proposer.nextSlot = sid2
	proposer.adu = sid1

	// Should be stored but not report a quorum.
	// AcceptorSlot is below adu and have no effect
	quorum := proposer.handlePromise(&px.Promise{
		ID:  r1id,
		Rnd: rnd01,
		AccSlots: []px.AcceptorSlot{
			{
				ID:   1,
				VRnd: rnd11,
				VVal: valFoo,
			},
		},
	})
	c.Assert(quorum, gc.Equals, false)
	c.Assert(proposer.phaseOnePromises[r1id.PxInt()], gc.NotNil)

	s1 := proposer.slots.GetSlot(1)
	c.Assert(s1.Proposal, gc.IsNil)
}

func (*propSuite) TestSetStateWithDifferingRoundsReportedForSameSlot(c *gc.C) {
	// New proposer with next slot 1, adu 0 and rnd (0,1)
	proposer := NewMultiProposer(ppThreeNodesNonLr)
	proposer.resetPhaseOneData()

	// Increase crnd to (0,3)
	proposer.crnd.Next()
	proposer.crnd.Next()

	// Add a quorum of promises.
	// Both r1 and r2 reports a value for slot 1.
	// R2 has the highest round and should be used.
	quorum := proposer.handlePromise(&px.Promise{
		ID:  r1id,
		Rnd: rnd03,
		AccSlots: []px.AcceptorSlot{
			{
				ID:   1,
				VRnd: rnd11,
				VVal: valFoo,
			},
		},
	})

	c.Assert(quorum, gc.Equals, false)

	quorum = proposer.handlePromise(&px.Promise{
		ID:  r2id,
		Rnd: rnd03,
		AccSlots: []px.AcceptorSlot{
			{
				ID:   1,
				VRnd: rnd12,
				VVal: valBar,
			},
		},
	})
	c.Assert(quorum, gc.Equals, true)

	proposer.setStateFromPromises()
	s1 := proposer.slots.GetSlot(1)
	c.Assert(s1.Proposal, gc.DeepEquals, &px.Accept{
		Rnd: rnd12,
		Val: valBar,
	})
}

// -----------------------------------------------------------------------
// Tests: Handling promises with Live Replacement enabled

func (*propSuite) TestDifferingEpochVectorShouldEnableVQCheck(c *gc.C) {
	// New proposer with next slot 1, adu 0 and rnd (0,1) and epoch vector
	// [0,0,0]
	proposer := NewMultiProposer(ppThreeNodesLr)
	proposer.resetPhaseOneData()

	// VQ-check for slot 1 should be false
	c.Assert(proposer.phaseOneVQCheck, gc.Equals, false)

	// Promise with differing epoch vector should enable phase 1 valid
	// quorum checking
	quorum := proposer.handlePromise(&px.Promise{
		ID:          r1id,
		Rnd:         rnd01,
		AccSlots:    []px.AcceptorSlot{},
		EpochVector: []grp.Epoch{0, 0, 1},
	})

	c.Assert(quorum, gc.Equals, false)
	c.Assert(proposer.phaseOneVQCheck, gc.Equals, true)
}

func (*propSuite) TestEqualEpochVectorShouldNotEnableVQCheck(c *gc.C) {
	// New proposer with next slot 1, adu 0 and rnd (0,1) and epoch vector
	// [0,0,0]
	proposer := NewMultiProposer(ppThreeNodesLr)
	proposer.resetPhaseOneData()

	// VQ-check for slot 1 should be false
	c.Assert(proposer.phaseOneVQCheck, gc.Equals, false)

	// Promise with equal epoch vector should not enable phase 1 valid
	// quorum checking
	quorum := proposer.handlePromise(&px.Promise{
		ID:          r1id,
		Rnd:         rnd01,
		AccSlots:    []px.AcceptorSlot{},
		EpochVector: []grp.Epoch{0, 0, 0},
	})

	c.Assert(quorum, gc.Equals, false)
	c.Assert(proposer.phaseOneVQCheck, gc.Equals, false)
}

func (*propSuite) TestTwoPromisesWithConflictingEpVecShouldNotGiveQuorum(c *gc.C) {
	// New proposer with next slot 1, adu 0 and rnd (0,1) and epoch vector
	// [0,0,0]
	proposer := NewMultiProposer(ppThreeNodesLr)
	proposer.resetPhaseOneData()

	// Promise from r1 with [0,0,0]
	quorum := proposer.handlePromise(&px.Promise{
		ID:          r1id,
		Rnd:         rnd01,
		AccSlots:    []px.AcceptorSlot{},
		EpochVector: []grp.Epoch{0, 0, 0},
	})

	c.Assert(quorum, gc.Equals, false)

	// Promise from r1 with [0,1,0]
	// Not a valid quorum
	quorum = proposer.handlePromise(&px.Promise{
		ID:          r2id,
		Rnd:         rnd01,
		AccSlots:    []px.AcceptorSlot{},
		EpochVector: []grp.Epoch{0, 1, 0},
	})

	c.Assert(quorum, gc.Equals, false)
}

func (*propSuite) TestPromisesWithNonConflictingEpVecShouldGiveQuorumVariationOne(c *gc.C) {
	// New proposer with next slot 1, adu 0 and rnd (0,1) and epoch vector
	// [0,0,0]
	proposer := NewMultiProposer(ppThreeNodesLr)
	proposer.resetPhaseOneData()

	// Promise from r0 with [0,0,0]
	quorum := proposer.handlePromise(&px.Promise{
		ID:          r0id,
		Rnd:         rnd01,
		AccSlots:    []px.AcceptorSlot{},
		EpochVector: []grp.Epoch{0, 0, 0},
	})
	c.Assert(quorum, gc.Equals, false)

	// Promise from r2 with [0,1,0]
	// A valid quorum frin r0 and r2
	quorum = proposer.handlePromise(&px.Promise{
		ID:          r2id,
		Rnd:         rnd01,
		AccSlots:    []px.AcceptorSlot{},
		EpochVector: []grp.Epoch{0, 1, 0},
	})
	c.Assert(quorum, gc.Equals, true)
}

func (*propSuite) TestPromisesWithNonConflictingEpVecShouldGiveQuorumVariationTwo(c *gc.C) {
	// New proposer with next slot 1, adu 0 and rnd (0,1) and epoch vector
	// [0,0,0]
	proposer := NewMultiProposer(ppThreeNodesLr)
	proposer.resetPhaseOneData()

	// Promise from r1 with [0,0,1]
	quorum := proposer.handlePromise(&px.Promise{
		ID:          r1id,
		Rnd:         rnd01,
		AccSlots:    []px.AcceptorSlot{},
		EpochVector: []grp.Epoch{0, 0, 1},
	})
	c.Assert(quorum, gc.Equals, false)

	// Promise from r2 with [0,1,0]
	// Not a valid quorum
	quorum = proposer.handlePromise(&px.Promise{
		ID:          r2id,
		Rnd:         rnd01,
		AccSlots:    []px.AcceptorSlot{},
		EpochVector: []grp.Epoch{0, 1, 0},
	})
	c.Assert(quorum, gc.Equals, false)

	// Promise from r0 with [0,0,1]
	// A valid quorum from r0 and r1
	quorum = proposer.handlePromise(&px.Promise{
		ID:          r0id,
		Rnd:         rnd01,
		AccSlots:    []px.AcceptorSlot{},
		EpochVector: []grp.Epoch{0, 0, 1},
	})
	c.Assert(quorum, gc.Equals, true)
}

func (*propSuite) TestPromisesWithNonConflictingEpVecShouldGiveQuorumVariationThree(c *gc.C) {
	// New proposer with next slot 1, adu 0 and rnd (0,1) and epoch vector
	// [0,0,0]
	proposer := NewMultiProposer(ppThreeNodesLr)
	proposer.resetPhaseOneData()

	// Promise from r1 with [0,0,1]
	quorum := proposer.handlePromise(&px.Promise{
		ID:          r1id,
		Rnd:         rnd01,
		AccSlots:    []px.AcceptorSlot{},
		EpochVector: []grp.Epoch{0, 0, 1},
	})
	c.Assert(quorum, gc.Equals, false)

	// Promise from r2 with [0,1,0]
	// Not a valid quorum
	quorum = proposer.handlePromise(&px.Promise{
		ID:          r2id,
		Rnd:         rnd01,
		AccSlots:    []px.AcceptorSlot{},
		EpochVector: []grp.Epoch{0, 1, 0},
	})
	c.Assert(quorum, gc.Equals, false)

	// Promise from r0 with [0,1,0]
	// A valid quorum from r0 and r2
	quorum = proposer.handlePromise(&px.Promise{
		ID:          r0id,
		Rnd:         rnd01,
		AccSlots:    []px.AcceptorSlot{},
		EpochVector: []grp.Epoch{0, 1, 0},
	})
	c.Assert(quorum, gc.Equals, true)
}

func (*propSuite) TestSetStateSingleValueReportedLiveReplacement(c *gc.C) {
	// New proposer with next slot 1, adu 0 and rnd (0,1) and epoch vector
	// [0,0,0]
	proposer := NewMultiProposer(ppThreeNodesLr)
	proposer.resetPhaseOneData()

	// Increase crnd to (0,2)
	proposer.crnd.Next()

	// Add a quorum of promises, where r2 reports a value for slot 1
	_ = proposer.handlePromise(&px.Promise{
		ID:          r1id,
		Rnd:         rnd02,
		EpochVector: []grp.Epoch{0, 0, 0},
		AccSlots:    []px.AcceptorSlot{},
	})
	_ = proposer.handlePromise(&px.Promise{
		ID:          r2id,
		Rnd:         rnd02,
		EpochVector: []grp.Epoch{0, 0, 0},
		AccSlots: []px.AcceptorSlot{
			{
				ID:   1,
				VRnd: rnd11,
				VVal: valFoo,
			},
		},
	})

	proposer.setStateFromPromises()

	// Slot 1 should a contain propsal but not be decided
	s1 := proposer.slots.GetSlot(1)
	c.Assert(s1.Proposal, gc.NotNil)
	c.Assert(s1.Proposal.Rnd, gc.DeepEquals, rnd11)
	c.Assert(s1.Proposal.Val, gc.DeepEquals, valFoo)
}

// -----------------------------------------------------------------------
// Tests: Generating accepts

func (*propSuite) TestGenNoAcceptWithEmptyReqQueue(c *gc.C) {
	// New proposer with next slot 1 and rnd (0,1)
	proposer := NewMultiProposer(ppThreeNodesNonLr)

	// Increment crnd to (0,1) to simulate phase 1 done
	proposer.crnd.Next()

	// No history and no requests in queue
	nextSlot := proposer.slots.GetSlot(proposer.nextSlot)
	accept := proposer.genAccept(nextSlot)
	c.Assert(accept, gc.IsNil)
}

func (*propSuite) TestGenAcceptWithNOnemptyReqQueue(c *gc.C) {
	// New proposer with next slot 1 and rnd (0,1)
	proposer := NewMultiProposer(ppThreeNodesNonLr)

	// Increment crnd to (0,2) to simulate phase 1 done
	proposer.crnd.Next()

	// Add One request to the queue
	proposer.reqQueue.PushFront(&valFoo)

	// No history but One request in queue
	nextSlot := proposer.slots.GetSlot(proposer.nextSlot)
	accept := proposer.genAccept(nextSlot)
	c.Assert(accept, gc.NotNil)
	c.Assert(accept.ID, gc.Equals, r0id)
	c.Assert(accept.Rnd, gc.Equals, rnd02)
	c.Assert(accept.Slot, gc.Equals, sid1)
	c.Assert(accept.Val, gc.DeepEquals, valFoo)
}

func (*propSuite) TestGenAcceptWithBoundedSlot(c *gc.C) {
	// New proposer with next slot 1 and rnd (0,1)
	proposer := NewMultiProposer(ppThreeNodesNonLr)

	// Increment crnd to (0,2) to simulate phase 1 done
	proposer.crnd.Next()

	// Simulate a recevied proposal from a phase 1 promise for slot 1
	nextSlot := proposer.slots.GetSlot(proposer.nextSlot)
	nextSlot.Proposal = &px.Accept{
		ID:   r1id,
		Slot: sid1,
		Rnd:  rnd21,
		Val:  valBar,
	}

	// Bound history should produce accept
	accept := proposer.genAccept(nextSlot)
	c.Assert(accept, gc.NotNil)
	c.Assert(accept.ID, gc.Equals, r0id)
	c.Assert(accept.Slot, gc.Equals, sid1)
	c.Assert(accept.Rnd, gc.Equals, rnd02)
	c.Assert(accept.Val, gc.DeepEquals, valBar)
}

func (*propSuite) TestGenAcceptWithBoundedSlotAndNOnemptyQueue(c *gc.C) {
	// New proposer with next slot 1 and rnd (0,1)
	proposer := NewMultiProposer(ppThreeNodesNonLr)

	// Increment crnd to (0,2) to simulate phase 1 done
	proposer.crnd.Next()

	// Simulate a recevied proposal from a phase 1 promise for slot 1
	nextSlot := proposer.slots.GetSlot(proposer.nextSlot)
	nextSlot.Proposal = &px.Accept{
		ID:   r1id,
		Slot: sid1,
		Rnd:  rnd21,
		Val:  valBar,
	}

	// Add One request to the queue
	proposer.reqQueue.PushFront(&valFoo)

	// Bound history should produce accept, no the queued client request
	accept := proposer.genAccept(nextSlot)
	c.Assert(accept, gc.NotNil)
	c.Assert(accept.ID, gc.Equals, r0id)
	c.Assert(accept.Slot, gc.Equals, sid1)
	c.Assert(accept.Rnd, gc.Equals, rnd02)
	c.Assert(accept.Val, gc.DeepEquals, valBar)
}

// -----------------------------------------------------------------------
// Tests: Utility methods

var alphatests = []struct {
	nextSlot, adu px.SlotID
	alpha         uint
	inRange       bool
}{
	{0, 0, 3, true},
	{1, 0, 3, true},
	{2, 0, 3, true},
	{3, 0, 3, true},
	{4, 0, 3, false},
	{3, 1, 3, true},
	{3, 2, 3, true},
	{3, 3, 3, true},
	{3, 4, 3, true},
	{3, 5, 3, true},
	{3, 6, 3, true},
	{0, 0, 1, true},
	{1, 0, 1, true},
	{1, 1, 1, true},
}

func (*propSuite) TestInAlphaRange(c *gc.C) {
	for i, tt := range alphatests {
		inRange := inAlphaRange(tt.nextSlot, tt.adu, tt.alpha)
		c.Assert(inRange, gc.Equals, tt.inRange,
			gc.Commentf("%d: ns: %d, adu: %d, alpha: %d",
				i, tt.nextSlot, tt.adu, tt.alpha))
	}
}

// -----------------------------------------------------------------------
// Tests: Resending accepts

func (*propSuite) TestResetSentCountersInAlphaWindow(c *gc.C) {
	// New proposer with next slot 1 and rnd (0,1)
	proposer := NewMultiProposer(ppThreeNodesNonLr)

	// Simulate state:
	// 	- ADU: 1
	//	- nextSlot: 4
	//	- Slot 2 and 3: >0 values for SentCount
	proposer.adu = 1
	proposer.nextSlot = 4
	proposer.slots = px.NewProposerSlotMap()
	s2 := proposer.slots.GetSlot(sid2)
	s2.SentCount = 8
	s3 := proposer.slots.GetSlot(sid3)
	s3.SentCount = 9

	// Set ADU and next slot to != 0 values to check that they are not
	// touched by reset call below.
	s1 := proposer.slots.GetSlot(sid1) // ADU
	s1.SentCount = 42
	s4 := proposer.slots.GetSlot(sid4) // Next slot
	s4.SentCount = 42

	// Reset
	proposer.resetSentCountersInAlphaWindow()

	// Counters in window should be reset
	c.Assert(s2.SentCount, gc.Equals, uint8(0))
	c.Assert(s3.SentCount, gc.Equals, uint8(0))

	// Counters outside window should not be reset
	c.Assert(s1.SentCount, gc.Equals, uint8(42))
	c.Assert(s4.SentCount, gc.Equals, uint8(42))
}
