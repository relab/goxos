package paxos

// The state that a Proposer needs to maintain for every Slot.
type ProposerSlot struct {
	ID        SlotID
	Proposal  *Accept
	SentCount uint8
}

// A PropserSlotMap allows easy access to the slots of a Proposer.
type ProposerSlotMap struct {
	Slots   map[SlotID]*ProposerSlot
	MaxSeen SlotID
}

// Construct a new ProposerSlotMap.
func NewProposerSlotMap() *ProposerSlotMap {
	return &ProposerSlotMap{Slots: make(map[SlotID]*ProposerSlot)}
}

// Given a SlotId, return the associated ProposerSlot, as well as possibly
// modify MaxSeen.
func (sm *ProposerSlotMap) GetSlot(id SlotID) *ProposerSlot {
	_, ok := sm.Slots[id]
	if !ok {
		sm.Slots[id] = &ProposerSlot{ID: id}
	}
	if id > sm.MaxSeen {
		sm.MaxSeen = id
	}
	return sm.Slots[id]
}
