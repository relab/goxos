package arec

import (
	"github.com/relab/goxos/grp"
	"github.com/relab/goxos/net"

	"github.com/relab/goxos/elog"
	e "github.com/relab/goxos/elog/event"

	"github.com/relab/goxos/Godeps/_workspace/src/github.com/golang/glog"
)

func (rh *AReconfHandler) handleReconfMsg(rm *ReconfMsg) *CPromise {
	//rh.eventLogger.Log(e.NewEvent(e.RecvPrepareEpoch))
	glog.V(2).Infoln("received reconf msg:", rm)

	// If we're a replacer node, are we statefull ?
	if !rh.stateful {
		glog.V(2).Info("we're not stateful, dont know what to do with this message.")
		return nil
	}

	// Is new epoch larger?
	if !(rm.NewEpoch > rh.id.Epoch) {
		glog.V(2).Info("new epoch lower then mine, ignoring")
		return nil
	}

	//Running Paxos?
	if rh.valid {
		elog.Log(e.NewEvent(e.ARecStopPaxos))
		// Stop Paxos, by stopping acceptor and get state.
		glog.V(2).Info("requesting acceptor state")
		//defer close(acceptorRelease)
		rh.acceptorState = rh.acceptor.GetState(rm.AduSlot, rh.acceptorRelease)
		rh.valid = false
	} else {
		for epoch := range rh.confs {
			if rh.id.Epoch < epoch && epoch < rm.NewEpoch {
				rh.updateConfs(rm)
				//We have to first get activated in one epoch, to go to the next.
				glog.V(2).Info("exists other reconf with lower epoch, ignoring")
				return nil
			}
		}

	}

	rh.updateConfs(rm)
	//TODO:Start NewConfTimeout
	//	broadcast a NewConf Message.

	// Are we being replaced?
	if _, ok := rm.OldIds[rh.id]; !ok {
		//We are being replaced, what should we do?
		return nil
	}

	// Our id and node info
	node, found := rh.grpmgr.NodeMap().LookupNode(rh.id)
	if !found {
		glog.Fatal("running node not found in nodemap")
		return nil
	}

	glog.V(2).Info("generating cpromise")
	cp := CPromise{rh.id, rh.grpmgr.Quorum(), node, rm.NewEpoch, rh.confs, rm.AduSlot, *rh.acceptorState}

	return &cp
}

func (rh *AReconfHandler) sendCP(cp *CPromise, oldids *map[grp.ID]bool) {

	newnodes := cp.Confs[cp.NewEpoch]
	oldPax := make([]bool, len(newnodes))
	for id := range *oldids {
		oldPax[id.PaxosID] = true
		rh.ucast <- net.Packet{DestID: id, Data: *cp}
	}

	for id, node := range newnodes {
		if !oldPax[id.PaxosID] {
			glog.V(2).Infof("connecting to new node %v", id)
			conn, err := net.GxConnectEphemeral(node, rh.id)
			if err != nil {
				glog.Warningf("connection attempt to %v failed", id)
				conn, err = net.GxConnectEphemeral(node, rh.id)
				if err != nil {
					glog.Errorf("sending cpromise to %v aborted, reason %v:", id, err)
					continue
				}
			}

			glog.V(2).Info("sending cpromise")
			if err = conn.Write(*cp); err != nil {
				glog.Errorln("handling cpromise aborted, reason:", err)
				conn.Close()
			}
			conn.Close()
		}
	}

}

func (rh *AReconfHandler) updateConfs(rm *ReconfMsg) {
	rh.confs[rm.NewEpoch] = rm.NodeMap
	if rm.NewEpoch > rh.maxEpochSeen {
		rh.maxEpochSeen = rm.NewEpoch
	}
}
