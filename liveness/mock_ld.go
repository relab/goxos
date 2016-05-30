package liveness

import (
	"github.com/relab/goxos/grp"
)

type MockLD struct{}

func NewMockLD() *MockLD {
	return &MockLD{}
}

func (mld *MockLD) Start() {}

func (mld *MockLD) Stop() {}

func (mld *MockLD) SubscribeToPaxosLdMsgs(name string) <-chan grp.ID {
	return nil
}

func (mld *MockLD) SubscribeToReplacementLdMsgs(name string) <-chan grp.ID {
	return nil
}

func (mld *MockLD) PaxosLeader() grp.ID {
	return grp.ID{}
}

func (mld *MockLD) ReplacementLeader() grp.ID {
	return grp.ID{}
}
