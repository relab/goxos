package multipaxos

import (
	"github.com/relab/goxos/grp"
	px "github.com/relab/goxos/paxos"

	gc "github.com/relab/goxos/Godeps/_workspace/src/gopkg.in/check.v1"
)

// -----------------------------------------------------------------------
// Initialization

type lrnSuite struct {
	lsm   *px.LearnerSlotMap
	lsmLr *px.LearnerLrSlotMap
}

var _ = gc.Suite(&lrnSuite{})

func (ls *lrnSuite) SetUpSuite(c *gc.C) {
	// Populate a learner slot map with history, slot 1-3
	ls.lsm = px.NewLearnerSlotMap()
	s1 := ls.lsm.GetSlot(1)
	s1.Decided = true
	s1.LearnedVal = valFoo
	s2 := ls.lsm.GetSlot(2)
	s2.Decided = true
	s2.LearnedVal = valBar
	s3 := ls.lsm.GetSlot(3)
	s3.Decided = true
	s3.LearnedVal = valFoo

	// Populate a learner LR slot map with history, slot 1-3
	ls.lsmLr = px.NewLearnerLrSlotMap()
	s1lr := ls.lsmLr.GetSlot(1)
	s1lr.Decided = true
	s1lr.LearnedVal = valFoo
	s2lr := ls.lsmLr.GetSlot(2)
	s2lr.Decided = true
	s2lr.LearnedVal = valBar
	s3lr := ls.lsmLr.GetSlot(3)
	s3lr.Decided = true
	s3lr.LearnedVal = valFoo
}

// -----------------------------------------------------------------------
// Tests: Handling learn messages

func (*lrnSuite) TestLearnWithQuorumOfTwo(c *gc.C) {
	learner := NewMultiLearner(ppThreeNodesNonLr)

	// Round 1 from r0 is not a quorum
	val, sid := learner.handleLearn(&px.Learn{
		ID:   r0id,
		Slot: 1,
		Rnd:  rnd01,
		Val:  valFoo,
	})
	c.Assert(val, gc.IsNil)
	c.Assert(sid, gc.Equals, sid0)

	// Round 1 from r1 is a quorum
	val, sid = learner.handleLearn(&px.Learn{
		ID:   r1id,
		Slot: 1,
		Rnd:  rnd01,
		Val:  valFoo,
	})
	c.Assert(val, gc.DeepEquals, &valFoo)
	c.Assert(sid, gc.Equals, sid1)

	// We should advance and no catch-up should trigger
	advance, startcu, cuslot := learner.learnValue(val, sid)
	c.Assert(advance, gc.Equals, true)
	c.Assert(startcu, gc.Equals, false)
	c.Assert(cuslot, gc.Equals, sid0)

	// First advance should return decided value and correct slot id
	dcdVal, slotID := learner.advance()
	c.Assert(dcdVal, gc.DeepEquals, &valFoo)
	c.Assert(slotID, gc.Equals, sid1)

	// Next advance should return no decided value
	dcdVal, slotID = learner.advance()
	c.Assert(dcdVal, gc.IsNil)
	c.Assert(slotID, gc.Equals, sid0)

	// Last learn should be ignored
	val, sid = learner.handleLearn(&px.Learn{
		ID:   r2id,
		Slot: 1,
		Rnd:  rnd01,
		Val:  valFoo,
	})
	c.Assert(val, gc.IsNil)
	c.Assert(sid, gc.Equals, sid0)
}

func (*lrnSuite) TestIgnoreOldRound(c *gc.C) {
	learner := NewMultiLearner(ppThreeNodesNonLr)

	// Round 2 from r0 is not a quorum
	val, sid := learner.handleLearn(&px.Learn{
		ID:   r0id,
		Slot: 1,
		Rnd:  rnd02,
		Val:  valFoo,
	})
	c.Assert(val, gc.IsNil)
	c.Assert(sid, gc.Equals, sid0)

	// Round 1 from r1 is an old round and should be ignored
	val, sid = learner.handleLearn(&px.Learn{
		ID:   r1id,
		Slot: 1,
		Rnd:  rnd01,
		Val:  valFoo,
	})
	c.Assert(val, gc.IsNil)
	c.Assert(sid, gc.Equals, sid0)

	// Round 2 from r2 is a quorum
	val, sid = learner.handleLearn(&px.Learn{
		ID:   r2id,
		Slot: 1,
		Rnd:  rnd02,
		Val:  valFoo,
	})
	c.Assert(val, gc.DeepEquals, &valFoo)
	c.Assert(sid, gc.Equals, sid1)

	// We should advance and no catch-up should trigger
	advance, startcu, cuslot := learner.learnValue(val, sid)
	c.Assert(advance, gc.Equals, true)
	c.Assert(startcu, gc.Equals, false)
	c.Assert(cuslot, gc.Equals, sid0)

	// First advance should return decided value and correct slot id
	dcdVal, slotID := learner.advance()
	c.Assert(dcdVal, gc.DeepEquals, &valFoo)
	c.Assert(slotID, gc.Equals, sid1)

	// Next advance should return no decided value
	dcdVal, slotID = learner.advance()
	c.Assert(dcdVal, gc.IsNil)
	c.Assert(slotID, gc.Equals, sid0)
}

func (*lrnSuite) TestIgnoreTwoEqualLearnsFromSameReplica(c *gc.C) {
	learner := NewMultiLearner(ppThreeNodesNonLr)

	// Not a quorum
	val, sid := learner.handleLearn(&px.Learn{
		ID:   r1id,
		Slot: 1,
		Rnd:  rnd01,
		Val:  valFoo,
	})
	c.Assert(val, gc.IsNil)
	c.Assert(sid, gc.Equals, sid0)

	// Not a quorum, already have a message from r1
	val, sid = learner.handleLearn(&px.Learn{
		ID:   r1id,
		Slot: 1,
		Rnd:  rnd01,
		Val:  valFoo,
	})
	c.Assert(val, gc.IsNil)
	c.Assert(sid, gc.Equals, sid0)
}

func (*lrnSuite) TestResetDueToHigherRound(c *gc.C) {
	learner := NewMultiLearner(ppThreeNodesNonLr)

	// Round 1 from r1 is not a quorum
	val, sid := learner.handleLearn(&px.Learn{
		ID:   r1id,
		Slot: 1,
		Rnd:  rnd01,
		Val:  valFoo,
	})
	c.Assert(val, gc.IsNil)
	c.Assert(sid, gc.Equals, sid0)

	// Round 2 from r2 should reset, still no quorum
	val, sid = learner.handleLearn(&px.Learn{
		ID:   r2id,
		Slot: 1,
		Rnd:  rnd02,
		Val:  valFoo,
	})
	c.Assert(val, gc.IsNil)
	c.Assert(sid, gc.Equals, sid0)

	// Round 2 from r1 is a quorum
	val, sid = learner.handleLearn(&px.Learn{
		ID:   r1id,
		Slot: 1,
		Rnd:  rnd02,
		Val:  valFoo,
	})
	c.Assert(val, gc.DeepEquals, &valFoo)
	c.Assert(sid, gc.Equals, sid1)

	// We should advance and no catch-up should trigger
	advance, startcu, cuslot := learner.learnValue(val, sid)
	c.Assert(advance, gc.Equals, true)
	c.Assert(startcu, gc.Equals, false)
	c.Assert(cuslot, gc.Equals, sid0)

	// First advance should return decided value and correct slot id
	dcdVal, slotID := learner.advance()
	c.Assert(dcdVal, gc.DeepEquals, &valFoo)
	c.Assert(slotID, gc.Equals, sid1)

	// Next advance should return no decided value
	dcdVal, slotID = learner.advance()
	c.Assert(dcdVal, gc.IsNil)
	c.Assert(slotID, gc.Equals, sid0)
}

func (*lrnSuite) TestTwoLearnsWithEqualSenderShouldNotGiveQuorum(c *gc.C) {
	// Learner with no history, next expected is slot 1
	learner := NewMultiLearner(ppThreeNodesNonLr)

	// Learn for slot 1 from r1
	val, sid := learner.handleLearn(&px.Learn{
		ID:   r1id,
		Slot: 1,
		Rnd:  rnd01,
		Val:  valFoo,
	})
	c.Assert(val, gc.IsNil)
	c.Assert(sid, gc.Equals, sid0)

	// Learn for slot 1 with [0,0,0] from r1
	// Not a valid quorum
	val, sid = learner.handleLearn(&px.Learn{
		ID:   r1id,
		Slot: 1,
		Rnd:  rnd01,
		Val:  valFoo,
	})
	c.Assert(val, gc.IsNil)
	c.Assert(sid, gc.Equals, sid0)
}

// -----------------------------------------------------------------------
// Tests: Handling learn messages with Live Replacement enabled

func (*lrnSuite) TestLearnWithQuorumOfTwoLr(c *gc.C) {
	learner := NewMultiLearner(ppThreeNodesLr)

	// Round 1 from r0 is not a quorum
	val, sid := learner.handleLearn(&px.Learn{
		ID:          r0id,
		Slot:        1,
		Rnd:         rnd01,
		Val:         valFoo,
		EpochVector: []grp.Epoch{0, 0, 0},
	})
	c.Assert(val, gc.IsNil)
	c.Assert(sid, gc.Equals, sid0)

	// Round 1 from r1 is a quorum
	val, sid = learner.handleLearn(&px.Learn{
		ID:          r1id,
		Slot:        1,
		Rnd:         rnd01,
		Val:         valFoo,
		EpochVector: []grp.Epoch{0, 0, 0},
	})
	c.Assert(val, gc.DeepEquals, &valFoo)
	c.Assert(sid, gc.Equals, sid1)

	// We should advance and no catch-up should trigger
	advance, startcu, cuslot := learner.learnValue(val, sid)
	c.Assert(advance, gc.Equals, true)
	c.Assert(startcu, gc.Equals, false)
	c.Assert(cuslot, gc.Equals, sid0)

	// First advance should return decided value and correct slot id
	dcdVal, slotID := learner.advance()
	c.Assert(dcdVal, gc.DeepEquals, &valFoo)
	c.Assert(slotID, gc.Equals, sid1)

	// Next advance should return no decided value
	dcdVal, slotID = learner.advance()
	c.Assert(dcdVal, gc.IsNil)
	c.Assert(slotID, gc.Equals, sid0)

	// Last learn should be ignored
	val, sid = learner.handleLearn(&px.Learn{
		ID:   r2id,
		Slot: 1,
		Rnd:  rnd01,
		Val:  valFoo,
	})
	c.Assert(val, gc.IsNil)
	c.Assert(sid, gc.Equals, sid0)
}

func (*lrnSuite) TestIgnoreOldRoundLr(c *gc.C) {
	learner := NewMultiLearner(ppThreeNodesLr)

	// Round 2 from r0 is not a quorum
	val, sid := learner.handleLearn(&px.Learn{
		ID:          r0id,
		Slot:        1,
		Rnd:         rnd02,
		Val:         valFoo,
		EpochVector: []grp.Epoch{0, 0, 0},
	})
	c.Assert(val, gc.IsNil)
	c.Assert(sid, gc.Equals, sid0)

	// Round 1 from r1 is an old round and should be ignored
	val, sid = learner.handleLearn(&px.Learn{
		ID:          r1id,
		Slot:        1,
		Rnd:         rnd01,
		Val:         valFoo,
		EpochVector: []grp.Epoch{0, 0, 0},
	})
	c.Assert(val, gc.IsNil)
	c.Assert(sid, gc.Equals, sid0)

	// Round 2 from r2 is a quorum
	val, sid = learner.handleLearn(&px.Learn{
		ID:          r2id,
		Slot:        1,
		Rnd:         rnd02,
		Val:         valFoo,
		EpochVector: []grp.Epoch{0, 0, 0},
	})
	c.Assert(val, gc.DeepEquals, &valFoo)
	c.Assert(sid, gc.Equals, sid1)

	// We should advance and no catch-up should trigger
	advance, startcu, cuslot := learner.learnValue(val, sid)
	c.Assert(advance, gc.Equals, true)
	c.Assert(startcu, gc.Equals, false)
	c.Assert(cuslot, gc.Equals, sid0)

	// First advance should return decided value and correct slot id
	dcdVal, slotID := learner.advance()
	c.Assert(dcdVal, gc.DeepEquals, &valFoo)
	c.Assert(slotID, gc.Equals, sid1)

	// Next advance should return no decided value
	dcdVal, slotID = learner.advance()
	c.Assert(dcdVal, gc.IsNil)
	c.Assert(slotID, gc.Equals, sid0)
}

func (*lrnSuite) TestIgnoreTwoEqualLearnsFromSameReplicaLr(c *gc.C) {
	learner := NewMultiLearner(ppThreeNodesLr)

	// Not a quorum
	val, sid := learner.handleLearn(&px.Learn{
		ID:          r1id,
		Slot:        1,
		Rnd:         rnd01,
		Val:         valFoo,
		EpochVector: []grp.Epoch{0, 0, 0},
	})
	c.Assert(val, gc.IsNil)
	c.Assert(sid, gc.Equals, sid0)

	// Not a quorum, already have a message from r1
	val, sid = learner.handleLearn(&px.Learn{
		ID:          r1id,
		Slot:        1,
		Rnd:         rnd01,
		Val:         valFoo,
		EpochVector: []grp.Epoch{0, 0, 0},
	})
	c.Assert(val, gc.IsNil)
	c.Assert(sid, gc.Equals, sid0)
}

func (*lrnSuite) TestResetDueToHigherRoundLr(c *gc.C) {
	learner := NewMultiLearner(ppThreeNodesLr)

	// Round 1 from r1 is not a quorum
	val, sid := learner.handleLearn(&px.Learn{
		ID:          r1id,
		Slot:        1,
		Rnd:         rnd01,
		Val:         valFoo,
		EpochVector: []grp.Epoch{0, 0, 0},
	})
	c.Assert(val, gc.IsNil)
	c.Assert(sid, gc.Equals, sid0)

	// Round 2 from r2 should reset, still no quorum
	val, sid = learner.handleLearn(&px.Learn{
		ID:          r2id,
		Slot:        1,
		Rnd:         rnd02,
		Val:         valFoo,
		EpochVector: []grp.Epoch{0, 0, 0},
	})
	c.Assert(val, gc.IsNil)
	c.Assert(sid, gc.Equals, sid0)

	// Round 2 from r1 is a quorum
	val, sid = learner.handleLearn(&px.Learn{
		ID:          r1id,
		Slot:        1,
		Rnd:         rnd02,
		Val:         valFoo,
		EpochVector: []grp.Epoch{0, 0, 0},
	})
	c.Assert(val, gc.DeepEquals, &valFoo)
	c.Assert(sid, gc.Equals, sid1)

	// We should advance and no catch-up should trigger
	advance, startcu, cuslot := learner.learnValue(val, sid)
	c.Assert(advance, gc.Equals, true)
	c.Assert(startcu, gc.Equals, false)
	c.Assert(cuslot, gc.Equals, sid0)

	// First advance should return decided value and correct slot id
	dcdVal, slotID := learner.advance()
	c.Assert(dcdVal, gc.DeepEquals, &valFoo)
	c.Assert(slotID, gc.Equals, sid1)

	// Next advance should return no decided value
	dcdVal, slotID = learner.advance()
	c.Assert(dcdVal, gc.IsNil)
	c.Assert(slotID, gc.Equals, sid0)
}

func (*lrnSuite) TestTwoLearnsWithEqualSenderButDifferentRoundsShouldNotGiveQuorumLr(c *gc.C) {
	// Learner with no history, next expected is slot 1
	// LR enabled and epoch vector [0,0,0]
	learner := NewMultiLearner(ppThreeNodesLr)

	// Learn for slot 1, round (1,1) with [0,0,0] from r1
	val, sid := learner.handleLearn(&px.Learn{
		ID:          r1id,
		Slot:        1,
		Rnd:         rnd11,
		Val:         valFoo,
		EpochVector: []grp.Epoch{0, 0, 0},
	})
	c.Assert(val, gc.IsNil)
	c.Assert(sid, gc.Equals, sid0)

	// Learn for slot 1, round (1,2) with [0,0,0] from r1
	// Not a valid quorum
	val, sid = learner.handleLearn(&px.Learn{
		ID:          r1id,
		Slot:        1,
		Rnd:         rnd12,
		Val:         valFoo,
		EpochVector: []grp.Epoch{0, 0, 0},
	})
	c.Assert(val, gc.IsNil)
	c.Assert(sid, gc.Equals, sid0)
}

// -----------------------------------------------------------------------
// Tests: Handling learn messages with Live Replacement enabled

func (*lrnSuite) TestDifferingEpochVectorShouldEnableVQCheck(c *gc.C) {
	// Learner with no history, next expected is slot 1
	// LR enabled and epoch vector [0,0,0]
	learner := NewMultiLearner(ppThreeNodesLr)

	// VQ-check for slot 1 should be false
	vqc := learner.slotsLr.GetSlot(1).CheckVQ
	c.Assert(vqc, gc.Equals, false)

	// Learn for slot 1 with differing epoch vector should enable VQ-check
	_, _ = learner.handleLearn(&px.Learn{
		ID:          r1id,
		Slot:        1,
		Rnd:         rnd01,
		Val:         valFoo,
		EpochVector: []grp.Epoch{0, 0, 1},
	})
	vqc = learner.slotsLr.GetSlot(1).CheckVQ
	c.Assert(vqc, gc.Equals, true)
}

func (*lrnSuite) TestEqualEpochVectorShouldNotEnableVQCheck(c *gc.C) {
	// Learner with no history, next expected is slot 1
	// LR enabled and epoch vector [0,0,0]
	learner := NewMultiLearner(ppThreeNodesLr)

	// VQ-check for slot 1 should be false
	vqc := learner.slotsLr.GetSlot(1).CheckVQ
	c.Assert(vqc, gc.Equals, false)

	// Learn for slot 1 with equal epoch vector should not enable VQ-check
	_, _ = learner.handleLearn(&px.Learn{
		ID:          r1id,
		Slot:        1,
		Rnd:         rnd01,
		Val:         valFoo,
		EpochVector: []grp.Epoch{0, 0, 0},
	})
	vqc = learner.slotsLr.GetSlot(1).CheckVQ
	c.Assert(vqc, gc.Equals, false)
}

func (*lrnSuite) TestLearnsWithConflictingEpVecShouldNotGiveQuorum(c *gc.C) {
	// Learner with no history, next expected is slot 1
	// LR enabled and epoch vector [0,0,0]
	learner := NewMultiLearner(ppThreeNodesLr)

	// Learn for slot 1 with [0,0,0]
	val, sid := learner.handleLearn(&px.Learn{
		ID:          r1id,
		Slot:        1,
		Rnd:         rnd01,
		Val:         valFoo,
		EpochVector: []grp.Epoch{0, 0, 0},
	})
	c.Assert(val, gc.IsNil)
	c.Assert(sid, gc.Equals, sid0)

	// Learn for slot 1 with [0,0,1]
	// Not a valid quorum
	val, sid = learner.handleLearn(&px.Learn{
		ID:          r2id,
		Slot:        1,
		Rnd:         rnd01,
		Val:         valFoo,
		EpochVector: []grp.Epoch{0, 1, 0},
	})
	c.Assert(val, gc.IsNil)
	c.Assert(sid, gc.Equals, sid0)
}

func (*lrnSuite) TestLearnsWithNonConflictingEpVecShouldGiveQuorumVariationOne(c *gc.C) {
	// Learner with no history, next expected is slot 1
	// LR enabled and epoch vector [0,0,0]
	learner := NewMultiLearner(ppThreeNodesLr)

	// Learn for slot 1 from r0 with [0,0,0]
	val, sid := learner.handleLearn(&px.Learn{
		ID:          r0id,
		Slot:        1,
		Rnd:         rnd01,
		Val:         valFoo,
		EpochVector: []grp.Epoch{0, 0, 0},
	})
	c.Assert(val, gc.IsNil)
	c.Assert(sid, gc.Equals, sid0)

	// Learn for slot 1 from r2 with [0,0,1]
	// A valid quorum from r0 and r2
	val, sid = learner.handleLearn(&px.Learn{
		ID:          r2id,
		Slot:        1,
		Rnd:         rnd01,
		Val:         valFoo,
		EpochVector: []grp.Epoch{0, 1, 0},
	})
	c.Assert(val, gc.NotNil)
	c.Assert(val, gc.DeepEquals, &valFoo)
	c.Assert(sid, gc.Equals, sid1)
}

func (*lrnSuite) TestLearnsWithNonConflictingEpVecShouldGiveQuorumVariationTwo(c *gc.C) {
	// Learner with no history, next expected is slot 1
	// LR enabled and epoch vector [0,0,0]
	learner := NewMultiLearner(ppThreeNodesLr)

	// Learn for slot 1 from r1 with [0,0,1]
	val, sid := learner.handleLearn(&px.Learn{
		ID:          r1id,
		Slot:        1,
		Rnd:         rnd01,
		Val:         valFoo,
		EpochVector: []grp.Epoch{0, 0, 1},
	})
	c.Assert(val, gc.IsNil)
	c.Assert(sid, gc.Equals, sid0)

	// Learn for slot 1 from r2 with [0,0,1]
	// Not a valid quorum
	val, sid = learner.handleLearn(&px.Learn{
		ID:          r2id,
		Slot:        1,
		Rnd:         rnd01,
		Val:         valFoo,
		EpochVector: []grp.Epoch{0, 1, 0},
	})
	c.Assert(val, gc.IsNil)
	c.Assert(sid, gc.Equals, sid0)

	// Learn for slot 1 from r0 with [0,0,1]
	// A valid quorum from r0 and r1
	val, sid = learner.handleLearn(&px.Learn{
		ID:          r0id,
		Slot:        1,
		Rnd:         rnd01,
		Val:         valFoo,
		EpochVector: []grp.Epoch{0, 0, 1},
	})
	c.Assert(val, gc.NotNil)
	c.Assert(val, gc.DeepEquals, &valFoo)
	c.Assert(sid, gc.Equals, sid1)
}

func (*lrnSuite) TestLearnsWithNonConflictingEpVecShouldGiveQuorumVariationThree(c *gc.C) {
	// Learner with no history, next expected is slot 1
	// LR enabled and epoch vector [0,0,0]
	learner := NewMultiLearner(ppThreeNodesLr)

	// Learn for slot 1 from r1 with [0,0,1]
	val, sid := learner.handleLearn(&px.Learn{
		ID:          r1id,
		Slot:        1,
		Rnd:         rnd01,
		Val:         valFoo,
		EpochVector: []grp.Epoch{0, 0, 1},
	})
	c.Assert(val, gc.IsNil)
	c.Assert(sid, gc.Equals, sid0)

	// Learn for slot 1 from r2 with [0,1,0]
	// Not a valid quorum
	val, sid = learner.handleLearn(&px.Learn{
		ID:          r2id,
		Slot:        1,
		Rnd:         rnd01,
		Val:         valFoo,
		EpochVector: []grp.Epoch{0, 1, 0},
	})
	c.Assert(val, gc.IsNil)
	c.Assert(sid, gc.Equals, sid0)

	// Learn for slot 1 from r0 with [0,1,0]
	// A valid quorum from r0 and r2
	val, sid = learner.handleLearn(&px.Learn{
		ID:          r0id,
		Slot:        1,
		Rnd:         rnd01,
		Val:         valFoo,
		EpochVector: []grp.Epoch{0, 1, 0},
	})
	c.Assert(val, gc.NotNil)
	c.Assert(val, gc.DeepEquals, &valFoo)
	c.Assert(sid, gc.Equals, sid1)
}

// -----------------------------------------------------------------------
// Tests: Catch-up

func (*lrnSuite) TestTiggerCatchUp(c *gc.C) {
	// We expect next decided to be slot 1
	learner := NewMultiLearner(ppThreeNodesNonLr)

	// Slot 4, round 1 from r1 is not a quorum
	val, sid := learner.handleLearn(&px.Learn{
		ID:   r1id,
		Slot: 4,
		Rnd:  rnd01,
		Val:  valFoo,
	})
	c.Assert(val, gc.IsNil)
	c.Assert(sid, gc.Equals, sid0)

	// Slot 4, round 1 from r2 is a quorum
	val, sid = learner.handleLearn(&px.Learn{
		ID:   r2id,
		Slot: 4,
		Rnd:  rnd01,
		Val:  valFoo,
	})
	c.Assert(val, gc.DeepEquals, &valFoo)
	c.Assert(sid, gc.Equals, sid4)

	// We should _not_ advance and catch-up _should_ trigger
	advance, startcu, cuslot := learner.learnValue(val, sid)
	c.Assert(advance, gc.Equals, false)
	c.Assert(startcu, gc.Equals, true)
	c.Assert(cuslot, gc.Equals, sid4)

	// Catch-Up range should be [1,3]
	ranges := learner.getUndecidedSlots(cuslot)
	c.Assert(ranges, gc.HasLen, 1)
	c.Assert(ranges[0].From, gc.Equals, sid1)
	c.Assert(ranges[0].To, gc.Equals, sid3)

	// Catch-Up request should contain range and replica id
	cureq, _ := learner.genCatchUpReq(cuslot)
	c.Assert(cureq, gc.NotNil)
	c.Assert(cureq.ID, gc.Equals, learner.id)
	c.Assert(cureq.Ranges, gc.HasLen, 1)
	c.Assert(cureq.Ranges[0].From, gc.Equals, sid1)
	c.Assert(cureq.Ranges[0].To, gc.Equals, sid3)
}

func (ls *lrnSuite) TestHandleCatchUpReq(c *gc.C) {
	// Learner at r0, set a slot map with history
	learner := NewMultiLearner(ppThreeNodesNonLr)
	learner.slots = ls.lsm

	// Catch-up request from r1 with range [1,3]
	cureq := px.CatchUpRequest{
		ID: r1id,
		Ranges: []px.RangeTuple{
			{From: 1, To: 3},
		},
	}

	// Catch-up response should not be nil, contain correct values for
	// range [1,3] and contain correct destination id

	curesp, destID := learner.handleCatchUpReq(&cureq)
	c.Assert(curesp, gc.NotNil)
	c.Assert(curesp.Vals, gc.HasLen, 3)
	c.Assert(curesp.Vals[0].Slot, gc.Equals, sid1)
	c.Assert(curesp.Vals[1].Slot, gc.Equals, sid2)
	c.Assert(curesp.Vals[2].Slot, gc.Equals, sid3)
	c.Assert(curesp.Vals[0].Val, gc.DeepEquals, valFoo)
	c.Assert(curesp.Vals[1].Val, gc.DeepEquals, valBar)
	c.Assert(curesp.Vals[2].Val, gc.DeepEquals, valFoo)
	c.Assert(destID, gc.Equals, r1id)
}

func (*lrnSuite) TestApplyCatchUpResp(c *gc.C) {
	// Learner with no history, next expected is slot 1
	learner := NewMultiLearner(ppThreeNodesNonLr)

	// A catch-up response for slot [1,3]
	cresp := &px.CatchUpResponse{
		ID: r1id,
		Vals: []px.ResponseTuple{
			{Slot: 1, Val: valFoo},
			{Slot: 2, Val: valBar},
			{Slot: 3, Val: valFoo},
		},
	}

	// Handle catch-up response
	learner.handleCatchUpResp(cresp)

	// Check that state for slot [1,3] has been set correctly
	s1 := learner.slots.GetSlot(1)
	c.Assert(s1.Learned, gc.Equals, true)
	c.Assert(s1.LearnedVal, gc.DeepEquals, valFoo)
	s2 := learner.slots.GetSlot(2)
	c.Assert(s2.Learned, gc.Equals, true)
	c.Assert(s2.LearnedVal, gc.DeepEquals, valBar)
	s3 := learner.slots.GetSlot(3)
	c.Assert(s3.Learned, gc.Equals, true)
	c.Assert(s3.LearnedVal, gc.DeepEquals, valFoo)
}

// -----------------------------------------------------------------------
// Tests: Catch-up (LR)

func (*lrnSuite) TestTiggerCatchUpLr(c *gc.C) {
	// We expect next decided to be slot 1
	learner := NewMultiLearner(ppThreeNodesLr)

	// Slot 4, round 1 from r1 is not a quorum
	val, sid := learner.handleLearn(&px.Learn{
		ID:          r1id,
		Slot:        4,
		Rnd:         rnd01,
		Val:         valFoo,
		EpochVector: []grp.Epoch{0, 0, 0},
	})
	c.Assert(val, gc.IsNil)
	c.Assert(sid, gc.Equals, sid0)

	// Slot 4, round 1 from r2 is a quorum
	val, sid = learner.handleLearn(&px.Learn{
		ID:          r2id,
		Slot:        4,
		Rnd:         rnd01,
		Val:         valFoo,
		EpochVector: []grp.Epoch{0, 0, 0},
	})
	c.Assert(val, gc.DeepEquals, &valFoo)
	c.Assert(sid, gc.Equals, sid4)

	// We should _not_ advance and catch-up _should_ trigger
	advance, startcu, cuslot := learner.learnValue(val, sid)
	c.Assert(advance, gc.Equals, false)
	c.Assert(startcu, gc.Equals, true)
	c.Assert(cuslot, gc.Equals, sid4)

	// Catch-Up range should be [1,3]
	ranges := learner.getUndecidedSlots(cuslot)
	c.Assert(ranges, gc.HasLen, 1)
	c.Assert(ranges[0].From, gc.Equals, sid1)
	c.Assert(ranges[0].To, gc.Equals, sid3)

	// Catch-Up request should contain range and replica id
	cureq, _ := learner.genCatchUpReq(cuslot)
	c.Assert(cureq, gc.NotNil)
	c.Assert(cureq.ID, gc.Equals, learner.id)
	c.Assert(cureq.Ranges, gc.HasLen, 1)
	c.Assert(cureq.Ranges[0].From, gc.Equals, sid1)
	c.Assert(cureq.Ranges[0].To, gc.Equals, sid3)
}

func (ls *lrnSuite) TestHandleCatchUpReqLr(c *gc.C) {
	// Learner at r0, set a slot map with history
	learner := NewMultiLearner(ppThreeNodesLr)
	learner.slotsLr = ls.lsmLr

	// Catch-up request from r1 with range [1,3]
	cureq := px.CatchUpRequest{
		ID: r1id,
		Ranges: []px.RangeTuple{
			{From: 1, To: 3},
		},
	}

	// Catch-up response should not be nil, contain correct values for
	// range [1,3] and contain correct destination id

	curesp, destID := learner.handleCatchUpReq(&cureq)
	c.Assert(curesp, gc.NotNil)
	c.Assert(curesp.Vals, gc.HasLen, 3)
	c.Assert(curesp.Vals[0].Slot, gc.Equals, sid1)
	c.Assert(curesp.Vals[1].Slot, gc.Equals, sid2)
	c.Assert(curesp.Vals[2].Slot, gc.Equals, sid3)
	c.Assert(curesp.Vals[0].Val, gc.DeepEquals, valFoo)
	c.Assert(curesp.Vals[1].Val, gc.DeepEquals, valBar)
	c.Assert(curesp.Vals[2].Val, gc.DeepEquals, valFoo)
	c.Assert(destID, gc.Equals, r1id)
}

func (*lrnSuite) TestApplyCatchUpRespLr(c *gc.C) {
	// Learner with no history, next expected is slot 1
	learner := NewMultiLearner(ppThreeNodesLr)

	// A catch-up response for slot [1,3]
	cresp := &px.CatchUpResponse{
		ID: r1id,
		Vals: []px.ResponseTuple{
			{Slot: 1, Val: valFoo},
			{Slot: 2, Val: valBar},
			{Slot: 3, Val: valFoo},
		},
	}

	// Handle catch-up response
	learner.handleCatchUpResp(cresp)

	// Check that state for slot [1,3] has been set correctly
	s1 := learner.slotsLr.GetSlot(1)
	c.Assert(s1.Learned, gc.Equals, true)
	c.Assert(s1.LearnedVal, gc.DeepEquals, valFoo)
	s2 := learner.slotsLr.GetSlot(2)
	c.Assert(s2.Learned, gc.Equals, true)
	c.Assert(s2.LearnedVal, gc.DeepEquals, valBar)
	s3 := learner.slotsLr.GetSlot(3)
	c.Assert(s3.Learned, gc.Equals, true)
	c.Assert(s3.LearnedVal, gc.DeepEquals, valFoo)
}
