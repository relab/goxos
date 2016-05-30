package lr

import (
	"math/rand"
	"time"

	"github.com/relab/goxos/config"
	"github.com/relab/goxos/net"
	"github.com/relab/goxos/paxos"

	"github.com/relab/goxos/elog"
	e "github.com/relab/goxos/elog/event"

	"github.com/relab/goxos/Godeps/_workspace/src/github.com/golang/glog"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func (rh *ReplacementHandler) handlePrepareEpoch(pe PrepareEpoch) {
	elog.Log(e.NewEvent(e.LRPrepareEpochRecv))
	glog.V(2).Infoln("received prepare epoch:", pe)

	// If we're a replacer node, are we activated ?
	if rh.replacer && !rh.activated {
		glog.V(2).Info("we're not activated, forwarding our EPs if any")
		if err := rh.forwardEpochPromises(pe); err != nil {
			glog.Errorln("forwarding EPs failed:", err)
		}
	}

	// Is epoch for PaxosId larger than already present?
	if !(pe.ReplacerID.Epoch > rh.grpmgr.Epochs()[pe.ReplacerID.PaxosID]) {
		glog.V(2).Info("epoch for id lower or equal to present, ignoring")
		return
	}

	// Are we being replaced?
	if pe.ReplacerID.PaxosID == rh.id.PaxosID {
		glog.Warning("received prepare epoch for our paxos id,",
			"returning and expecting terminate request")
		return
	}

	// Only for use of Approach 2.:
	if pe.SlotMarker > rh.adu.Value() {
		glog.Warning("received prepare epoch with slot marker higher than adu.",
			"Resending to self.")
		rh.prepareEpochChan <- pe
		return
	}

	glog.V(2).Info("connecting to replacer node")
	if rh.config.GetDuration("LRExpRndSleep", config.DefLRExpRndSleep) != 0 {
		if rand.Intn(2) == 1 {
			elog.Log(e.NewEvent(e.LRPreConnectSleep))
			glog.V(2).Infoln("performing pre-connect sleep, duration:",
				rh.config.GetDuration("LRExpRndSleep", config.DefLRExpRndSleep))
			time.Sleep(rh.config.GetDuration("LRExpRndSleep", config.DefLRExpRndSleep))
		}
	}
	conn, err := net.GxConnectTo(pe.ReplacerNode, rh.id, pe.ReplacerID, rh.dmx)
	if err != nil {
		glog.Errorln("handling epoch promise aborted, reason:", err)
		return
	}

	glog.V(2).Info("requesting hold from grpmgr")
	holdChan := make(chan bool)
	rh.grpmgr.RequestHold(holdChan)
	defer close(holdChan)

	glog.V(2).Info("requesting acceptor state")
	var acceptorState *paxos.AcceptorSlotMap
	acceptorRelease := make(chan bool)
	defer close(acceptorRelease)

	/*
		Important notice regarding obtaining acceptor state for
		the EpochPromise message.

		There are three different approaches:

			1. Request all acceptor state above the slot for which
			the new replica received an application state.  This
			slot is contained in the PrepareEpoch message.  This
			state may be quite large if initialization of the
			replacer replica requires a considerable amount of time
			(for example due to a large amount of application state
			transfer). This is because the replicated service can
			decide on consensus instances concurrently while the
			new replica is initialized.

			2. Request acceptor state above the adu (all decided
			upto) slot on the replacement leader. This slot is
			contained in the PrepareEpoch message. The state can
			still contain many instances, but fewer than in the
			first approach.  This approach also relies on the check
			above.

			3. In this apprach we only ask for the highest slot
			(MaxSeen) in the acceptor state. The method GetMaxSlot
			returns an acceptor state with rnd and maxSeen, but an
			empty slotmap.  To change between method number 3 and
			2, you have to also change code in ep_handle.go.

		The different approaches are further discussed in our Paper.
	*/

	switch rh.config.GetInt("LRStrategy", config.DefLRStrategy) {
	case 1, 2:
		// Request all Acceptor state from the slot contained in the
		// PrepareEpoch message:
		acceptorState = rh.acceptor.GetState(pe.SlotMarker, acceptorRelease)
	case 3:
		// Only get highest nonempty slotid:
		acceptorState = rh.acceptor.GetMaxSlot(acceptorRelease)
	}

	// Our id and node info
	id := rh.id
	node, found := rh.grpmgr.NodeMap().LookupNode(id)
	if !found {
		glog.Fatal("running node not found in nodemap")
	}

	var ep EpochPromise
	glog.V(2).Info("generating epoch promise")
	switch rh.config.GetInt("LRStrategy", config.DefLRStrategy) {
	case 1, 2:
		ep = EpochPromise{id, node, rh.grpmgr.Epochs(), pe.SlotMarker, *acceptorState}
	case 3:
		ep = EpochPromise{id, node, rh.grpmgr.Epochs(), acceptorState.MaxSeen, *acceptorState}
	}

	/*
		Notes for sending epoch promise as discussed in code review
		meeting Leander/Hein/Tormod. Async. resend until success or
		until new PE for same PxId? Less Disruption?
	*/
	go func() {
		if err = conn.Write(ep); err != nil {
			glog.Errorln("error sending epoch promise:", err)
			conn.Close()
		}
		glog.V(2).Info("sending epoch promise")
	}()

	glog.V(2).Info("updating node map")
	err = rh.grpmgr.NodeMap().Replace(pe.ReplacerID, pe.ReplacerNode)
	if err != nil {
		glog.Errorln("handling epoch promise aborted, reason:", err)
		return
	}

	glog.V(2).Info("passing connection to network modules")
	if err = net.AddToConnections(conn, true); err != nil {
		glog.Fatalln("error when adding replacer connection to network connections:", err)
		// TODO(tormod): How to handle this?
		// Need to revert replacement call to NodeMap.
	}

	elog.Log(e.NewEvent(e.LRActivatedFromPE))
	glog.V(2).Infoln("installed replacer", pe.ReplacerNode, "with id", pe.ReplacerID)
}

func (rh *ReplacementHandler) forwardEpochPromises(pe PrepareEpoch) error {
	if len(rh.epochPromises) == 0 {
		return nil
	}

	conn, err := net.GxConnectTo(pe.ReplacerNode, rh.id, pe.ReplacerID, rh.dmx)
	if err != nil {
		return err
	}

	defer conn.Close()

	feps := ForwardedEpSet{EpSet: rh.epochPromises}
	if err = conn.Enc.Encode(feps); err != nil {
		return err
	}

	return nil
}
