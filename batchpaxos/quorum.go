package batchpaxos

import (
	"github.com/relab/goxos/grp"
)

// A structure for storing the state of a batch -- whether it has been executed
// and what messages we have received with this batch id.
type BatchState struct {
	Execute        bool
	Executed       bool
	ExecuteRanges  RangeMap
	Learns         []BatchLearnMsg
	LearnsNotified bool
}

// A batch quorum checker (BQC) is used to check for quorum for a particular batch
// id.
type BatchQuorumChecker struct {
	States     map[uint]*BatchState
	QuorumSize uint
}

// Create a new BQC given a quorum size.
func NewBatchQuorumChecker(qsize uint) *BatchQuorumChecker {
	return &BatchQuorumChecker{
		States:     make(map[uint]*BatchState),
		QuorumSize: qsize,
	}
}

// Pull out the state for a given batch id, create a new state if it doesn't
// already exist.
func (bqc *BatchQuorumChecker) GetState(batchid uint) *BatchState {
	_, ok := bqc.States[batchid]
	if !ok {
		bqc.States[batchid] = &BatchState{}
	}
	return bqc.States[batchid]
}

// Add a learn to the batch id state.
func (bqc *BatchQuorumChecker) AddLearn(msg BatchLearnMsg) {
	state := bqc.GetState(msg.BatchID)
	state.Learns = append(state.Learns, msg)
}

// Check if a quorum exists for a given batch id, returns true if there exists a
// quorum, false otherwise.
func (bqc *BatchQuorumChecker) LearnQuorumExists(batchid uint) bool {
	state := bqc.GetState(batchid)
	uniques := make(map[grp.ID]bool)

	for _, msg := range state.Learns {
		uniques[msg.ID] = true
	}

	quorum := len(uniques) >= int(bqc.QuorumSize)

	if quorum && !state.LearnsNotified {
		state.LearnsNotified = true
		return true
	}

	return false
}

// Perform an intersection of all of the learn messages for a batch id. Returns the
// potential batchpoint or nil otherwise.
func (bqc *BatchQuorumChecker) Intersect(batchid uint) map[string]uint32 {
	state := bqc.GetState(batchid)
	count := make(map[string]int)

	for _, msg := range state.Learns {
		for cid := range msg.HSS {
			count[cid]++
		}
	}

	mins := make(map[string]uint32)

	for cid, ct := range count {
		if ct == len(state.Learns) {
			min := findMin(state.Learns, cid)
			mins[cid] = min
		}
	}

	return mins
}

func findMin(msgs []BatchLearnMsg, cid string) uint32 {
	var min uint32
	first := true

	for _, msg := range msgs {
		if first {
			first = false
			min = msg.HSS[cid]
		} else if msg.HSS[cid] < min {
			min = msg.HSS[cid]
		}
	}

	return min
}
