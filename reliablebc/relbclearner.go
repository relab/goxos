package reliablebc

import (
	"sync"

	"github.com/relab/goxos/config"
	"github.com/relab/goxos/grp"
	"github.com/relab/goxos/net"
	px "github.com/relab/goxos/paxos"

	"github.com/relab/goxos/Godeps/_workspace/src/github.com/golang/glog"
)

type RelLearner struct {
	id          grp.ID
	nAcceptors  uint
	nFaulty     uint
	dmx         net.Demuxer
	rounds      *LearnerStorage
	maxlearned  uint
	alpha       uint
	actors      []grp.ID
	authmgr     *AuthManager
	bcast       chan<- interface{}
	ucast       chan<- net.Packet
	dcdChan     chan<- *px.Value
	commitChan  <-chan Commit
	valRespChan <-chan ValResp
	started     bool
	stop        chan bool
	stopCheckIn *sync.WaitGroup
}

// Construct a new RelLearner
func NewRelLearner(pp *px.Pack) *RelLearner {
	return &RelLearner{
		id:          pp.ID,
		dmx:         pp.Dmx,
		maxlearned:  0,
		alpha:       uint(pp.Config.GetInt("alpha", config.DefAlpha)),
		nAcceptors:  pp.NrOfAcceptors,
		nFaulty:     (pp.NrOfAcceptors - 1) / 3,
		rounds:      NewLearnerStorage((uint(pp.Config.GetInt("alpha", config.DefAlpha)) + 10) * 3),
		actors:      pp.Gm.NodeMap().AcceptorIDs(),
		authmgr:     NewAuthManager(pp.ID, pp.Config),
		ucast:       pp.Ucast,
		bcast:       pp.Bcast,
		dcdChan:     pp.DcdChan,
		started:     false,
		stop:        make(chan bool),
		stopCheckIn: pp.StopCheckIn,
	}
}

func (l *RelLearner) Start() {
	if l.started {
		glog.V(2).Infoln("ignoring start request")
		return
	}
	l.registerChannels()
	glog.V(1).Infoln("starting")
	l.started = true
	go func() {
		defer l.stopCheckIn.Done()
		for {
			select {
			//Receive commit
			case cmsg := <-l.commitChan:
				l.handleCommit(&cmsg)
			//Receive value from acceptor
			case valresp := <-l.valRespChan:
				l.storeVal(valresp)
				l.sendDecided(valresp.Rnd)
			//Receive stop signal
			case <-l.stop:
				glog.V(2).Infoln("exiting")
				l.started = false
				return
			}
		}
	}()
}

//Register channels with the demuxer
func (l *RelLearner) registerChannels() {
	commitChan := make(chan Commit, 128)
	l.commitChan = commitChan
	l.dmx.RegisterChannel(commitChan)
	valRespChan := make(chan ValResp, 32)
	l.valRespChan = valRespChan
	l.dmx.RegisterChannel(valRespChan)
}

//Stops the learner
func (l *RelLearner) Stop() {
	l.stop <- true
}

// ****************************************************************
// Handling commits
// ****************************************************************

//Function that handles incoming commit messages
//returns true if the message caused a learn
func (l *RelLearner) handleCommit(msg *Commit) bool {
	glog.V(2).Infof("Received commit for round %d from id %d\n", msg.Rnd, msg.ID.PaxosID)
	//If the commit has wrong mac, drop
	if !l.authmgr.checkMAC(msg.ID, msg.Hmac, l.authmgr.hashToBytes(msg.Hash), msg.Rnd) {
		glog.Warningln("Received wrong HMAC from ", msg.ID)
		return false
	}
	//If this is a commit for an old round, return
	if !l.rounds.IsValid(msg.Rnd) {
		return false
	}

	rnd := l.rounds.Get(msg.Rnd)
	// If no commits received for round, then initialize it
	if len(rnd.Commits) == 0 {
		rnd.RndID = msg.Rnd
		rnd.Commits = make(map[grp.ID]Commit)
		rnd.Decided = false
		rnd.SntReq = false
		rnd.Learned = false
	}
	//If value already decided, return
	if rnd.Decided {
		return false
	}

	//If already received commit from that Id, return
	if _, ok := rnd.Commits[msg.ID]; ok {
		glog.V(2).Infoln("Received duplicate commit from ", msg.ID)
		return false
	}

	//Store the commit and check if enough commits
	rnd.Commits[msg.ID] = *msg
	rnd.Decided = l.compareCommits(rnd.Commits)
	//If the value is decided on, then ask for value
	if rnd.Decided && !rnd.SntReq {
		l.sendValReq(msg.Rnd)
		rnd.SntReq = true
	}
	return true
}

//Checks if there is enough commits for a round. Returns true if there are
//enough commits of the same value. This function is only called after a
//new commit is added, so it doesn't need to return which commit is correct
func (l *RelLearner) compareCommits(commits map[grp.ID]Commit) bool {
	commitcount := make(map[uint32]uint)
	for _, commit := range commits {
		commitcount[commit.Hash]++
	}
	for _, v := range commitcount {
		if v > (2 * l.nFaulty) {
			return true
		}
	}
	return false
}

// ****************************************************************
// After decide
// ****************************************************************

//Sends a request to the acceptor for the needed value
func (l *RelLearner) sendValReq(rnd uint) {
	l.send(l.id, ValReq{rnd})
}

//Stores a value for the appropriate round
func (l *RelLearner) storeVal(resp ValResp) {
	glog.V(2).Infoln("Received value response from acceptor")
	if !l.rounds.IsValid(resp.Rnd) {
		glog.Warningln("Received old value. Failed to learn")
		return
	}
	l.rounds.Get(resp.Rnd).Val = resp.Val
	l.rounds.Get(resp.Rnd).Learned = true
}

// ****************************************************************
// After learn
// ****************************************************************

//Sends decides for the learned rounds in the correct order
func (l *RelLearner) sendDecided(rnd uint) {
	if l.maxlearned+l.alpha < rnd {
		l.maxlearned++
	} else if l.maxlearned+1 != rnd {
		glog.V(2).Infoln("Learned for round out of order, holding responses")
		return
	}
	i := l.maxlearned
	for l.rounds.Get(rnd).Learned {
		//Send decide to server
		l.dcdChan <- &l.rounds.Get(rnd).Val
		//Send decide to publisher
		l.send(l.actors[0], DcdRnd{l.maxlearned})
		rnd++
		l.maxlearned++
	}
	glog.V(2).Infof("Sent decided values for %d round(s)\n", l.maxlearned-i)

}

// ****************************************************************
// Network functions
// ****************************************************************
func (l *RelLearner) broadcast(msg interface{}) {
	l.bcast <- msg
}

func (l *RelLearner) send(id grp.ID, msg interface{}) {
	l.ucast <- net.Packet{DestID: id, Data: msg}
}
