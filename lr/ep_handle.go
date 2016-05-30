package lr

import (
	"github.com/relab/goxos/config"
	px "github.com/relab/goxos/paxos"

	"github.com/relab/goxos/Godeps/_workspace/src/github.com/golang/glog"
)

func (rh *ReplacementHandler) handleEpochPromise(ep EpochPromise) {
	glog.V(2).Infoln("received", ep)

	if !rh.replacer {
		glog.V(2).Info("not a replacer node, ignoring epoch promise")
		return
	}

	// Add sender to NodeMap even if we're activated. This is how we (for
	// now) get to know each node's id.
	if err := rh.grpmgr.NodeMap().Add(ep.ID, ep.Sender); err != nil {
		glog.Warningln("epoch promise sender not added to nodemap:", err)
	}

	if rh.activated {
		glog.V(2).Info("already activated, ignoring epoch promise")
		return
	}

	// Store EpochPromise
	rh.epochPromises = append(rh.epochPromises, &ep)

	// Is it not possible to obtain a valid quorum due to size?
	if uint(len(rh.epochPromises)) < rh.grpmgr.Quorum() {
		glog.V(2).Infof("need %d epoch promises before staring to check for "+
			"a valid quorum", rh.grpmgr.Quorum()-uint(len(rh.epochPromises)))
		return
	}

	validQuorum, vqIndices := extractValidQuorumEp(rh.epochPromises, rh.grpmgr.Quorum())
	if !validQuorum {
		glog.V(2).Info("could not extract a valid quorum from " +
			"current set of epoch promises")
		return
	}

	accState := extractAcceptorState(rh.epochPromises, vqIndices)
	glog.V(2).Info("setting acceptor state")
	rh.acceptor.SetState(accState)

	// Startegies: see comments in pe_handle.go
	switch rh.config.GetInt("LRStrategy", config.DefLRStrategy) {
	case 1, 2:
		rh.acceptor.SetLowSlot(ep.SlotMarker)
	case 3:
		rh.acceptor.SetLowSlot(accState.MaxSeen)
	}

	glog.V(2).Info("signaling activated")

	// Disabled for the current implementation using TCP
	//rh.cancelActivationTimer <- true

	rh.activated = true
	close(rh.activatedSignal)
}

func extractAcceptorState(eps []*EpochPromise, vqIndices []int) *px.AcceptorSlotMap {
	glog.V(2).Info("extracting acceptor state from epoch promises")
	slotMap := px.NewAcceptorSlotMap()
	for _, idx := range vqIndices {
		if eps[idx].AccState.Rnd.Compare(slotMap.Rnd) == 1 {
			slotMap.Rnd = eps[idx].AccState.Rnd
		}
		if eps[idx].AccState.MaxSeen > slotMap.MaxSeen {
			slotMap.MaxSeen = eps[idx].AccState.MaxSeen
		}
		for _, epAccSlot := range eps[idx].AccState.Slots {
			currentSlot := slotMap.GetSlot(epAccSlot.ID)
			if epAccSlot.VRnd.Compare(currentSlot.VRnd) >= 0 &&
				epAccSlot.VRnd.Compare(currentSlot.VRnd) != 0 {
				currentSlot = epAccSlot
			}
		}
	}

	return slotMap
}
