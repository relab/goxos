package paxos

// A Paxos actor implements Start() and Stop() methods. Maybe this could
// be generalized as an interface for a Module which we Start() and Stop()
// in the server module.
type Actor interface {
	Start()
	Stop()
}

// The interface a Proposer must fulfill, also includes the base PaxosActor
// methods.
type Proposer interface {
	Actor
	SetNextSlot(ns SlotID)
}

// The interface an Acceptor must fulfill.
type Acceptor interface {
	Actor
	GetState(afterSlot SlotID, release <-chan bool) *AcceptorSlotMap
	GetMaxSlot(release <-chan bool) *AcceptorSlotMap
	SetState(slotmap *AcceptorSlotMap)
	SetLowSlot(slot SlotID)
}

// The interface a Learner must fulfill.
type Learner interface {
	Actor
}
