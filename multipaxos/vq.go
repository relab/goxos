package multipaxos

import (
	"github.com/relab/goxos/lr"
	"github.com/relab/goxos/paxos"
)

func extractVqLearns(msgs []*paxos.Learn, quorumSize uint) (bool, []int) {
	if len(msgs) == 0 || uint(len(msgs)) < quorumSize {
		return false, nil
	}

	ci := lr.NewCombinationIterator(len(msgs), int(quorumSize))
	for ci.HasNext() {
		if checkCombinationLrn(ci.S(), msgs) {
			return true, ci.S()
		}

		ci.Next()
	}

	return false, []int{}
}

func checkCombinationLrn(indices []int, msgs []*paxos.Learn) bool {
	for i := 0; i < len(indices); i++ {
		for j := i + 1; j < len(indices); j++ {
			if lrnConflict(msgs[indices[i]], msgs[indices[j]]) {
				return false
			}
		}
	}

	return true
}

func lrnConflict(a, b *paxos.Learn) bool {
	if a.ID.PaxosID == b.ID.PaxosID {
		return true
	}

	return a.ID.Epoch < b.EpochVector[a.ID.PaxosID] || b.ID.Epoch < a.EpochVector[b.ID.PaxosID]
}

func extractVqPromises(msgs []*paxos.Promise, quorumSize uint) (bool, []int) {
	if len(msgs) == 0 || uint(len(msgs)) < quorumSize {
		return false, nil
	}

	ci := lr.NewCombinationIterator(len(msgs), int(quorumSize))
	for ci.HasNext() {
		if checkCombinationPromises(ci.S(), msgs) {
			return true, ci.S()
		}

		ci.Next()
	}

	return false, []int{}
}

func checkCombinationPromises(indices []int, msgs []*paxos.Promise) bool {
	for i := 0; i < len(indices); i++ {
		for j := i + 1; j < len(indices); j++ {
			if promiseConflict(msgs[indices[i]], msgs[indices[j]]) {
				return false
			}
		}
	}

	return true
}

func promiseConflict(a, b *paxos.Promise) bool {
	if a.ID.PaxosID == b.ID.PaxosID {
		return true
	}

	return a.ID.Epoch < b.EpochVector[a.ID.PaxosID] || b.ID.Epoch < a.EpochVector[b.ID.PaxosID]
}
