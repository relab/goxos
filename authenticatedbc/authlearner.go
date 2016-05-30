package authenticatedbc

import (
	"sync"

	"github.com/relab/goxos/config"
	"github.com/relab/goxos/grp"
	"github.com/relab/goxos/net"
	px "github.com/relab/goxos/paxos"

	"github.com/relab/goxos/Godeps/_workspace/src/github.com/golang/glog"
)

type AuthLearner struct {
	id          grp.ID
	nAcceptors  uint
	nFaulty     uint
	dmx         net.Demuxer
	rounds      *RoundStorage
	maxlearned  uint
	alpha       uint
	actors      []grp.ID
	authmgr     *AuthManager
	bcast       chan<- interface{}
	ucast       chan<- net.Packet
	dcdChan     chan<- *px.Value
	forwardChan <-chan Forward
	valRespChan <-chan ValResp
	started     bool
	stop        chan bool
	stopCheckIn *sync.WaitGroup
}

// Construct a new AuthLearner
func NewAuthLearner(pp *px.Pack) *AuthLearner {
	return &AuthLearner{
		id:          pp.ID,
		dmx:         pp.Dmx,
		nAcceptors:  pp.NrOfAcceptors,
		nFaulty:     (pp.NrOfAcceptors - 1) / 3,
		rounds:      NewRoundStorage((uint(pp.Config.GetInt("alpha", config.DefAlpha)) + 10) * 3),
		maxlearned:  0,
		alpha:       uint(pp.Config.GetInt("alpha", config.DefAlpha)),
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

func (l *AuthLearner) Start() {
	if l.started {
		glog.V(2).Info("ignoring start request")
		return
	}
	l.registerChannels()
	glog.V(1).Info("starting")
	l.started = true
	go func() {
		defer l.stopCheckIn.Done()
		for {
			select {
			//Receive forward
			case cmsg := <-l.forwardChan:
				l.handleForward(&cmsg)
			//Receive value from acceptor
			case valresp := <-l.valRespChan:
				l.storeVal(valresp)
				l.sendDecided(valresp.Rnd)
			//Receive stop signal
			case <-l.stop:
				glog.V(2).Info("exiting")
				l.started = false
				return
			}
		}
	}()
}

//Register channels with the demuxer
func (l *AuthLearner) registerChannels() {
	forwardChan := make(chan Forward, 64)
	l.forwardChan = forwardChan
	l.dmx.RegisterChannel(forwardChan)
	valRespChan := make(chan ValResp, 8)
	l.valRespChan = valRespChan
	l.dmx.RegisterChannel(valRespChan)
}

//Stops the learner
func (l *AuthLearner) Stop() {
	l.stop <- true
}

// ****************************************************************
// Handling forwards
// ****************************************************************

//Function that handles incoming forward messages
//returns true if the message caused a learn
func (l *AuthLearner) handleForward(msg *Forward) bool {
	glog.V(2).Infof("Received forward for round %d from id %d\n", msg.Rnd, msg.ID.PaxosID)
	//If the forward has wrong mac, drop
	if !l.authmgr.checkMAC(msg.ID, msg.Hmac, l.authmgr.hashToBytes(msg.Hash), msg.Rnd) {
		glog.Warningln("Received wrong HMAC from ", msg.ID)
		return false
	}
	//If this is a forward for an old round, return
	if !l.rounds.IsValid(msg.Rnd) {
		return false
	}

	rnd := l.rounds.Get(msg.Rnd)
	// If no forwards received for round, then initialize it
	if len(rnd.Forwards) == 0 {
		rnd.RndID = msg.Rnd
		rnd.Forwards = make(map[grp.ID]Forward)
		rnd.Decided = false
		rnd.SntReq = false
		rnd.Learned = false
	}
	//If value already decided, return
	if rnd.Decided {
		return false
	}

	//If already received forward from that Id, return
	if _, ok := rnd.Forwards[msg.ID]; ok {
		glog.V(2).Infoln("Received duplicate forward from ", msg.ID)
		return false
	}

	//Store the forward and check if enough forwards
	rnd.Forwards[msg.ID] = *msg
	rnd.Decided = l.compareForwards(rnd.Forwards)
	//If the value is decided on, then ask for value
	if rnd.Decided && !rnd.SntReq {
		l.sendValReq(msg.Rnd)
		rnd.SntReq = true
	}
	return true
}

//Checks if there is enough forwards for a round. Returns true if there are
//enough forwards of the same value. This function is only called after a
//new forward is added, so it doesn't need to return which forward is correct
func (l *AuthLearner) compareForwards(forwards map[grp.ID]Forward) bool {
	forwardcount := make(map[uint32]uint)
	for _, forward := range forwards {
		forwardcount[forward.Hash]++
	}
	for _, v := range forwardcount {
		if v > (2 * l.nFaulty) {
			return true
		}
	}
	return false
}

//Sends a request to the acceptor for the needed value
func (l *AuthLearner) sendValReq(rnd uint) {
	l.send(l.id, ValReq{rnd})
}

//Stores a value for the appropriate round
func (l *AuthLearner) storeVal(resp ValResp) {
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
func (l *AuthLearner) sendDecided(rnd uint) {
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
func (l *AuthLearner) broadcast(msg interface{}) {
	l.bcast <- msg
}

func (l *AuthLearner) send(id grp.ID, msg interface{}) {
	l.ucast <- net.Packet{DestID: id, Data: msg}
}
