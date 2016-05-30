package net

import (
	"encoding/gob"
	"fmt"
	"io"
	"net"

	"github.com/relab/goxos/grp"

	"github.com/relab/goxos/Godeps/_workspace/src/github.com/golang/glog"
)

// A Connection represents a base connection between two replicas in Goxos.
type Connection struct {
	conn io.ReadWriteCloser
	Dec  *gob.Decoder
	Enc  *gob.Encoder
	addr string
}

// Creates a new base connection between two replicas. The required argument
// is a low-level connection to another replica.
func NewConnection(conn net.Conn) *Connection {
	return &Connection{
		conn: conn,
		Dec:  gob.NewDecoder(conn),
		Enc:  gob.NewEncoder(conn),
		addr: conn.RemoteAddr().String(),
	}
}

// Read a message off of the connection. The message is placed in the location
// passed to the method. Returns nil or an error.
func (c *Connection) Read(msg interface{}) error {
	if err := c.Dec.Decode(&msg); err != nil {
		return err
	}

	return nil
}

// Write a message to the connection. Returns nil or an error.
func (c *Connection) Write(msg interface{}) error {
	if err := c.Enc.Encode(&msg); err != nil {
		return err
	}

	return nil
}

// Close the connection.
func (c *Connection) Close() error {
	return c.conn.Close()
}

func (c *Connection) sendID(id grp.ID) error {
	idexch := IDExchange{id}
	if err := c.Enc.Encode(&idexch); err != nil {
		return err
	}

	return nil
}

func (c *Connection) waitForID() (grp.ID, error) {
	var idexch IDExchange
	if err := c.Dec.Decode(&idexch); err != nil {
		return grp.ID{}, err
	}

	return idexch.ID, nil
}

func (c *Connection) waitForIDResp() (IDResponse, error) {
	var idresp IDResponse
	if err := c.Dec.Decode(&idresp); err != nil {
		return idresp, err
	}

	return idresp, nil
}

func (c *Connection) sendIDResp(ok bool, err string) error {
	idresp := IDResponse{ok, err}
	if err := c.Enc.Encode(&idresp); err != nil {
		return err
	}

	return nil
}

// Returns a string-based representation of the Connection.
func (c Connection) String() string {
	return fmt.Sprintf("connection from %v", c.addr)
}

// A GxConnection represents a connection between two replicas. A GxConnection is setup
// after the replica has been properly validated.
type GxConnection struct {
	*Connection
	id            grp.ID
	dmx           Demuxer
	outgoing      chan interface{}
	heartbeatChan chan<- grp.ID
}

// Create a new GxConnection. The low-level connection as well as the Goxos id and Demuxer
// must be passed in as arguments.
func NewGxConnection(conn *Connection, id grp.ID, dmx Demuxer) *GxConnection {
	return &GxConnection{
		Connection:    conn,
		id:            id,
		dmx:           dmx,
		outgoing:      make(chan interface{}, 128),
		heartbeatChan: heartbeatChan,
	}
}

func (gc *GxConnection) handleIn() {
	glog.V(2).Infof("%v: starting to handle incomming", gc)
	var err error
	var msg interface{}
	defer gc.Close()
	for {
		if err = gc.Dec.Decode(&msg); err == nil {
			gc.dmx.HandleMessage(msg)
			gc.heartbeatChan <- gc.id
		}
		if err == io.EOF {
			break
		} else if err != nil {
			glog.Errorf("%v: %v", gc, err)
			break
		}
	}
}

func (gc *GxConnection) handleOut() {
	glog.V(2).Infof("%v: starting to handle outgoing", gc)
	var err error
	var msg interface{}
	for {
		select {
		case msg = <-gc.outgoing:
			if err = gc.Write(msg); err == nil {
				continue
			}
			if err == io.EOF {
				glog.V(2).Infof("%v: connection closed")
				return
			}
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				glog.V(2).Infof("%v: tmp error: %v", ne)
				continue
			}
			glog.Errorf("%v: closing due to: %v", gc, err)
			return
		}
	}
}

func (gc *GxConnection) Outgoing() chan<- interface{} {
	return gc.outgoing
}

// Returns a string-based representation of the GxConnection.
func (gc GxConnection) String() string {
	return fmt.Sprintf("connection %v (%v)", gc.id, gc.addr)
}

// Create a new mock Connection, used for testing purposes.
func NewMockConnection(conn io.ReadWriteCloser) *Connection {
	return &Connection{
		conn: conn,
		Dec:  gob.NewDecoder(conn),
		Enc:  gob.NewEncoder(conn),
		addr: "unknown",
	}
}
