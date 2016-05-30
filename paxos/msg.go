package paxos

import (
	"encoding/gob"

	"github.com/relab/goxos/grp"
)

func init() {
	gob.Register(Prepare{})
	gob.Register(Promise{})
	gob.Register(Accept{})
	gob.Register(Learn{})
	gob.Register(CatchUpRequest{})
	gob.Register(CatchUpResponse{})
}

type Prepare struct {
	ID   grp.ID
	CRnd ProposerRound
	Slot SlotID
}

type Promise struct {
	ID          grp.ID
	Rnd         ProposerRound
	AccSlots    []AcceptorSlot
	EpochVector []grp.Epoch
}

type Accept struct {
	ID   grp.ID
	Slot SlotID
	Rnd  ProposerRound
	Val  Value
}

type Learn struct {
	ID          grp.ID
	Slot        SlotID
	Rnd         ProposerRound
	Val         Value
	EpochVector []grp.Epoch
}

type CatchUpRequest struct {
	ID     grp.ID
	Ranges []RangeTuple
}

type CatchUpResponse struct {
	ID   grp.ID
	Vals []ResponseTuple
}

type RangeTuple struct {
	From SlotID
	To   SlotID
}

type ResponseTuple struct {
	Slot SlotID
	Val  Value
}
