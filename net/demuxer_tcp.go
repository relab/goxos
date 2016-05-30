package net

import (
	"errors"
	"net"
	"reflect"
	"sync"

	"github.com/relab/goxos/grp"

	"github.com/relab/goxos/Godeps/_workspace/src/github.com/golang/glog"
)

var (
	ErrIDIsEqual     = errors.New("id is equal to this node")
	ErrIDOutOfBounds = errors.New("id is out of bounds for current cluster size")
)

type msgtype reflect.Type

// The Demuxer handles all of the incoming messages to a replica -- not including
// the client communication.
type TcpDemuxer struct {
	id          grp.ID
	grpmgr      grp.GroupManager
	listener    net.Listener
	closed      bool
	channels    map[msgtype][]reflect.Value
	fdChan      chan interface{}
	stopCheckIn *sync.WaitGroup
}

// NewDemuxer creates a new Demuxer for a replica. A valid id from the configuration must
// be passed in.
func NewTcpDemuxer(id grp.ID, gm grp.GroupManager, stopCheckIn *sync.WaitGroup) *TcpDemuxer {
	me, found := gm.NodeMap().LookupNode(id)
	if !found {
		glog.Fatal("couldn't find running node in configuration")
	}

	listener, err := net.Listen("tcp", me.PaxosAddr())
	if err != nil {
		glog.Fatalf("couldn't listen on %v (%v)", me.PaxosAddr(), err)
	}

	return &TcpDemuxer{
		id:          id,
		grpmgr:      gm,
		listener:    listener,
		channels:    make(map[msgtype][]reflect.Value),
		stopCheckIn: stopCheckIn,
	}
}

// Start handling new connections from other replicas.
func (dmx *TcpDemuxer) Start() {
	glog.V(1).Info("starting")
	go func() {
		for !dmx.closed {
			conn, err := dmx.listener.Accept()
			if err != nil {
				if IsSocketClosed(err) {
					glog.Warningln("replica listener closed; exiting")
					break
				}
				glog.Error(err)
				continue
			}

			c := NewConnection(conn)
			glog.V(2).Infoln("received connection from", c)

			cid, err := c.waitForID()
			if err != nil {
				glog.Errorln("error receiving id,", err)
				c.Close()
				continue
			}

			err = dmx.validateID(cid)
			if err != nil {
				glog.Errorln("id rejected,", err)
				c.sendIDResp(false, err.Error())
				c.Close()
				continue
			} else {
				err = c.sendIDResp(true, "")
				if err != nil {
					glog.Errorln("error sending id resp,", err)
					c.Close()
					continue
				}
			}

			gc := NewGxConnection(c, cid, dmx)
			if err = AddToConnections(gc, dmx.grpmgr.LrEnabled() ||
				dmx.grpmgr.ArEnabled()); err != nil {
				glog.Error(err)
				c.Close()
				continue
			}
		}
		glog.V(1).Info("exiting")
	}()
}

// Shut the Demuxer down. Stops the main Demuxer goroutine.
func (dmx *TcpDemuxer) Stop() {
	dmx.listener.Close()
	dmx.closed = true
	dmx.stopCheckIn.Done()
}

// Register channel for receiving messages of the type defined by the channel
func (dmx *TcpDemuxer) RegisterChannel(ch interface{}) {
	chVal := reflect.ValueOf(ch)

	// TODO: This should return an error instead?
	if chVal.Kind() != reflect.Chan {
		glog.Fatal("argument 'ch' to RegisterChannel must be a channel")
	}

	// Extract the message type supported by the provided channel
	chType := chVal.Type().Elem()
	if _, present := dmx.channels[chType]; !present {
		//log.Panicf("demuxer: multiple receivers registered for message type: %s", chType)
		dmx.channels[chType] = make([]reflect.Value, 0)
	}

	// The message/channel type is used as map key for channels
	dmx.channels[chType] = append(dmx.channels[chType], chVal)
	glog.V(2).Infof("registered channel for: %s", chType)
}

// Delegate handling of specific messages to a priori registered channels
func (dmx *TcpDemuxer) HandleMessage(msg interface{}) {
	// Inspect type of msg
	mx := reflect.ValueOf(msg)
	msgType := mx.Type()
	if glog.V(4) {
		glog.Infof("received message of type: %s", msgType)
	}

	// Lookup channel on which to send the given message type
	if chVals, ok := dmx.channels[msgType]; ok {
		for _, ch := range chVals {
			ch.Send(mx)
		}
	}
}

func (dmx *TcpDemuxer) validateID(cid grp.ID) error {
	if cid == dmx.id && !dmx.grpmgr.LrEnabled() {
		return ErrIDIsEqual
	}

	if cid.PaxosID <= grp.MinPaxosID || cid.PaxosID > grp.PaxosID(dmx.grpmgr.NrOfNodes()-1) {
		return ErrIDOutOfBounds
	}

	return nil
}
