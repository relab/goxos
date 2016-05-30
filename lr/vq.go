package lr

func extractValidQuorumEp(msgs []*EpochPromise, quorumSize uint) (bool, []int) {
	if len(msgs) == 0 || uint(len(msgs)) < quorumSize {
		return false, nil
	}

	ci := NewCombinationIterator(len(msgs), int(quorumSize))
	for ci.HasNext() {
		if checkCombination(ci.S(), msgs) {
			return true, ci.S()
		}

		ci.Next()
	}

	return false, []int{}
}

func checkCombination(indices []int, msgs []*EpochPromise) bool {
	for i := 0; i < len(indices); i++ {
		for j := i + 1; j < len(indices); j++ {
			if epConflict(msgs[indices[i]], msgs[indices[j]]) {
				return false
			}
		}
	}

	return true
}

func epConflict(a, b *EpochPromise) bool {
	if a.ID.PaxosID == b.ID.PaxosID {
		return true
	}

	return a.ID.Epoch < b.EpochVector[a.ID.PaxosID] || b.ID.Epoch < a.EpochVector[b.ID.PaxosID]
}
