package client

import (
	"errors"
	"io"
	"net"
	"time"

	"github.com/relab/goxos/Godeps/_workspace/src/github.com/golang/glog"
)

// A ClientConn is the server-side represention a connection between a replica
// and a client. Connected is set to true/false depending on the status of the
// connection.
type ClientConn struct {
	conn      net.Conn
	connected bool
	addr      string
	reqChan   chan<- *Request
	respChan  chan *Response // Used by WriteAsync to queue responses for writing
}

// Create a new ClientConn. A low-level socket structure must be passed in along with a
// string-based id and the channel where requests are sent.
func NewClientConn(c net.Conn, reqch chan<- *Request) *ClientConn {
	return &ClientConn{
		conn:      c,
		connected: true,
		addr:      c.RemoteAddr().String(),
		reqChan:   reqch,
		respChan:  make(chan *Response, 512),
	}
}

// Start serving this ClientConn. Reads a Request from the connection, which
// gets sent to Goxos for agreement.
func (cc *ClientConn) Serve() {
	glog.V(1).Infoln("starting to serve client", cc)
	var err error
	defer func() {
		if err != io.EOF {
			glog.Errorf("client %v: %v", cc, err)
		}
		cc.Close()
	}()

	go cc.handleRespChan()

	for {
		var req Request
		if err = read(cc.conn, &req); err != nil {
			return
		}
		if req.IsIDInvalid() {
			err = errors.New("invalid id: " + req.GetId())
			return
		}
		cc.reqChan <- &req
	}
}

func (cc *ClientConn) handleRespChan() {
	for {
		select {
		case resp := <-cc.respChan:
			cc.conn.SetWriteDeadline(time.Now().Add(100 * time.Millisecond))
			if err := write(cc.conn, resp); err != nil {
				glog.Errorln("error writing response to client:", err)
				if err == io.EOF {
					cc.Close()
					// TODO: Should we also return here?
				}
			}
		}
	}
}

// Write a Response to the ClientConn. This method gets called when a Request
// has been executed.
func (cc *ClientConn) WriteAsync(resp *Response) {
	cc.respChan <- resp
}

func (cc *ClientConn) Close() {
	cc.connected = false
	cc.conn.Close()
	//TODO Should we also close the respChan ???
	// close(cc.respChan)
}

func (cc *ClientConn) String() string {
	return cc.addr
}
