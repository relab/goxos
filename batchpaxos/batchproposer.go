package batchpaxos

import (
	"sync"

	px "github.com/relab/goxos/paxos"
)

// Unused batch proposer -- but needed to fill actor interface requirements.
type BatchProposer struct {
	stopCheckIn *sync.WaitGroup
}

func NewBatchProposer(pp px.Pack) *BatchProposer {
	return &BatchProposer{
		stopCheckIn: pp.StopCheckIn,
	}
}

func (bp *BatchProposer) Start() {
}

func (bp *BatchProposer) Stop() {
	bp.stopCheckIn.Done()
}

func (bp *BatchProposer) SetNextSlot(ns px.SlotID) {
}
