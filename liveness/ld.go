package liveness

import (
	"github.com/relab/goxos/grp"
)

type LeaderDetector interface {
	Start()
	Stop()
	SubscribeToPaxosLdMsgs(name string) <-chan grp.ID
	SubscribeToReplacementLdMsgs(name string) <-chan grp.ID
	PaxosLeader() grp.ID
	ReplacementLeader() grp.ID
}
