package net

import (
	"errors"
	"sync"
	"time"

	"github.com/relab/goxos/grp"
	"github.com/relab/goxos/liveness"

	"github.com/relab/goxos/Godeps/_workspace/src/github.com/golang/glog"
)

const (
	fullyConnectedWait = 250 * time.Millisecond
)

var (
	errNodeNotFound = errors.New("can't connect, node not found in nodemap")
	errConnNotFound = errors.New("connection not found")
)

// A Sender is responsible for outward communication from one replica to all others. Goroutines
// in a replica communicate with the Sender through channels, which are then sent out onto the
// network.
type Sender struct {
	id            grp.ID
	connected     bool
	grpmgr        grp.GroupManager
	grpSubscriber grp.Subscriber
	resetch       chan bool
	stop          chan bool
	outB          <-chan interface{} // Broadcast to all nodes
	outP          <-chan interface{} // Broadcast channel for proposers
	outA          <-chan interface{} // Broadcast channel for acceptors
	outL          <-chan interface{} // Broadcast channel for learners
	outU          <-chan Packet      // Unicast channel
	dmx           Demuxer
	stopCheckIn   *sync.WaitGroup
}

// Create a new Sender for the given replica id. Also passed in are channels which the sender receives
// messages from.
func NewSender(id grp.ID, gm grp.GroupManager, outU <-chan Packet,
	outB, outP, outA, outL <-chan interface{},
	dmx Demuxer, stopCheckIn *sync.WaitGroup) (snd *Sender) {
	return &Sender{
		id:          id,
		connected:   false,
		grpmgr:      gm,
		resetch:     make(chan bool, 64),
		stop:        make(chan bool),
		outB:        outB,
		outP:        outP,
		outA:        outA,
		outL:        outL,
		outU:        outU,
		dmx:         dmx,
		stopCheckIn: stopCheckIn,
	}
}

// Start the initial connection phase, where the sender tries to connect to all other replicas in the system
// with a higher id than ours. We wait for connections from all other replicas with ids less than or equal to
// our own.
//
// This function blocks until we establish connections to all replicas in the Goxos configuration.
func (snd *Sender) InitialConnect() {
	glog.V(1).Info("starting initial-connect procedure")
	for _, id := range snd.grpmgr.NodeMap().IDs() {
		if id.PaxosID > snd.id.PaxosID {
			node, found := snd.grpmgr.NodeMap().LookupNode(id)
			if !found {
				glog.Fatalln("initial-connect failed", errNodeNotFound)
			}

			conn, err := GxConnectTo(node, snd.id, id, snd.dmx)
			if err != nil {
				glog.Fatalln("initial-connect failed:", err)
			}

			connections[id.PaxosID] = conn
			go conn.handleIn()
			go conn.handleOut()
		}
	}

	glog.V(2).Infoln("connected to the nodes that we should initiate connections to",
		"checking if we are fully connected")
	for !snd.areWeFullyConnected() {
		glog.V(2).Infoln("we are not fully connected, waiting", fullyConnectedWait)
		time.Sleep(fullyConnectedWait)
	}
	glog.V(1).Info("connected to all nodes from inital configuration")
	snd.connected = true
}

// Start receiving messages from other Goxos modules.
func (snd *Sender) Start() {
	glog.V(1).Info("starting")
	snd.grpSubscriber = snd.grpmgr.SubscribeToHold("sender")

	go func() {
		defer snd.stopCheckIn.Done()

		for {
			select {
			case msg := <-snd.outB:
				snd.broadcastAll(msg)
			case msg := <-snd.outP:
				snd.broadcastProposers(msg)
			case msg := <-snd.outA:
				snd.broadcastAcceptors(msg)
			case msg := <-snd.outL:
				snd.broadcastLearners(msg)
			case packet := <-snd.outU:
				snd.unicast(packet.Data, packet.DestID)
			case grpPrepare := <-snd.grpSubscriber.PrepareChan():
				snd.handleGrpHold(grpPrepare)
			case <-snd.stop:
				glog.V(1).Info("exiting")
				return
			}
		}
	}()
}

// Stop the Sender goroutine.
func (snd *Sender) Stop() {
	snd.stop <- true
}

func (snd *Sender) ResetChan() <-chan bool {
	return snd.resetch
}

func (snd *Sender) unicast(msg interface{}, id grp.ID) {
	if id == snd.id {
		switch msg.(type) {
		case liveness.Heartbeat:
			return
		default:
			snd.dmx.HandleMessage(msg)
			return
		}
	}

	c, err := snd.getConnection(id.PaxosID)
	if err != nil {
		glog.Error(err)
		return
	}

	select {
	case c.Outgoing() <- msg:
		// Send was non-blocking
	default:
		if glog.V(4) {
			glog.Infof("Send on %v was blocking, dropping message", c)
		}
	}
}

func (snd *Sender) broadcast(msg interface{}, ids []grp.ID) {
	for _, id := range ids {
		snd.unicast(msg, id)
	}
	switch msg.(type) {
	case liveness.Heartbeat:
		return
	default:
		snd.resetch <- true // reset heartbeat emitter
	}
}

func (snd *Sender) broadcastAll(msg interface{}) {
	snd.broadcast(msg, snd.grpmgr.NodeMap().IDs())
}

func (snd *Sender) broadcastProposers(msg interface{}) {
	snd.broadcast(msg, snd.grpmgr.NodeMap().ProposerIDs())
}

func (snd *Sender) broadcastAcceptors(msg interface{}) {
	snd.broadcast(msg, snd.grpmgr.NodeMap().AcceptorIDs())
}

func (snd *Sender) broadcastLearners(msg interface{}) {
	snd.broadcast(msg, snd.grpmgr.NodeMap().LearnerIDs())
}

func (snd *Sender) getConnection(pid grp.PaxosID) (*GxConnection, error) {
	c, found := connections[pid]
	if !found {
		return nil, errConnNotFound
	}
	return c, nil
}

func (snd *Sender) areWeFullyConnected() bool {
	for _, id := range snd.grpmgr.NodeMap().IDs() {
		if _, connFound := connections[id.PaxosID]; !connFound {
			if id == snd.id {
				continue
			}
			return false
		}
	}

	return true
}

func (snd *Sender) handleGrpHold(gp *sync.WaitGroup) {
	glog.V(2).Info("grpmgr hold req")
	gp.Done()
	<-snd.grpSubscriber.ReleaseChan()
	glog.V(2).Info("grpmgr release")
	snd.id = snd.grpmgr.GetID()
}
