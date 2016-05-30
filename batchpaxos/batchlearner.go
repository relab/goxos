package batchpaxos

import (
	"sync"

	px "github.com/relab/goxos/paxos"
)

// Unused batch learner actor -- we only use batch acceptors in this protocol.
type BatchLearner struct {
	stopCheckIn *sync.WaitGroup
}

func NewBatchLearner(pp px.Pack) *BatchLearner {
	return &BatchLearner{
		stopCheckIn: pp.StopCheckIn,
	}
}

func (bl *BatchLearner) Start() {
}

func (bl *BatchLearner) Stop() {
	bl.stopCheckIn.Done()
}

func (bl *BatchLearner) HandleLearn(msg px.Learn) error {
	return nil
}
