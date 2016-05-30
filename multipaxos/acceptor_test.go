package multipaxos

import (
	"github.com/relab/goxos/grp"
	px "github.com/relab/goxos/paxos"

	gc "github.com/relab/goxos/Godeps/_workspace/src/gopkg.in/check.v1"
)

// -----------------------------------------------------------------------
// Initialization

type accSuite struct {
	asm *px.AcceptorSlotMap
}

var _ = gc.Suite(&accSuite{})

func (accs *accSuite) SetUpSuite(c *gc.C) {
	// Populate an acceptor slot map with history, slot 1-2 and 4
	accs.asm = px.NewAcceptorSlotMap()
	s1 := accs.asm.GetSlot(1)
	s1.VRnd = rnd11
	s1.VVal = valFoo
	s2 := accs.asm.GetSlot(2)
	s2.VRnd = rnd11
	s2.VVal = valBar
	s3 := accs.asm.GetSlot(4)
	s3.VRnd = rnd11
	s3.VVal = valFoo
}

// -----------------------------------------------------------------------
// Tests: Handle prepare messages

func (*accSuite) TestIgnorePromiseIfOldRound(c *gc.C) {
	// New acceptor with low slot 1 and rnd (0,0)
	acceptor := NewMultiAcceptor(ppThreeNodesNonLr)

	// Set rnd to (1,1)
	acceptor.slots.Rnd = rnd11

	// Prepare for round (0,1) should be ignored
	promise, _ := acceptor.handlePrepare(&px.Prepare{
		ID:   r0id,
		Slot: 1,
		CRnd: rnd01,
	})
	c.Assert(promise, gc.IsNil)
}

func (*accSuite) TestIgnorePromiseIfEqualRound(c *gc.C) {
	// New acceptor with low slot 1 and rnd (0,0)
	acceptor := NewMultiAcceptor(ppThreeNodesNonLr)

	// Set rnd to (1,1)
	acceptor.slots.Rnd = rnd11

	// Prepare for round (1,1) should be ignored
	promise, _ := acceptor.handlePrepare(&px.Prepare{
		ID:   r1id,
		Slot: 1,
		CRnd: rnd11,
	})
	c.Assert(promise, gc.IsNil)
}

func (*accSuite) TestSetRndIfCrndIsHigher(c *gc.C) {
	// New acceptor with low slot 1 and rnd (0,0)
	acceptor := NewMultiAcceptor(ppThreeNodesNonLr)

	// Set rnd to (1,1)
	acceptor.slots.Rnd = rnd11

	// Prepare for round (1,2) should set rnd to (1,2)
	_, _ = acceptor.handlePrepare(&px.Prepare{
		ID:   r1id,
		Slot: 1,
		CRnd: rnd12,
	})
	c.Assert(acceptor.slots.Rnd, gc.DeepEquals, rnd12)
}

func (*accSuite) TestEmptyAccSlotsInPromiseIfNoHistoryToSend(c *gc.C) {
	// New acceptor with low slot 1 and rnd (0,0)
	acceptor := NewMultiAcceptor(ppThreeNodesNonLr)

	// Set rnd to (1,1)
	acceptor.slots.Rnd = rnd11

	// Prepare for round (2,1) should produce a promise with no AccSlots
	// and destination r2id
	promise, dest := acceptor.handlePrepare(&px.Prepare{
		ID:   r2id,
		Slot: 1,
		CRnd: rnd21,
	})
	c.Assert(promise, gc.NotNil)
	c.Assert(promise.ID, gc.Equals, r0id)
	c.Assert(promise.Rnd, gc.Equals, rnd21)
	c.Assert(promise.AccSlots, gc.HasLen, 0)
	c.Assert(dest, gc.Equals, r2id)
}

func (accs *accSuite) TestNonEmptyAccSlotsInPromiseIfHistoryToSend(c *gc.C) {
	// New acceptor with low slot 1 and rnd (0,0)
	acceptor := NewMultiAcceptor(ppThreeNodesNonLr)

	// Use a slot map with history at slot 1,2 and 4.
	acceptor.slots = accs.asm

	// Set rnd to (1,1)
	acceptor.slots.Rnd = rnd11

	// Prepare for round (2,1) should produce a promise with non-empty
	// AccSlots (slot 1, 2 and 3) and destination r2id

	promise, dest := acceptor.handlePrepare(&px.Prepare{
		ID:   r2id,
		Slot: 1,
		CRnd: rnd21,
	})
	c.Assert(promise, gc.NotNil)
	c.Assert(promise.ID, gc.Equals, r0id)
	c.Assert(promise.Rnd, gc.Equals, rnd21)
	c.Assert(dest, gc.Equals, r2id)

	c.Assert(promise.AccSlots, gc.HasLen, 3)

	c.Assert(promise.AccSlots[0].ID, gc.Equals, sid1)
	c.Assert(promise.AccSlots[0].VRnd, gc.Equals,
		rnd11)
	c.Assert(promise.AccSlots[0].VVal, gc.DeepEquals, valFoo)

	c.Assert(promise.AccSlots[1].ID, gc.Equals, sid2)
	c.Assert(promise.AccSlots[1].VRnd, gc.Equals,
		rnd11)
	c.Assert(promise.AccSlots[1].VVal, gc.DeepEquals, valBar)

	c.Assert(promise.AccSlots[2].ID, gc.Equals, sid4)
	c.Assert(promise.AccSlots[2].VRnd, gc.Equals,
		rnd11)
	c.Assert(promise.AccSlots[2].VVal, gc.DeepEquals, valFoo)
}

// -----------------------------------------------------------------------
// Tests: Handle prepare messages with Live Replacement these are
// duplicates of the ones above, but additionally checking that the
// promise contain a correct epoch vector.

func (*accSuite) TestIgnorePromiseIfOldRoundLr(c *gc.C) {
	// New acceptor with low slot 1 and rnd (0,0)
	acceptor := NewMultiAcceptor(ppThreeNodesLr)

	// Set rnd to (1,1)
	acceptor.slots.Rnd = rnd11

	// Prepare for round (0,1) should be ignored
	promise, _ := acceptor.handlePrepare(&px.Prepare{
		ID:   r0id,
		Slot: 1,
		CRnd: rnd01,
	})
	c.Assert(promise, gc.IsNil)
}

func (*accSuite) TestIgnorePromiseIfEqualRoundLr(c *gc.C) {
	// New acceptor with low slot 1 and rnd (0,0)
	acceptor := NewMultiAcceptor(ppThreeNodesLr)

	// Set rnd to (1,1)
	acceptor.slots.Rnd = rnd11

	// Prepare for round (1,1) should be ignored
	promise, _ := acceptor.handlePrepare(&px.Prepare{
		ID:   r1id,
		Slot: 1,
		CRnd: rnd11,
	})
	c.Assert(promise, gc.IsNil)
}

func (*accSuite) TestSetRndIfCrndIsHigherLr(c *gc.C) {
	// New acceptor with low slot 1 and rnd (0,0)
	acceptor := NewMultiAcceptor(ppThreeNodesLr)

	// Set rnd to (1,1)
	acceptor.slots.Rnd = rnd11

	// Prepare for round (1,2) should set rnd to (1,2)
	_, _ = acceptor.handlePrepare(&px.Prepare{
		ID:   r1id,
		Slot: 1,
		CRnd: rnd12,
	})
	c.Assert(acceptor.slots.Rnd, gc.DeepEquals, rnd12)
}

func (*accSuite) TestEmptyAccSlotsInPromiseIfNoHistoryToSendLr(c *gc.C) {
	// New acceptor with low slot 1 and rnd (0,0)
	acceptor := NewMultiAcceptor(ppThreeNodesLr)

	// Set rnd to (1,1)
	acceptor.slots.Rnd = rnd11

	// Prepare for round (2,1) should produce a promise with no AccSlots
	// and destination r2id
	promise, dest := acceptor.handlePrepare(&px.Prepare{
		ID:   r2id,
		Slot: 1,
		CRnd: rnd21,
	})
	c.Assert(promise, gc.NotNil)
	c.Assert(promise.ID, gc.Equals, r0id)
	c.Assert(promise.Rnd, gc.Equals, rnd21)
	c.Assert(promise.AccSlots, gc.HasLen, 0)
	c.Assert(promise.EpochVector, gc.DeepEquals, []grp.Epoch{0, 0, 0})
	c.Assert(dest, gc.Equals, r2id)
}

func (accs *accSuite) TestNonEmptyAccSlotsInPromiseIfHistoryToSendLr(c *gc.C) {
	// New acceptor with low slot 1 and rnd (0,0)
	acceptor := NewMultiAcceptor(ppThreeNodesLr)

	// Use a slot map with history at slot 1,2 and 4.
	acceptor.slots = accs.asm

	// Set rnd to (1,1)
	acceptor.slots.Rnd = rnd11

	// Prepare for round (2,1) should produce a promise with non-empty
	// AccSlots (slot 1, 2 and 3) and destination r2id

	promise, dest := acceptor.handlePrepare(&px.Prepare{
		ID:   r2id,
		Slot: 1,
		CRnd: rnd21,
	})
	c.Assert(promise, gc.NotNil)
	c.Assert(promise.ID, gc.Equals, r0id)
	c.Assert(promise.Rnd, gc.Equals, rnd21)
	c.Assert(promise.EpochVector, gc.DeepEquals, []grp.Epoch{0, 0, 0})
	c.Assert(dest, gc.Equals, r2id)

	c.Assert(promise.AccSlots, gc.HasLen, 3)

	c.Assert(promise.AccSlots[0].ID, gc.Equals, sid1)
	c.Assert(promise.AccSlots[0].VRnd, gc.Equals,
		rnd11)
	c.Assert(promise.AccSlots[0].VVal, gc.DeepEquals, valFoo)

	c.Assert(promise.AccSlots[1].ID, gc.Equals, sid2)
	c.Assert(promise.AccSlots[1].VRnd, gc.Equals,
		rnd11)
	c.Assert(promise.AccSlots[1].VVal, gc.DeepEquals, valBar)

	c.Assert(promise.AccSlots[2].ID, gc.Equals, sid4)
	c.Assert(promise.AccSlots[2].VRnd, gc.Equals,
		rnd11)
	c.Assert(promise.AccSlots[2].VVal, gc.DeepEquals, valFoo)
}

// -----------------------------------------------------------------------
// Tests: Handle accept messages

func (*accSuite) TestIgnoreAcceptIfBelowLowSlot(c *gc.C) {
	// New acceptor with low slot 1 and rnd (0,0)
	acceptor := NewMultiAcceptor(ppThreeNodesNonLr)

	// Handling an accept for slot 0 should not result in a learn
	learn := acceptor.handleAccept(&px.Accept{
		ID:   r0id,
		Slot: 0,
		Rnd:  rnd01,
		Val:  valFoo,
	})
	c.Assert(learn, gc.IsNil)
}

func (*accSuite) TestIgnoreAcceptfOldRound(c *gc.C) {
	// New acceptor with low slot 1
	acceptor := NewMultiAcceptor(ppThreeNodesNonLr)

	// Set rnd to (1,1)
	acceptor.slots.Rnd = rnd11

	// Handling an accept for slot 1 with rnd (0,1) should not result in a learn
	learn := acceptor.handleAccept(&px.Accept{
		ID:   r0id,
		Slot: 0,
		Rnd:  rnd01,
		Val:  valFoo,
	})
	c.Assert(learn, gc.IsNil)
}

func (*accSuite) TestIgnoreAcceptIfEqualRound(c *gc.C) {
	// New acceptor with low slot 1
	acceptor := NewMultiAcceptor(ppThreeNodesNonLr)

	// Set rnd to (1,1)
	acceptor.slots.Rnd = rnd11

	// Handling an accept for slot 1 with rnd (1,1) should not result in a learn
	learn := acceptor.handleAccept(&px.Accept{
		ID:   r0id,
		Slot: 0,
		Rnd:  rnd11,
		Val:  valFoo,
	})
	c.Assert(learn, gc.IsNil)
}

func (*accSuite) TestGenLearnIfNoHistory(c *gc.C) {
	// New acceptor with low slot 1 and rnd (0,0)
	acceptor := NewMultiAcceptor(ppThreeNodesNonLr)

	// Handling an accept for slot 1 with rnd (0,1) should result in learn
	learn := acceptor.handleAccept(&px.Accept{
		ID:   r0id,
		Slot: 1,
		Rnd:  rnd01,
		Val:  valFoo,
	})
	c.Assert(learn, gc.NotNil)
	c.Assert(learn, gc.DeepEquals, &px.Learn{
		ID:   r0id,
		Slot: 1,
		Rnd:  rnd01,
		Val:  valFoo,
	})

	// Rnd should now be set to (0,1)
	c.Assert(acceptor.slots.Rnd, gc.Equals, rnd01)
}

func (*accSuite) TestGenLearnIfHigherRound(c *gc.C) {
	// New acceptor with low slot 1 and rnd (0,0)
	acceptor := NewMultiAcceptor(ppThreeNodesNonLr)

	// Set rnd to (1,1)

	// Handling an accept for slot 1 with rnd (2,1) should result in learn
	learn := acceptor.handleAccept(&px.Accept{
		ID:   r0id,
		Slot: 1,
		Rnd:  rnd21,
		Val:  valFoo,
	})
	c.Assert(learn, gc.NotNil)
	c.Assert(learn, gc.DeepEquals, &px.Learn{
		ID:   r0id,
		Slot: 1,
		Rnd:  rnd21,
		Val:  valFoo,
	})

	// Rnd should now be set to (2,1)
	c.Assert(acceptor.slots.Rnd, gc.Equals, rnd21)
}

// -----------------------------------------------------------------------
// Tests: Handle accept messages with Live Replacement enabled. These are
// duplicates of the ones above, but additionally checking that the
// learns contain a correct epoch vector.

func (*accSuite) TestIgnoreAcceptIfBelowLowSlotLr(c *gc.C) {
	// New acceptor with low slot 1 and rnd (0,0)
	acceptor := NewMultiAcceptor(ppThreeNodesLr)

	// Handling an accept for slot 0 should not result in a learn
	learn := acceptor.handleAccept(&px.Accept{
		ID:   r0id,
		Slot: 0,
		Rnd:  rnd01,
		Val:  valFoo,
	})
	c.Assert(learn, gc.IsNil)
}

func (*accSuite) TestIgnoreAcceptIfOldRoundLr(c *gc.C) {
	// New acceptor with low slot 1
	acceptor := NewMultiAcceptor(ppThreeNodesLr)

	// Set rnd to (1,1)
	acceptor.slots.Rnd = rnd11

	// Handling an accept for slot 1 with rnd (0,1) should not result in a learn
	learn := acceptor.handleAccept(&px.Accept{
		ID:   r0id,
		Slot: 0,
		Rnd:  rnd01,
		Val:  valFoo,
	})
	c.Assert(learn, gc.IsNil)
}

func (*accSuite) TestIgnoreAcceptIfEqualRoundLr(c *gc.C) {
	// New acceptor with low slot 1
	acceptor := NewMultiAcceptor(ppThreeNodesLr)

	// Set rnd to (1,1)
	acceptor.slots.Rnd = rnd11

	// Handling an accept for slot 1 with rnd (1,1) should not result in a learn
	learn := acceptor.handleAccept(&px.Accept{
		ID:   r0id,
		Slot: 0,
		Rnd:  rnd11,
		Val:  valFoo,
	})
	c.Assert(learn, gc.IsNil)
}

func (*accSuite) TestGenLearnIfNoHistoryLr(c *gc.C) {
	// New acceptor with low slot 1 and rnd (0,0)
	acceptor := NewMultiAcceptor(ppThreeNodesLr)

	// Handling an accept for slot 1 with rnd (0,1) should result in learn
	learn := acceptor.handleAccept(&px.Accept{
		ID:   r0id,
		Slot: 1,
		Rnd:  rnd01,
		Val:  valFoo,
	})
	c.Assert(learn, gc.NotNil)
	c.Assert(learn, gc.DeepEquals, &px.Learn{
		ID:          r0id,
		Slot:        1,
		Rnd:         rnd01,
		Val:         valFoo,
		EpochVector: []grp.Epoch{0, 0, 0},
	})

	// Rnd should now be set to (0,1)
	c.Assert(acceptor.slots.Rnd, gc.Equals, rnd01)
}

func (*accSuite) TestGenLearnIfHigherRoundLr(c *gc.C) {
	// New acceptor with low slot 1 and rnd (0,0)
	acceptor := NewMultiAcceptor(ppThreeNodesLr)

	// Set rnd to (1,1)

	// Handling an accept for slot 1 with rnd (2,1) should result in learn
	learn := acceptor.handleAccept(&px.Accept{
		ID:   r0id,
		Slot: 1,
		Rnd:  rnd21,
		Val:  valFoo,
	})
	c.Assert(learn, gc.NotNil)
	c.Assert(learn, gc.DeepEquals, &px.Learn{
		ID:          r0id,
		Slot:        1,
		Rnd:         rnd21,
		Val:         valFoo,
		EpochVector: []grp.Epoch{0, 0, 0},
	})

	// Rnd should now be set to (2,1)
	c.Assert(acceptor.slots.Rnd, gc.Equals, rnd21)
}
