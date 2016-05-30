package arec

import (
	"github.com/relab/goxos/grp"
	px "github.com/relab/goxos/paxos"

	"github.com/relab/goxos/elog"
	e "github.com/relab/goxos/elog/event"

	"github.com/relab/goxos/Godeps/_workspace/src/github.com/golang/glog"
)

func (rh *AReconfHandler) handleCPromise(cp CPromise) *Activation {
	glog.V(2).Infoln("received", cp)

	if cp.NewEpoch < rh.id.Epoch {
		glog.V(2).Info("ignoring old cpromise")
		return nil
	}

	if cp.NewEpoch == rh.id.Epoch && rh.stateful {
		glog.V(2).Info("ignoring old cpromise")
		return nil
	}

	for epoch := range cp.Confs {
		if epoch >= cp.NewEpoch { //Store Conf
			if _, ok := rh.confs[epoch]; !ok {
				rh.confs[epoch] = cp.Confs[epoch]
				if epoch > rh.maxEpochSeen {
					rh.maxEpochSeen = epoch
				}
			}
		} else if epoch < cp.NewEpoch && cp.ID.Epoch < epoch {
			//Send NewConf message to configuration with epoch.
			//This cPromise is not stored.
			return nil
		}
	}

	// Store CPromise
	var promises []*CPromise
	if cpros, exists := rh.cPromises[cp.ID.Epoch]; exists {
		promises = append(cpros, &cp)
		rh.cPromises[cp.ID.Epoch] = promises
	} else {
		promises = make([]*CPromise, 0, cp.QuorumSize)
		rh.cPromises[cp.ID.Epoch] = append(promises, &cp)
		glog.V(2).Infof("cPromise in new Epoch, not yet a quorum")
		return nil
	}

	// Is it not possible to obtain a valid quorum due to size?
	if uint(len(promises)) < cp.QuorumSize {
		glog.V(2).Infof("need %d c promises before staring to check for "+
			"a valid quorum", cp.QuorumSize-uint(len(promises)))
		return nil
	}

	validQuorum, quorum := check(promises, cp.ID.Epoch)
	if !validQuorum {
		glog.V(2).Info("could not extract a valid quorum from " +
			"current set of cpromises")
		return nil
	}

	elog.Log(e.NewEvent(e.ARecActivatedFromCPs))
	rh.acceptorState, rh.accFirstSlot = extractAcceptorState(quorum)
	glog.V(2).Info("found a state")

	glog.V(2).Info("send Activation")
	return &Activation{cp.NewEpoch, rh.accFirstSlot, rh.acceptorState, rh.confs}

	// Disabled for the current implementation using TCP
	//rh.cancelActivationTimer <- true

}

func check(msgs []*CPromise, epoch grp.Epoch) (bool, []*CPromise) {
	returmsgs := make([]*CPromise, 0, len(msgs))
	ids := make(map[grp.PaxosID]bool, len(msgs))
	for _, cp := range msgs {
		if _, exists := ids[cp.ID.PaxosID]; cp.ID.Epoch == epoch && !exists {
			ids[cp.ID.PaxosID] = true
			returmsgs = append(returmsgs, cp)
		} else {
			glog.V(2).Infof("found a cp with id %v in epochs for %v",
				cp.ID, epoch)
		}

	}
	if len(returmsgs) == 0 || uint(len(returmsgs)) < returmsgs[0].QuorumSize {
		return false, returmsgs
	}

	return true, returmsgs

}

//TODO: we should decouple rnd and the slots. After that, change also this.
func extractAcceptorState(cps []*CPromise) (slotMap *px.AcceptorSlotMap, firstSlot px.SlotID) {
	glog.V(2).Info("extracting acceptor state from cpromises")
	slotMap = px.NewAcceptorSlotMap()
	for _, cp := range cps {
		if cp.AduSlot > firstSlot {
			firstSlot = cp.AduSlot
		}
		if cp.AccState.Rnd.Compare(slotMap.Rnd) == 1 {
			slotMap.Rnd = cp.AccState.Rnd
		}
		for id, accSlot := range cp.AccState.Slots {
			slot := slotMap.GetSlot(id)
			if accSlot.VRnd.Compare(slot.VRnd) >= 0 &&
				accSlot.VRnd.Compare(slot.VRnd) != 0 {
				slot = accSlot
			}
		}
	}
	slotMap.Rnd.ID.Epoch = cps[0].NewEpoch

	return
}
