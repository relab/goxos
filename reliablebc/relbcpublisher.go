package reliablebc

import (
	"container/list"
	"crypto/rand"
	"fmt"
	"sync"
	"time"

	"github.com/relab/goxos/client"
	"github.com/relab/goxos/config"
	"github.com/relab/goxos/grp"
	"github.com/relab/goxos/net"
	px "github.com/relab/goxos/paxos"

	"github.com/relab/goxos/Godeps/_workspace/src/github.com/golang/glog"
)

var (
	seq            uint32
	reqID          = "GenericReq"
	reqValue       string
	reqType        = client.Request_EXEC
	failedReplicas uint
)

type RelPublisher struct {
	id             grp.ID
	config         *config.Config
	alpha          uint
	dmx            net.Demuxer
	currentRnd     uint
	maxdecided     uint
	prevmaxdecided uint
	decided        map[uint]uint
	actors         []grp.ID
	authmgr        *AuthManager
	bcast          chan<- interface{}
	ucast          chan<- net.Packet
	nAcceptors     uint
	publishqueue   *list.List
	dcdChan        <-chan DcdRnd
	started        bool
	stop           chan bool
	stopCheckIn    *sync.WaitGroup
}

// Construct a new RelPublisher
func NewRelPublisher(pp *px.Pack) *RelPublisher {
	return &RelPublisher{
		id:           pp.ID,
		config:       pp.Config,
		alpha:        uint(pp.Config.GetInt("alpha", config.DefAlpha)),
		dmx:          pp.Dmx,
		authmgr:      NewAuthManager(pp.ID, pp.Config),
		actors:       pp.Gm.NodeMap().AcceptorIDs(),
		ucast:        pp.Ucast,
		bcast:        pp.Bcast,
		nAcceptors:   pp.NrOfAcceptors,
		publishqueue: list.New(),
		decided:      make(map[uint]uint),
		started:      false,
		stop:         make(chan bool),
		stopCheckIn:  pp.StopCheckIn,
	}
}

//Register channels with the demuxer
func (p *RelPublisher) registerChannels() {
	dcdChan := make(chan DcdRnd, 64)
	p.dcdChan = dcdChan
	p.dmx.RegisterChannel(dcdChan)
}

//Function for generating publications. We use clienth.Request for this
func (p *RelPublisher) generateRequests(rounds uint, payload uint) {
	var rnd uint
	for rnd = 0; rnd < rounds; rnd++ {
		//Generating a random request
		b := make([]byte, payload)
		rand.Read(b)
		req := client.Request{
			Type: &reqType,
			Id:   &reqID,
			Seq:  &seq,
			Val:  b,
		}
		p.publishqueue.PushBack(&px.Value{
			Vt: 1,
			Cr: []*client.Request{&req},
		})
	}
}

func (p *RelPublisher) Start() {
	if p.started {
		glog.V(2).Info("ignoring start request")
		return
	}
	p.registerChannels()
	glog.V(1).Info("starting publisher")
	defer p.stopCheckIn.Done()

	if p.id.PaxosID == 0 {
		p.started = true
		go func() {
			//Decide number of requests and payload here
			p.generateRequests(
				uint(p.config.GetInt(
					"nrOfPublications",
					config.DefNrOfPublications,
				)),
				uint(p.config.GetInt(
					"publicationPayloadSize",
					config.DefPublicationPayloadSize,
				)),
			)
			//Short sleep to make sure everyone is connected
			time.Sleep(time.Second * 3)
			//Ticker to ensure liveness. Should be set someway else
			ticker := time.NewTicker(time.Millisecond * 500)
			p.genPublish()
			for {
				select {
				case drnd := <-p.dcdChan:
					if drnd.Rnd > p.maxdecided {
						p.decided[drnd.Rnd]++
						if p.decided[drnd.Rnd] >= p.nAcceptors-failedReplicas {
							p.maxdecided = drnd.Rnd
						}
						p.genPublish()
					}
					//To ensure liveness.
				case <-ticker.C:
					//If a round has not been decided in a set interval and
					//there are values in queue, then force next round.
					if p.maxdecided == p.prevmaxdecided && p.publishqueue.Len() != 0 {
						p.maxdecided++
						failedReplicas++
						p.genPublish()
					}
					p.prevmaxdecided = p.maxdecided
				case <-p.stop:
					glog.V(2).Info("exiting")
					p.started = false
					return
				}
			}
		}()
	}
}

//Stops the publisher
func (p *RelPublisher) Stop() {
	if p.started {
		p.stop <- true
	}
}

// Sends publish messages limited by maxdecided and alpha
func (p *RelPublisher) genPublish() {
	current := p.currentRnd
	to := p.maxdecided + p.alpha
	for ; current < to; current++ {
		if p.publishqueue.Len() != 0 {
			val := p.publishqueue.Front().Value.(*px.Value)
			p.publishqueue.Remove(p.publishqueue.Front())
			p.sendPublish(&Publish{
				Rnd: current,
				Val: *val,
			})
		} else {
			break
		}
	}
	if p.publishqueue.Len() == 0 {
		glog.V(2).Info("finished!")
		fmt.Println("finished!")
	}
	p.currentRnd = current
}

func (p *RelPublisher) sendPublish(msg *Publish) {
	for _, id := range p.actors {
		p.send(id, Publish{
			ID:  p.id,
			Rnd: msg.Rnd,
			Val: msg.Val,
			Hmac: p.authmgr.generateMAC(
				id,
				p.authmgr.hashToBytes(msg.Val.Hash()),
				msg.Rnd,
			),
		})
	}
}

// ---------------------------------------------------------
// Network functions
func (p *RelPublisher) broadcast(msg interface{}) {
	p.bcast <- msg
}

func (p *RelPublisher) send(id grp.ID, msg interface{}) {
	p.ucast <- net.Packet{DestID: id, Data: msg}
}

// ****************************************************************
// Not used by RelBC, but needed to fulfill the actor interface
// ****************************************************************
func (p *RelPublisher) SetNextSlot(ns px.SlotID) {
}
