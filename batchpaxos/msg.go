package batchpaxos

import (
	"encoding/gob"

	"github.com/relab/goxos/grp"
)

func init() {
	gob.Register(BatchLearnMsg{})
	gob.Register(BatchCommitMsg{})

	gob.Register(UpdateRequestMsg{})
	gob.Register(UpdateReplyMsg{})
}

// Main protocol

type BatchLearnMsg struct {
	ID      grp.ID
	BatchID uint
	HSS     BatchPoint
}

type BatchCommitMsg struct {
	ID      grp.ID
	BatchID uint
	Ranges  RangeMap
}

// Update protocol

type UpdateRequestMsg struct {
	ID      grp.ID
	BatchID uint
	Ranges  RangeMap
}

type UpdateReplyMsg struct {
	ID      grp.ID
	BatchID uint
	Ranges  RangeMap
	Values  UpdateValueMap
}
