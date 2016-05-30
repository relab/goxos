package paxos

func CopyAccSlotMapRange(afterSlotID SlotID, sourceSlotMap *AcceptorSlotMap) *AcceptorSlotMap {
	sm := NewAcceptorSlotMap()
	if sourceSlotMap == nil {
		return sm
	}

	sm.Rnd = sourceSlotMap.Rnd
	for slotCounter := afterSlotID + 1; slotCounter <= sourceSlotMap.MaxSeen; slotCounter++ {
		if slot, found := sourceSlotMap.Slots[slotCounter]; found {
			sm.Slots[slotCounter] = slot
			sm.MaxSeen = slotCounter
		}
	}

	return sm
}
