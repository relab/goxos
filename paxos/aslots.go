package paxos

// In MultiPaxos we need to maintain multiple instances of Paxos. We call
// each of these instances a slot. In this case, this AcceptorSlot contains
// the slot state that is used by an Acceptor.
type AcceptorSlot struct {
	ID   SlotID
	VRnd ProposerRound // The highest proposal number in which we have casted a vote
	VVal Value         // The value voted for with proposal number vrnd
}

// An AcceptorSlotMap allows easy maintenance of slots. It contains a map
// from SlotId to an AcceptorSlot, as well as the highest accessed SlotId.
type AcceptorSlotMap struct {
	Slots   map[SlotID]*AcceptorSlot
	Rnd     ProposerRound // The highest proposal number in which we have participated
	MaxSeen SlotID
}

// Construct a new AcceptorSlotMap.
func NewAcceptorSlotMap() *AcceptorSlotMap {
	return &AcceptorSlotMap{
		Slots: make(map[SlotID]*AcceptorSlot),
		Rnd:   ZeroRound,
	}
}

// Given a SlotId, get the corresponding AcceptorSlot. Also may modify MaxSeen,
// the highest accessed SlotId.
func (sm *AcceptorSlotMap) GetSlot(id SlotID) *AcceptorSlot {
	_, ok := sm.Slots[id]
	if !ok {
		sm.Slots[id] = &AcceptorSlot{ID: id}
	}
	if id > sm.MaxSeen {
		sm.MaxSeen = id
	}
	return sm.Slots[id]
}
