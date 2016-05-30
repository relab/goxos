package client

import (
	"net"
	"strings"
	"sync"

	"github.com/relab/goxos/grp"
	"github.com/relab/goxos/liveness"
	gnet "github.com/relab/goxos/net"

	"github.com/relab/goxos/Godeps/_workspace/src/github.com/golang/glog"
)

var allowDirect = map[string]bool{
	"fastpaxos":  true,
	"batchpaxos": true,
}

// The ClientHandler module maintains all of the connections that a replica has with
// its clients.
type ClientHandlerTCP struct {
	id            grp.ID
	leader        grp.ID
	paxosType     string
	grpmgr        grp.GroupManager
	grpSubscriber grp.Subscriber
	listener      net.Listener
	trust         <-chan grp.ID
	reqChan       chan *Request
	propChan      chan<- *Request
	respChan      chan *Response
	clients       map[string]*ClientConn
	replies       map[string]*Response
	stop          chan bool
	stopCheckIn   *sync.WaitGroup
}

// Create a new ClientHandler.
func NewClientHandlerTCP(id grp.ID, paxosType string, gm grp.GroupManager,
	ld liveness.LeaderDetector, propChan chan<- *Request,
	stopCheckIn *sync.WaitGroup) *ClientHandlerTCP {

	me, found := gm.NodeMap().LookupNode(id)
	if !found {
		glog.Fatal("could not find myself in configuration")
	}

	listener, err := net.Listen("tcp", me.ClientAddr())
	if err != nil {
		glog.Fatalf("could not listen on %v (%v)", me.ClientAddr(), err)
	}

	return &ClientHandlerTCP{
		id:          id,
		leader:      ld.PaxosLeader(),
		paxosType:   strings.TrimSpace(strings.ToLower(paxosType)),
		grpmgr:      gm,
		listener:    listener,
		trust:       ld.SubscribeToPaxosLdMsgs("clienthandler"),
		reqChan:     make(chan *Request, 64),
		propChan:    propChan,
		respChan:    make(chan *Response, 512),
		clients:     make(map[string]*ClientConn),
		replies:     make(map[string]*Response),
		stop:        make(chan bool),
		stopCheckIn: stopCheckIn,
	}
}

// Start handling new client connections. Spawns two goroutines, one for
// handshaking with new connections, and the other for handling requests and
// responses from other modules.
func (ch *ClientHandlerTCP) Start() {
	go func() {
		glog.V(1).Info("start listening for clients")
		for {
			conn, err := ch.listener.Accept()
			if err != nil {
				if gnet.IsSocketClosed(err) {
					glog.Warningln("client listener closed; exiting")
					//Closing all client connections
					for _, cc := range ch.clients {
						cc.Close()
					}
					break
				}
				glog.Errorln(err)
				continue
			}
			glog.V(2).Infoln("received connection from", conn.RemoteAddr())
			go ch.greetClient(conn)
		}
	}()

	go func() {
		if ch.grpmgr.ArEnabled() {
			ch.grpSubscriber = ch.grpmgr.SubscribeToHold("ch")
		}
		glog.V(1).Info("start handling requests and responses")
		defer ch.stopCheckIn.Done()

		for {
			select {
			case trustID := <-ch.trust:
				ch.leader = trustID
			case req := <-ch.reqChan:
				ch.handleRequest(req)
			case resp := <-ch.respChan:
				ch.handleResponse(resp)
			case grpPrepare := <-ch.grpSubscriber.PrepareChan():
				ch.handleGrpHold(grpPrepare)
			case <-ch.stop:
				ch.listener.Close()

				glog.V(1).Info("exiting")
				return
			}
		}
	}()
}

// Forward a response to the ClientHandler. The response will be routed back to the
// correct ClientConn which the request was received from.
func (ch *ClientHandlerTCP) ForwardResponse(resp *Response) {
	ch.respChan <- resp
}

// Shut down the ClientHandler module.
func (ch *ClientHandlerTCP) Stop() {
	ch.stop <- true
}

// Do the handshake with a new potential client connection.
func (ch *ClientHandlerTCP) greetClient(conn net.Conn) {
	glog.V(2).Info("greetClient: ", conn.RemoteAddr())

	var req Request
	err := read(conn, &req)
	if err != nil {
		glog.Warning("greetClient: read error, closing:", err)
		conn.Close()
		return
	}
	if req.GetType() != Request_HELLO {
		glog.Warning("greetClient: request was not handshake, closing")
		conn.Close()
		return
	}

	if ch.redirect() {
		// If I'm not the leader and don't allow direct messages, then
		// redirect the client to the leader.
		if leader, found := ch.grpmgr.NodeMap().LookupNode(ch.leader); found {
			glog.V(2).Info("greetClient: sending redirect and closing")
			resp := genErrResp(Response_HELLO_RESP, Response_REDIRECT, leader.ClientAddr())
			write(conn, resp)
			conn.Close()
			return
		}
	}

	if req.IsIDInvalid() {
		glog.Warning("greetClient: invalid id, closing")
		conn.Close()
		return
	}

	glog.V(2).Infof("greetClient: id %v accepted, replying and serving", req.GetId())
	resp := genResp(Response_HELLO_RESP, nil)
	resp.Protocol = ch.getPaxosType()
	err = write(conn, resp)
	if err != nil {
		glog.Warning("greetClient: write error, closing:", err)
		conn.Close()
		return
	}
	cc := NewClientConn(conn, ch.reqChan)
	ch.clients[req.GetId()] = cc
	go cc.Serve()
	return
}

func genResp(rt Response_Type, value []byte) *Response {
	return &Response{Type: &rt, Val: value}
}

func genErrResp(rt Response_Type, ec Response_Error, err string) *Response {
	return &Response{Type: &rt, ErrorCode: &ec, ErrorDetail: &err}
}

func (ch *ClientHandlerTCP) handleRequest(req *Request) {
	if glog.V(3) {
		glog.Infoln("received", req.SimpleString())
	}

	if ch.redirect() {
		// If I'm not the leader and don't allow direct messages, then
		// redirect the client to the leader.
		leader, found := ch.grpmgr.NodeMap().LookupNode(ch.leader)

		if !found {
			glog.Errorf("leader %v not in Nodemap", ch.leader)
			return
		}

		client, found := ch.clients[req.GetId()]

		if !found || !client.connected {
			glog.Errorf("client not connected?")
			return
		}

		resp := genErrResp(Response_EXEC_RESP, Response_REDIRECT, leader.ClientAddr())
		client.WriteAsync(resp)
		return
	}

	if req.GetType() != Request_EXEC {
		glog.Warning("received message from client was not a command, ignoring")
		return
	}

	// Deliver message
	lastReply, found := ch.replies[req.GetId()]

	if found {
		if req.GetSeq() == lastReply.GetSeq() {
			// Seq equals last reply, retransmit
			if client, found := ch.clients[req.GetId()]; found && client.connected {
				client.WriteAsync(lastReply)
			}
			return
		}
	}

	ch.propChan <- req
}

func (ch *ClientHandlerTCP) handleResponse(resp *Response) {
	cc, found := ch.clients[resp.GetId()]
	if !found || !cc.connected {
		return
	}

	if glog.V(3) {
		glog.Infoln("client found and connected, sending", resp.SimpleString())
	}

	ch.replies[resp.GetId()] = resp
	cc.WriteAsync(resp)
}

// redirect returns true if a request should be redirected to the current
// leader.
func (ch *ClientHandlerTCP) redirect() bool {
	// If we're not the leader and don't allow direct messages, then
	// we redirect the client to the leader.
	return ch.id != ch.leader && !allowDirect[ch.paxosType]
}

func (ch *ClientHandlerTCP) handleGrpHold(gp *sync.WaitGroup) {
	glog.V(2).Info("grpmgr hold req")
	gp.Done()
	<-ch.grpSubscriber.ReleaseChan()
	glog.V(2).Info("grpmgr release")
	var leader bool
	leader = ch.id == ch.leader
	ch.id = ch.grpmgr.GetID()
	glog.V(2).Infoln("new id is", ch.id)
	if leader {
		ch.leader = ch.id
	}
}

func (ch *ClientHandlerTCP) getPaxosType() *Response_Protocol {
	switch ch.paxosType {
	case "batchpaxos":
		return Response_BATCHPAXOS.Enum()
	case "fastpaxos":
		return Response_FASTPAXOS.Enum()
	case "multipaxos":
		return Response_MULTIPAXOS.Enum()
	default:
		return Response_UNKNOWN.Enum()
	}
}
