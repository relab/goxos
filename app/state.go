package app

import (
	"github.com/relab/goxos/paxos"
)

type StateReq struct {
	respChan chan<- State
}

func NewStateReq(respChan chan<- State) StateReq {
	return StateReq{respChan}
}

func (sr *StateReq) RespChan() chan<- State {
	return sr.respChan
}

type State struct {
	SlotMarker paxos.SlotID
	State      []byte
}

func NewState(slotMarker paxos.SlotID, state []byte) State {
	return State{slotMarker, state}
}
