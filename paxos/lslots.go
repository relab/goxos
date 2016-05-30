package paxos

import (
	"github.com/relab/goxos/grp"
)

// The state that a Learner needs to maintain for every Slot.
type LearnerSlot struct {
	ID         SlotID
	Learned    bool
	Rnd        ProposerRound
	Learns     []*Learn
	Votes      uint
	LearnedVal Value
	Decided    bool
}

// A LearnerSlotMap allows easy access to the slots of a Learner.
type LearnerSlotMap struct {
	Slots   map[SlotID]*LearnerSlot
	MaxSeen SlotID
}

// Construct a new LearnerSlotMap.
func NewLearnerSlotMap() *LearnerSlotMap {
	return &LearnerSlotMap{Slots: make(map[SlotID]*LearnerSlot)}
}

// Given a SlotId, return the associated LearnerSlot, as well as possibly
// modify MaxSeen.
func (llrsm *LearnerSlotMap) GetSlot(id SlotID) *LearnerSlot {
	_, ok := llrsm.Slots[id]
	if !ok {
		llrsm.Slots[id] = &LearnerSlot{ID: id}
	}
	if id > llrsm.MaxSeen {
		llrsm.MaxSeen = id
	}

	return llrsm.Slots[id]
}

// The state that a LrLearner needs to maintain for every Slot.
type LearnerLrSlot struct {
	ID         SlotID
	Learned    bool
	Rnd        ProposerRound
	Learns     []*Learn
	Votes      uint
	Voted      map[grp.ID]bool
	LearnedVal Value
	Decided    bool
	CheckVQ    bool
}

// A LearnerSlotMap allows easy access to the slots of a Learner.
type LearnerLrSlotMap struct {
	Slots   map[SlotID]*LearnerLrSlot
	MaxSeen SlotID
}

// Construct a new LearnerSlotMap.
func NewLearnerLrSlotMap() *LearnerLrSlotMap {
	return &LearnerLrSlotMap{Slots: make(map[SlotID]*LearnerLrSlot)}
}

// Given a SlotId, return the associated LearnerSlot, as well as possibly
// modify MaxSeen.
func (llrsm *LearnerLrSlotMap) GetSlot(id SlotID) *LearnerLrSlot {
	_, ok := llrsm.Slots[id]
	if !ok {
		llrsm.Slots[id] = &LearnerLrSlot{
			ID:    id,
			Voted: make(map[grp.ID]bool),
		}
	}

	if id > llrsm.MaxSeen {
		llrsm.MaxSeen = id
	}

	return llrsm.Slots[id]
}
