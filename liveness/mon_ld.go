package liveness

import (
	"sync"

	"github.com/relab/goxos/grp"

	"github.com/relab/goxos/Godeps/_workspace/src/github.com/golang/glog"
)

// The state of the leader detector (Ld)
type MonarchicalLD struct {
	pleader        grp.ID
	rleader        grp.ID
	fd             *Fd
	grpmgr         grp.GroupManager
	grpSubscriber  grp.Subscriber
	suspected      map[grp.ID]bool
	fdChan         <-chan FdMsg
	pldSubscribers map[string]chan grp.ID
	rldSubscribers map[string]chan grp.ID
	checkLeaders   func()
	stop           chan bool
	stopCheckIn    *sync.WaitGroup
}

// Construct a new leader detector.
func NewMonarchicalLD(gm grp.GroupManager, fd *Fd, stopCheckIn *sync.WaitGroup) *MonarchicalLD {
	ld := &MonarchicalLD{
		fd:             fd,
		grpmgr:         gm,
		suspected:      make(map[grp.ID]bool),
		pldSubscribers: make(map[string]chan grp.ID),
		rldSubscribers: make(map[string]chan grp.ID),
		stop:           make(chan bool),
		stopCheckIn:    stopCheckIn,
	}

	if !ld.grpmgr.LrEnabled() && !ld.grpmgr.ArEnabled() {
		ld.checkLeaders = ld.checkPaxosLeader
	} else {
		ld.checkLeaders = ld.checkLeadersLrAr
	}

	ld.checkLeaders()

	return ld
}

// Initialize the leader detector and spawn a goroutine handling messages. The
// leader detector subscribes to messages from the failure detector.
func (mld *MonarchicalLD) Start() {
	glog.V(1).Info("starting")
	mld.grpSubscriber = mld.grpmgr.SubscribeToHold("ld")
	mld.fdChan = mld.fd.SubscribeToFdMsgs("ld")
	go func() {
		defer mld.stopCheckIn.Done()
		mld.checkLeaders()
		for {
			select {
			case fdmsg := <-mld.fdChan:
				mld.handleFdMsg(fdmsg)
			case grpPrepare := <-mld.grpSubscriber.PrepareChan():
				mld.handleGrpHold(grpPrepare)
			case <-mld.stop:
				glog.V(1).Info("exiting")
				return
			}
		}
	}()
}

// Stop the leader detector.
func (mld *MonarchicalLD) Stop() {
	mld.stop <- true
}

// Modules may register interest in receiving notifications when a new leader is
// elected. A name is needed to uniquely identify the module who is interested. This
// method returns a channel where information about newly elected leaders can be read
// from.
func (mld *MonarchicalLD) SubscribeToPaxosLdMsgs(name string) <-chan grp.ID {
	ldChan := make(chan grp.ID, 4)
	mld.pldSubscribers[name] = ldChan
	return ldChan
}

// Modules may also register interest in receiving notifications when a new replacement
// leader is elected. This is used by the Live Replacement module.
func (mld *MonarchicalLD) SubscribeToReplacementLdMsgs(name string) <-chan grp.ID {
	rldChan := make(chan grp.ID, 4)
	mld.rldSubscribers[name] = rldChan
	return rldChan
}

// Get the id of the replica considered as the Paxos leader.
func (mld *MonarchicalLD) PaxosLeader() grp.ID {
	return mld.pleader
}

// Get the id of the replica considered as the replacement leader.
func (mld *MonarchicalLD) ReplacementLeader() grp.ID {
	return mld.rleader
}

func (mld *MonarchicalLD) checkPaxosLeader() {
	if newLeader := mld.maxRank(); newLeader != mld.pleader {
		mld.pleader = newLeader
		mld.publishPaxosTrust(newLeader)
		glog.V(2).Infof("now trusting %v as paxos leader", newLeader)
	}
}

func (mld *MonarchicalLD) checkLeadersLrAr() {
	newPaxosLeader, newReplacementLeader := mld.maxRankLr()

	if newPaxosLeader != mld.pleader {
		mld.pleader = newPaxosLeader
		mld.publishPaxosTrust(newPaxosLeader)
		glog.V(2).Infof("now trusting %v as paxos leader", newPaxosLeader)
	}

	if newReplacementLeader != mld.rleader {
		mld.rleader = newReplacementLeader
		mld.publishReplacementTrust(newReplacementLeader)
		glog.V(2).Infof("now trusting %v as replacement leader", newReplacementLeader)
	}
}

func (mld *MonarchicalLD) handleFdMsg(fdmsg FdMsg) {
	switch fdmsg.Event {
	case Suspect:
		glog.V(2).Infoln("received", fdmsg)
		mld.suspected[fdmsg.ID] = true
		mld.checkLeaders()
	case Restore:
		glog.V(2).Infoln("received", fdmsg)
		delete(mld.suspected, fdmsg.ID)
		mld.checkLeaders()
	default:
		glog.Warning("unknown message from fd")
	}
}

func (mld *MonarchicalLD) publishPaxosTrust(id grp.ID) {
	for _, sub := range mld.pldSubscribers {
		sub <- id
	}
}

func (mld *MonarchicalLD) publishReplacementTrust(id grp.ID) {
	for _, sub := range mld.rldSubscribers {
		sub <- id
	}
}

func (mld *MonarchicalLD) maxRank() grp.ID {
	// Paxos leader: node with the highest paxos id
	paxosLeader := grp.MinID()

	if mld.grpmgr.NodeMap().Len() == 0 {
		glog.Fatal("no nodes to rank")
	}

	for _, id := range mld.grpmgr.NodeMap().ProposerIDs() {
		if _, suspected := mld.suspected[id]; id.PaxosID > paxosLeader.PaxosID && !suspected {
			paxosLeader = id
		}
	}

	glog.V(2).Infoln("highest paxos rank was", paxosLeader)

	return paxosLeader
}

func (mld *MonarchicalLD) maxRankLr() (paxosLeader, replacementLeader grp.ID) {
	// Paxos leader: node with lowest epoch and highest paxos id.
	paxosLeader = grp.MinID()

	// Replacement leader: node with lowest epoch and lowest paxos id.
	replacementLeader = grp.MaxID()

	if mld.grpmgr.NodeMap().Len() == 0 {
		glog.Fatal("no nodes to rank")
	}

	// What is the lowest epoch
	minSeenEpoch := grp.MaxEpoch
	for _, id := range mld.grpmgr.NodeMap().ProposerIDs() {
		if id.Epoch < minSeenEpoch {
			minSeenEpoch = id.Epoch
		}
	}

	// Which nodes have the lowest epoch
	nodesWithMinEpoch := make(map[grp.ID]bool)
	for _, id := range mld.grpmgr.NodeMap().ProposerIDs() {
		if id.Epoch == minSeenEpoch {
			nodesWithMinEpoch[id] = true
		}
	}

	// Find highest and lowest paxos id
	for id := range nodesWithMinEpoch {
		if _, suspected := mld.suspected[id]; id.PaxosID > paxosLeader.PaxosID && !suspected {
			paxosLeader = id
		}
		if _, suspected := mld.suspected[id]; id.PaxosID < replacementLeader.PaxosID && !suspected {
			replacementLeader = id
		}
	}

	glog.V(2).Infof("node ids %v, epochs %v",
		mld.grpmgr.NodeMap().IDs(), mld.grpmgr.NodeMap().Epochs())
	glog.V(2).Infof("nr of nodes with lowest epoch: %v",
		len(nodesWithMinEpoch))

	glog.V(2).Infoln("highest paxos rank was", paxosLeader)
	glog.V(2).Infoln("highest replacement rank was", replacementLeader)

	return paxosLeader, replacementLeader
}

func (mld *MonarchicalLD) handleGrpHold(gp *sync.WaitGroup) {
	glog.V(2).Info("grpmgr hold req")
	gp.Done()
	<-mld.grpSubscriber.ReleaseChan()
	glog.V(2).Info("grpmgr release")
	mld.checkLeaders()
}
