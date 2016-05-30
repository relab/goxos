package arec

import (
	"time"

	"github.com/relab/goxos/grp"
	"github.com/relab/goxos/net"

	"github.com/relab/goxos/elog"
	e "github.com/relab/goxos/elog/event"

	"github.com/relab/goxos/Godeps/_workspace/src/github.com/golang/glog"
)

const (
	connectionwait = 20 * time.Millisecond
	connwaittimes  = 30
)

func (rh *AReconfHandler) checkOutsideActivation(ac *Activation) bool {

	// Is the activation for an old Epoch?
	if ac.Epoch < rh.id.Epoch {
		glog.V(2).Infoln("aborting activation in old epoch")
		return false
	}

	if ac.Epoch == rh.id.Epoch && rh.stateful {
		glog.V(2).Infoln("aborting redundent activation in current epoch")
		return false
	}

	for ep := range ac.Confs {
		if _, ok := rh.confs[ep]; !ok && ep > ac.Epoch {
			rh.confs[ep] = ac.Confs[ep]
			if ep > rh.maxEpochSeen {
				rh.maxEpochSeen = ep
			}
		}
	}

	return true
}

func (rh *AReconfHandler) handleActivation(ac *Activation) {
	glog.V(2).Infoln("received", ac)

	// If we are still valid, make the acceptor stop.
	if rh.valid && rh.stateful {
		rh.acceptorState = rh.acceptor.GetState(rh.adu.Value(), rh.acceptorRelease)
	}

	rh.changeEpoch(ac.Epoch)

	rh.acceptorState = ac.AccState
	rh.accFirstSlot = ac.AduSlot + 1
	rh.acceptor.SetState(ac.AccState)
	rh.acceptor.SetLowSlot(rh.accFirstSlot)
	rh.proposer.SetNextSlot(rh.accFirstSlot)

	for id := range rh.confs[ac.Epoch] {
		rh.fd.SetAlive(id)
	}

	if rh.maxEpochSeen <= ac.Epoch {
		rh.valid = true
		rh.confs = make(map[grp.Epoch]map[grp.ID]grp.Node)
		if rh.stateful {
			rh.acceptorRelease <- true
		} else {
			close(rh.runPaxosChan)
			rh.stateful = true
		}
		elog.Log(e.NewEvent(e.ARecRestart))
	} else {
		rh.valid = false
		rh.stateful = true
		glog.V(2).Infoln("sending Cpromise to higher confs")
		nextEpoch := rh.maxEpochSeen
		for ep := range rh.confs {
			if ep <= rh.id.Epoch {
				delete(rh.confs, ep)
			} else if ep < nextEpoch {
				nextEpoch = ep
			}
			//Send CPromise to nextEpoch
		}
	}
}

//Change the local epoch.
func (rh *AReconfHandler) changeEpoch(epoch grp.Epoch) {
	if _, ok := rh.confs[epoch]; !ok {
		//Can the following happen?
		glog.Errorln("received activation for Epoch I do not know")
		return
	}

	glog.V(2).Info("requesting hold from grpmgr")
	holdChan := make(chan bool)
	rh.grpmgr.RequestHold(holdChan)
	defer close(holdChan)

	getconn := rh.reconnect(epoch)

	holdChan <- false

	rh.id.Epoch = epoch
	rh.grpmgr.SetNewNodeMap(rh.confs[epoch])
	rh.grpmgr.SetID(rh.id)

	rh.waitForFullConnection(getconn)
}

//Important! This has to be done, while the grpmgr is on Hold.
//We reconnect everybody to get the new ids into the new connections.
func (rh *AReconfHandler) reconnect(epoch grp.Epoch) (notconn []grp.ID) {
	pid := rh.id.PaxosID
	notconn = make([]grp.ID, 0, pid)
	for gcid := range rh.confs[epoch] {
		if pid < gcid.PaxosID {
			glog.V(2).Infof("connecting to %v\n", gcid)
			conn, err := net.GxConnectTo(rh.confs[epoch][gcid], grp.NewID(rh.id.PaxosID, epoch), gcid, rh.dmx)
			if err != nil {
				glog.V(2).Infof("could not connect to %v", gcid)
				//should we retry later?
			} else if err = net.AddToConnections(conn, true); err != nil {
				glog.Fatalln("error when adding replacer connection", err)
			}
		} else if pid > gcid.PaxosID {
			notconn = append(notconn, gcid)
		}
	}
	return notconn
}

func (rh *AReconfHandler) waitForFullConnection(getconn []grp.ID) {
	for i := 0; i < connwaittimes; i++ {
		notconn := net.CheckConnections(getconn)
		if len(notconn) == 0 {
			glog.V(3).Infoln("we are fully connected")
			return
		}
		glog.V(2).Infof("we are not connected to %v, waiting %v", notconn, connectionwait)
		time.Sleep(connectionwait)
	}
	glog.V(2).Infoln("waited 5 times, still not fully connected")
	return
}

//For new Replica.
func (rh *AReconfHandler) newconnect(epoch grp.Epoch) {
	var keys []grp.ID
	for key := range rh.confs[epoch] {
		keys = append(keys, key)
	}
	notconn := net.CheckConnections(keys)
	if len(notconn) == 0 {
		return
	}
	glog.V(1).Infoln("missing connections to %v", notconn)
	for _, id := range notconn {
		if id.PaxosID > rh.id.PaxosID {
			glog.V(2).Infof("connecting to %v\n", id)
			conn, err := net.GxConnectTo(rh.confs[epoch][id], grp.NewID(rh.id.PaxosID, epoch), id, rh.dmx)
			if err != nil {
				glog.V(2).Infof("could not connect to %v", id)
				//should we retry later?
			} else if err = net.AddToConnections(conn, true); err != nil {
				glog.Fatalln("error when adding replacer connection", err)
			}
		}
	}
}

/*
func (rh *AReconfHandler) reconnect(epoch grp.Epoch) {
	for gc := range rh.confs[epoch] {
		// Are we increasing the number of nodes?
		if int(gc.PaxosId) > int(rh.grpmgr.NodeMap().NrOfNodes())-1 {
			glog.V(5).Infof("increasing number of nodes")
			//TODO: Connect
			//Or is the node a replacement?
		} else if nw := rh.grpmgr.NodeMap().IsNew(gc, rh.confs[epoch][gc]); nw {
			glog.V(5).Infof("connecting to new node %v\n", gc)
			conn, err := net.GxConnectTo(rh.confs[epoch][gc], rh.id, grp.NewId(gc.PaxosId, epoch), rh.dmx)
			if err != nil {
				glog.V(5).Infof("could not connect to %v", gc)
				//should we retry later?
			} else if err = net.AddToConnections(conn, true); err != nil {
				glog.Fatalln("error when adding replacer connection", err)
			}
		}
	}
}
*/
