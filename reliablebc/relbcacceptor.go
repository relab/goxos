package reliablebc

import (
	"sync"

	"github.com/relab/goxos/config"
	"github.com/relab/goxos/grp"
	"github.com/relab/goxos/net"
	px "github.com/relab/goxos/paxos"

	"github.com/relab/goxos/Godeps/_workspace/src/github.com/golang/glog"
)

type RelAcceptor struct {
	id          grp.ID
	nAcceptors  uint
	nFaulty     uint
	dmx         net.Demuxer
	rounds      *AcceptorStorage
	actors      []grp.ID
	authmgr     *AuthManager
	bcast       chan<- interface{}
	ucast       chan<- net.Packet
	publishChan <-chan Publish
	prepareChan <-chan Prepare
	commitChan  <-chan CommitA
	valReqChan  <-chan ValReq
	started     bool
	stop        chan bool
	stopCheckIn *sync.WaitGroup
}

// Construct a new RelAcceptor
func NewRelAcceptor(pp *px.Pack) *RelAcceptor {
	return &RelAcceptor{
		id:          pp.ID,
		dmx:         pp.Dmx,
		nAcceptors:  pp.NrOfAcceptors,
		nFaulty:     (pp.NrOfAcceptors - 1) / 3,
		rounds:      NewAcceptorStorage(uint(pp.Config.GetInt("alpha", config.DefAlpha)+10) * 3),
		authmgr:     NewAuthManager(pp.ID, pp.Config),
		actors:      pp.Gm.NodeMap().AcceptorIDs(),
		ucast:       pp.Ucast,
		bcast:       pp.Bcast,
		started:     false,
		stop:        make(chan bool),
		stopCheckIn: pp.StopCheckIn,
	}
}

func (a *RelAcceptor) Start() {
	if a.started {
		glog.V(2).Info("ignoring start request")
		return
	}
	a.registerChannels()
	glog.V(1).Info("starting")
	a.started = true
	go func() {
		defer a.stopCheckIn.Done()
		for {
			select {
			//Receive publish
			case publish := <-a.publishChan:
				prepare := a.handlePublish(&publish)
				if prepare != nil {
					a.sendPrepare(prepare)
				}
			//Receive prepare
			case pmsg := <-a.prepareChan:
				prepare, commit := a.handlePrepare(&pmsg)
				if prepare != nil {
					a.sendPrepare(prepare)
				}
				if commit != nil {
					a.sendCommit(commit)
				}
			//Receive commit
			case cmsg := <-a.commitChan:
				prepare, commit := a.handleCommit(&cmsg)
				if prepare != nil {
					a.sendPrepare(prepare)
				}
				if commit != nil {
					a.sendCommit(commit)
				}
			//Receive value request from learner
			case reqRnd := <-a.valReqChan:
				a.sendVal(reqRnd.Rnd)
			//Receive stop signal
			case <-a.stop:
				glog.V(2).Info("exiting")
				a.started = false
				return
			}
		}
	}()
}

//Register channels with the demuxer
func (a *RelAcceptor) registerChannels() {
	glog.V(2).Infoln("Registering channels")
	publishChan := make(chan Publish, 64)
	a.publishChan = publishChan
	a.dmx.RegisterChannel(publishChan)
	prepareChan := make(chan Prepare, 128)
	a.prepareChan = prepareChan
	a.dmx.RegisterChannel(prepareChan)
	commitChan := make(chan CommitA, 128)
	a.commitChan = commitChan
	a.dmx.RegisterChannel(commitChan)
	valReqChan := make(chan ValReq, 32)
	a.valReqChan = valReqChan
	a.dmx.RegisterChannel(valReqChan)
}

//Stops the acceptor
func (a *RelAcceptor) Stop() {
	if a.started {
		a.stop <- true
	}
}

// ****************************************************************
// Handling publishes
// ****************************************************************
//Handles incoming publish message and returns a prepare message
func (a *RelAcceptor) handlePublish(msg *Publish) *Prepare {
	glog.V(2).Infoln("Received publish for round", msg.Rnd)
	//If the prepare has wrong mac, drop
	if !a.authmgr.checkMAC(
		msg.ID,
		msg.Hmac,
		a.authmgr.hashToBytes(msg.Val.Hash()),
		msg.Rnd,
	) {
		return nil
	}
	//If it's an old publish, discard
	if !a.rounds.IsValid(msg.Rnd) {
		glog.Warningln("Received old publish. Failed to store")
		return nil
	}
	//Store the publish
	rnd := a.rounds.Get(msg.Rnd)
	rnd.RndID = msg.Rnd
	rnd.Value = msg.Val
	if rnd.SentPrepare {
		return nil
	}
	return a.createPrepare(msg)
}

// Creates a prepare message for the received publish
// This function also increase the currentrnd
func (a *RelAcceptor) createPrepare(msg *Publish) *Prepare {
	//Create value hash
	hash := msg.Val.Hash()
	return &Prepare{
		ID:   a.id,
		Rnd:  msg.Rnd,
		Hash: hash,
		Hmac: nil,
	}
}

// ****************************************************************
// Handling prepares
// ****************************************************************

//Function for handling incoming prepare messages.
func (a *RelAcceptor) handlePrepare(msg *Prepare) (*Prepare, *Commit) {
	glog.V(2).Infoln("Received prepare from ", msg.ID, "for round", msg.Rnd)
	//If the prepare has wrong mac, drop
	if !a.authmgr.checkMAC(msg.ID, msg.Hmac, a.authmgr.hashToBytes(msg.Hash), msg.Rnd) {
		glog.Warningln("Received wrong HMAC from ", msg.ID)
		return nil, nil
	}
	//If this is a prepare for an old round, return
	if !a.rounds.IsValid(msg.Rnd) {
		return nil, nil
	}

	rnd := a.rounds.Get(msg.Rnd)
	// If no prepares (or commits) received for round, then initialize it
	if len(rnd.Prepares) == 0 && len(rnd.Commits) == 0 {
		rnd.RndID = msg.Rnd
		rnd.Prepares = make(map[grp.ID]Prepare)
		rnd.Commits = make(map[grp.ID]CommitA)
		rnd.SentPrepare = false
		rnd.SentCommit = false
	}
	//If commit is already sent, return
	if rnd.SentCommit {
		return nil, nil
	}

	//If already received prepare from that Id, return
	if _, ok := rnd.Prepares[msg.ID]; ok {
		glog.V(2).Infoln("Received duplicate prepare from ", msg.ID)
		return nil, nil
	}

	//Store the prepare and check if enough prepares
	rnd.Prepares[msg.ID] = *msg
	if a.comparePrepares(rnd.Prepares) {
		//If this acceptor somehow did not receive the publish,
		//create prepare based on decided prepare
		var prepare *Prepare
		if !rnd.SentPrepare {
			prepare = &Prepare{
				ID:   a.id,
				Rnd:  msg.Rnd,
				Hash: msg.Hash,
				Hmac: nil,
			}
		}
		//Create commit
		commit := a.createCommit(msg.Rnd, msg.Hash)
		return prepare, commit
	}
	return nil, nil
}

//Checks if there is enough prepares for a round. Returns true if they are
//the same. This function is only called after a new prepare is added, so it
//doesn't need to return which prepare is correct.
func (a *RelAcceptor) comparePrepares(prepares map[grp.ID]Prepare) bool {
	preparecount := make(map[uint32]uint)
	for _, prepare := range prepares {
		preparecount[prepare.Hash]++
	}
	for _, v := range preparecount {
		if v > (2 * a.nFaulty) {
			return true
		}
	}
	return false
}

//Sends prepare message
func (a *RelAcceptor) sendPrepare(msg *Prepare) {
	for _, id := range a.actors {
		if id == a.id {
			a.handlePrepare(msg)
		} else {
			a.send(id, Prepare{
				ID:   a.id,
				Rnd:  msg.Rnd,
				Hash: msg.Hash,
				Hmac: a.authmgr.generateMAC(id, a.authmgr.hashToBytes(msg.Hash), msg.Rnd),
			})
		}
	}
	//a.broadcast(msg)
	rnd := a.rounds.Get(msg.Rnd)
	rnd.SentPrepare = true
}

// ****************************************************************
// Handling commits
// ****************************************************************

//Function for handling incoming commit messages.
func (a *RelAcceptor) handleCommit(msg *CommitA) (*Prepare, *Commit) {
	glog.V(2).Infoln("Received commit from ", msg.ID, "for round", msg.Rnd)
	//If the commit has wrong mac, drop
	if !a.authmgr.checkMAC(msg.ID, msg.Hmac, a.authmgr.hashToBytes(msg.Hash), msg.Rnd) {
		glog.Warningln("Received wrong HMAC from ", msg.ID)
		return nil, nil
	}
	//If this is a commit for an old round, return
	if !a.rounds.IsValid(msg.Rnd) {
		return nil, nil
	}

	rnd := a.rounds.Get(msg.Rnd)
	// If no prepares or commits received for round, then initialize it
	if len(rnd.Prepares) == 0 && len(rnd.Commits) == 0 {
		rnd.RndID = msg.Rnd
		rnd.Prepares = make(map[grp.ID]Prepare)
		rnd.Commits = make(map[grp.ID]CommitA)
		rnd.SentPrepare = false
		rnd.SentCommit = false
	}
	//If commit is already sent, return
	if rnd.SentCommit {
		return nil, nil
	}

	//If already received commit from that Id, return
	if _, ok := rnd.Commits[msg.ID]; ok {
		return nil, nil
	}

	//Store the commit and check if enough commits
	rnd.Commits[msg.ID] = *msg
	if a.compareCommits(rnd.Commits) {
		//If this acceptor somehow did not receive the publish
		var prepare *Prepare
		if !rnd.SentPrepare {
			prepare = &Prepare{
				ID:   a.id,
				Rnd:  msg.Rnd,
				Hash: msg.Hash,
				Hmac: nil,
			}
		}
		//Create commit
		commit := a.createCommit(msg.Rnd, msg.Hash)
		return prepare, commit
	}
	return nil, nil
}

//Creates a commit message using the given round and hash
func (a *RelAcceptor) createCommit(rnd uint, hash uint32) *Commit {
	return &Commit{
		ID:   a.id,
		Rnd:  rnd,
		Hash: hash,
		Hmac: nil,
	}
}

//Broadcasts commit message, then sets sentcommit true for current round
func (a *RelAcceptor) sendCommit(msg *Commit) {
	for _, id := range a.actors {
		if id == a.id {
			a.handleCommit(&CommitA{
				ID:   a.id,
				Rnd:  msg.Rnd,
				Hash: msg.Hash,
				Hmac: a.authmgr.generateMAC(id, a.authmgr.hashToBytes(msg.Hash), msg.Rnd),
			})
			a.send(id, Commit{
				ID:   a.id,
				Rnd:  msg.Rnd,
				Hash: msg.Hash,
				Hmac: a.authmgr.generateMAC(id, a.authmgr.hashToBytes(msg.Hash), msg.Rnd),
			})
		} else {
			//Send commit to other acceptors
			a.send(id, CommitA{
				ID:   a.id,
				Rnd:  msg.Rnd,
				Hash: msg.Hash,
				Hmac: a.authmgr.generateMAC(id, a.authmgr.hashToBytes(msg.Hash), msg.Rnd),
			})
			//Send commit to other Learners
			a.send(id, Commit{
				ID:   a.id,
				Rnd:  msg.Rnd,
				Hash: msg.Hash,
				Hmac: a.authmgr.generateMAC(id, a.authmgr.hashToBytes(msg.Hash), msg.Rnd),
			})
		}
	}
	//a.broadcast(msg)
	rnd := a.rounds.Get(msg.Rnd)
	rnd.SentCommit = true
}

//Checks if there is enough commits for a round. Returns true if there are
//enough commits of the same value. This function is only called after a
//new commit is added, so it doesn't need to return which commit is correct
func (a *RelAcceptor) compareCommits(commits map[grp.ID]CommitA) bool {
	commitcount := make(map[uint32]uint)
	for _, commit := range commits {
		commitcount[commit.Hash]++
	}
	for _, v := range commitcount {
		if v > (a.nFaulty) {
			return true
		}
	}
	return false
}

// ****************************************************************
// Handling value requests
// ****************************************************************
func (a *RelAcceptor) getRndValue(rnd uint) px.Value {
	if !a.rounds.IsValid(rnd) {
		glog.Errorln("Value not stored in acceptor")
		return px.Value{}
	}
	return a.rounds.Get(rnd).Value
}

func (a *RelAcceptor) sendVal(rnd uint) {
	glog.V(2).Infoln("Received value request from learner for round", rnd)
	val := a.getRndValue(rnd)
	a.send(a.id, ValResp{rnd, val})
}

// ****************************************************************
// Network functions
// ****************************************************************
func (a *RelAcceptor) broadcast(msg interface{}) {
	a.bcast <- msg
}

func (a *RelAcceptor) send(id grp.ID, msg interface{}) {
	a.ucast <- net.Packet{DestID: id, Data: msg}
}

// ****************************************************************
// Not used by RelBC, but needed to fulfill the actor interface
// ****************************************************************
func (a *RelAcceptor) GetState(afterSlot px.SlotID, release <-chan bool) *px.AcceptorSlotMap {
	return nil
}

func (a *RelAcceptor) GetMaxSlot(release <-chan bool) *px.AcceptorSlotMap {
	return nil
}

func (a *RelAcceptor) SetState(slotmap *px.AcceptorSlotMap) {
}

func (a *RelAcceptor) SetLowSlot(slot px.SlotID) {
}
