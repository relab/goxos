package net

import (
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/relab/goxos/grp"

	"github.com/relab/goxos/Godeps/_workspace/src/github.com/golang/glog"
)

const (
	reconnectWait   = 500 * time.Millisecond
	maxConnAttempts = 120
)

var (
	connections   = make(map[grp.PaxosID]*GxConnection)
	heartbeatChan chan<- grp.ID
)

func SetHeartbeatChan(hbChan chan<- grp.ID) {
	heartbeatChan = hbChan
}

// Add a GxConnection to the connection map.
func AddToConnections(gc *GxConnection, lrArEnabled bool) error {
	existingConn, found := connections[gc.id.PaxosID]
	if !lrArEnabled {
		connections[gc.id.PaxosID] = gc
		go gc.handleIn()
		go gc.handleOut()
		if found {
			existingConn.Close()
		}
		return nil
	}

	// LR/Arec enabled
	if found {
		comparison := gc.id.CompareTo(existingConn.id)
		if comparison < 0 {
			return errors.New("id for connection is lower than already present")
		}
		existingConn.Close()
	}

	connections[gc.id.PaxosID] = gc
	go gc.handleIn()
	go gc.handleOut()

	return nil
}

// Connect to another replica, and verify the ids are correct. Returns a GxConnection.
func GxConnectTo(node grp.Node, callerID, calledID grp.ID,
	dmx Demuxer) (*GxConnection, error) {
	conn, err := ConnectToNode(node)
	if err != nil {
		return nil, err
	}

	if err = conn.sendID(callerID); err != nil {
		return nil, err
	}

	idresp, err := conn.waitForIDResp()
	if err != nil {
		return nil, err
	}

	if !idresp.Accepted {
		errs := fmt.Sprintf("id rejected: %v", idresp.Error)
		return nil, errors.New(errs)
	}

	return NewGxConnection(conn, calledID, dmx), nil
}

// Connect to another replica based on the address in the configuration file.
func ConnectToNode(node grp.Node) (*Connection, error) {
	glog.V(2).Infoln("attempting to connect to", node)
	return ConnectToAddr(node.PaxosAddr())
}

// Connect to another replica based on the address of the replica in the form
// hostname:port. Returns a Connection.
func ConnectToAddr(addr string) (*Connection, error) {
	glog.V(2).Infoln("attempting to connect to", addr)
	conn, err := net.Dial("tcp", addr)
	for i := 0; err != nil; i++ {
		if err == nil {
			break
		}
		if i >= maxConnAttempts {
			errmsg := fmt.Sprintf("network: unable to connect to addr %v"+
				"(tried %v times)", addr, maxConnAttempts)
			return nil, errors.New(errmsg)
		}
		glog.Warningln("error on connecting to addr", addr, ":", err,
			"waiting", reconnectWait, "before trying again")
		time.Sleep(reconnectWait)
		conn, err = net.Dial("tcp", addr)
	}

	return NewConnection(conn), nil
}

func GxConnectEphemeral(node grp.Node, callerID grp.ID) (*Connection, error) {
	c, err := net.DialTimeout("tcp", node.PaxosAddr(), 500*time.Millisecond)
	if err != nil {
		return nil, err
	}

	conn := NewConnection(c)

	if err = conn.sendID(callerID); err != nil {
		return nil, err
	}

	idresp, err := conn.waitForIDResp()
	if err != nil {
		return nil, err
	}

	if !idresp.Accepted {
		errs := fmt.Sprintf("id rejected: %v", idresp.Error)
		return nil, errors.New(errs)
	}

	return conn, nil
}

func CheckConnections(conf []grp.ID) (notconn []grp.ID) {
	notconn = make([]grp.ID, 0)
	for _, id := range conf {
		if gc, ok := connections[id.PaxosID]; ok {
			if id != gc.id {
				notconn = append(notconn, id)
			}
		} else {
			notconn = append(notconn, id)
		}
	}
	return notconn
}

func UpdateConnID(oldID grp.ID, newEpoch grp.Epoch) bool {
	if gc, ok := connections[oldID.PaxosID]; ok {
		if gc.id == oldID {
			gc.id.Epoch = newEpoch
			return true
		}
	}
	return false
}
