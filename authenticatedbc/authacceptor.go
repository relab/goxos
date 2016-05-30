package authenticatedbc

import (
	"sync"

	"github.com/relab/goxos/config"
	"github.com/relab/goxos/grp"
	"github.com/relab/goxos/net"
	px "github.com/relab/goxos/paxos"

	"github.com/relab/goxos/Godeps/_workspace/src/github.com/golang/glog"
)

type AuthAcceptor struct {
	id          grp.ID
	dmx         net.Demuxer
	rounds      *RoundStorage
	lowSlot     px.SlotID
	actors      []grp.ID
	authmgr     *AuthManager
	bcast       chan<- interface{}
	ucast       chan<- net.Packet
	publishChan <-chan Publish
	valReqChan  <-chan ValReq
	stop        chan bool
	stopCheckIn *sync.WaitGroup
}

// Construct a new AuthAcceptor.
func NewAuthAcceptor(pp *px.Pack) *AuthAcceptor {
	return &AuthAcceptor{
		id:  pp.ID,
		dmx: pp.Dmx,
		//The delta will later decide the size of the roundstorage
		rounds:      NewRoundStorage((uint(pp.Config.GetInt("alpha", config.DefAlpha)) + 10) * 3),
		ucast:       pp.Ucast,
		bcast:       pp.Bcast,
		actors:      pp.Gm.NodeMap().AcceptorIDs(),
		authmgr:     NewAuthManager(pp.ID, pp.Config),
		stop:        make(chan bool),
		stopCheckIn: pp.StopCheckIn,
	}
}

func (aa *AuthAcceptor) registerChannels() {
	publishChan := make(chan Publish, 64)
	aa.publishChan = publishChan
	aa.dmx.RegisterChannel(publishChan)
	valReqChan := make(chan ValReq, 8)
	aa.valReqChan = valReqChan
	aa.dmx.RegisterChannel(valReqChan)
}

func (aa *AuthAcceptor) Start() {
	glog.V(2).Info("starting")
	aa.registerChannels()

	go func() {
		defer aa.stopCheckIn.Done()
		for {
			select {
			case publish := <-aa.publishChan:
				//fmt.Printf("Received publish from rnd: %d\n", publish.Slot)
				forward := aa.handlePublish(publish)
				if forward != nil {
					glog.V(2).Info("sending forward")
					aa.sendForward(forward)
				}
			case reqRnd := <-aa.valReqChan:
				aa.sendVal(reqRnd.Rnd)
			case <-aa.stop:
				glog.V(2).Info("exiting")
				return
			}
		}
	}()
}

func (aa *AuthAcceptor) Stop() {
	aa.stop <- true
}

func (aa *AuthAcceptor) handlePublish(msg Publish) *Forward {
	glog.V(2).Infoln("Recieved publish from", msg.ID)
	if !aa.authmgr.checkMAC(
		msg.ID,
		msg.Hmac,
		aa.authmgr.hashToBytes(msg.Val.Hash()),
		msg.Rnd,
	) {
		glog.Warningln("Received wrong HMAC from ", msg.ID)
		return nil
	}
	round := aa.rounds.Get(msg.Rnd)

	if !aa.rounds.IsValid(msg.Rnd) {
		return nil
	}

	//Store round
	round.RndID = msg.Rnd
	round.Val = msg.Val

	return aa.createForward(msg)
}

func (aa *AuthAcceptor) createForward(msg Publish) *Forward {

	//Create value hash
	hash := msg.Val.Hash()

	return &Forward{
		ID:   aa.id,
		Hash: hash,
		Rnd:  msg.Rnd,
	}
}

//Sends the forward messages
func (aa *AuthAcceptor) sendForward(msg *Forward) {
	for _, id := range aa.actors {
		aa.send(id, Forward{
			ID:   aa.id,
			Rnd:  msg.Rnd,
			Hash: msg.Hash,
			Hmac: aa.authmgr.generateMAC(id, aa.authmgr.hashToBytes(msg.Hash), msg.Rnd),
		})
	}
	//a.broadcast(msg)
}

func (aa *AuthAcceptor) getValue(rnd uint) px.Value {
	//Check if round is stored
	if &aa.rounds.Get(rnd).Val == nil {
		glog.Errorln("Value not stored in acceptor")
		return px.Value{}
	}

	return aa.rounds.Get(rnd).Val
}

func (aa *AuthAcceptor) sendVal(rnd uint) {
	val := aa.getValue(rnd)
	aa.ucast <- net.Packet{DestID: aa.id, Data: ValResp{rnd, val}}
}

// ****************************************************************
// Network functions
// ****************************************************************
func (aa *AuthAcceptor) broadcast(msg interface{}) {
	aa.bcast <- msg
}

func (aa *AuthAcceptor) send(id grp.ID, msg interface{}) {
	aa.ucast <- net.Packet{DestID: id, Data: msg}
}

// ****************************************************************
// Not used by Authenticated BC, but needed to fulfill the actor interface
// ****************************************************************

func (aa *AuthAcceptor) GetState(afterSlot px.SlotID, release <-chan bool) *px.AcceptorSlotMap {
	return nil
}

func (aa *AuthAcceptor) GetMaxSlot(release <-chan bool) *px.AcceptorSlotMap {
	return nil
}

func (aa *AuthAcceptor) SetState(slotmap *px.AcceptorSlotMap) {
}

func (aa *AuthAcceptor) SetLowSlot(slot px.SlotID) {
}
