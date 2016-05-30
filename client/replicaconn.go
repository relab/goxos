package client

import (
	"errors"
	"log"
	"net"
	"sync"
	"time"

	"github.com/relab/goxos/config"
)

type ReplicaConn struct {
	id                  string
	conn                net.Conn
	seq                 uint32
	nextNodeToConnectTo int
	respMap             responseMap
	reqSent             chan bool
	handshakeLock       *sync.RWMutex //this is used to avoid the bug where tryReceive would sometimes read the response intended to be read in exhangeID
	nodes               []string
	conf                *config.Config
}

func newReplicaConn(config *config.Config) *ReplicaConn {
	return &ReplicaConn{
		respMap:       newResponseMap(),
		reqSent:       make(chan bool, 560),
		handshakeLock: new(sync.RWMutex),
		nodes:         make([]string, 0),
		conf:          config,
	}
}

func (c *ReplicaConn) Send(request []byte) <-chan ResponseData {
	req := c.genReq(request)
	respInfo := newResponseInfo()
	c.respMap.add(req.GetSeq(), respInfo)

	go c.trySend(&req, respInfo)
	return respInfo.respChan
}

func (c *ReplicaConn) Close() error {
	return c.conn.Close()
}

// trySend tries to send the request to the currently connected replica. If
// the current connection times out or the connection is otherwise lost, we
// try to connect to another replica and retry sending the request to that
// replica.
func (c *ReplicaConn) trySend(req *Request, respInfo *responseInfo) {
	for {
		// The write timeout should be long enough so that we don't need to
		// retry writing to the same replica twice because if we timeout, we
		// simply reconnect to another replica.
		c.conn.SetWriteDeadline(time.Now().Add(c.conf.GetDuration("writeTimeout", config.DefWriteTimeout)))
		err := c.write(req)
		if err != nil {
			log.Println("trySend: write failed:", err)
			// sleep to allow new leader to emerge
			// TODO: maybe this should be coordinated with liveness value on replicas?
			// perhaps this is something that can be sent in the handshake?
			time.Sleep(300 * time.Millisecond)
			err := c.reconnect()
			if err != nil {
				log.Println("trySend: reconnect failed; giving up:", err)
				respInfo.respChan <- ResponseData{Err: err}
				return
			}
			// we got a connection; try to write request again
			//time.Sleep(100 * time.Millisecond)
			log.Println("trySend: reconnected, resending seq#: ", req.GetSeq())
			continue
		}

		// the request has been sent; signal receiver to begin receiving
		c.reqSent <- true
		// garbage collect on return
		defer func() { c.respMap.delete(req.GetSeq()) }()

		// await response delivery or timeout
		select {
		case <-time.After(c.conf.GetDuration("awaitResponseTimeout", config.DefAwaitResponseTimeout)):
			log.Println("trySend: timeout:", req.GetSeq())
			// giving up; returning error on client's response channel
			respInfo.respChan <- ResponseData{Err: errors.New("timeout")}
			return
		case <-respInfo.resend:
			//resending request
			log.Println("trySend: resending seq#:", req.GetSeq())
		case <-respInfo.gotResponse:
			log.Println("trySend: got response, seq#:", req.GetSeq())
			return

		}
	}
}

func (c *ReplicaConn) write(req *Request) error {
	//c.writeLock.Lock()
	//defer c.writeLock.Unlock()
	c.handshakeLock.RLock()
	defer c.handshakeLock.RUnlock()
	return write(c.conn, req)
}

func (c *ReplicaConn) tryReceive() {
	for {
		var r Response
		<-c.reqSent // wait until something has been sent
		c.conn.SetReadDeadline(time.Now().Add(c.conf.GetDuration("readTimeout", config.DefReadTimeout)))
		// await reply
		c.handshakeLock.RLock()
		err := read(c.conn, &r)
		c.handshakeLock.RUnlock()
		if err != nil {
			log.Println("tryReceive: read failed:", err, c.conn.RemoteAddr())
			time.Sleep(100 * time.Millisecond)
			err := c.reconnect()
			if err != nil {
				log.Println("tryReceive: reconnect failed; giving up:", err)
				// returning error on all response channels for the client
				for seqNum, respInfo := range c.respMap.respMap {
					log.Println("tryReceive: failed to send seq#:", seqNum)
					respInfo.respChan <- ResponseData{Err: err}
				}
				return
			}
			//time.Sleep(100 * time.Millisecond)
			// got connection; resend outstanding requests and await replies
			for seqNum, respInfo := range c.respMap.respMap {
				log.Println("tryReceive: trigger resend for seq#:", seqNum)
				select {
				case respInfo.resend <- struct{}{}: //resending
				default:
				}
			}
			continue
		}

		// we got a reply
		respInfo, exist := c.respMap.get(r.GetSeq())
		if !exist {
			//TODO what could cause this?
			// getting a redirect could cause this, should it be handled here?
			// (this can also happen if we receive multiple responses with same seq)
			//log.Println("tryReceive: response channel not found for:", r.GetSeq(), r)
			//respInfo.resend <- true //resending

			// if the request timed out, and then we got this response, we
			// have no way to reach the client to pass along the reply.
			log.Println("tryReceive: response channel for", r,
				"has been garbage collected; ignoring.")
			continue
		}
		//TODO: We can avoid the ResponseData struct by adding SendTime and ReceiveTime to Response in msg.proto
		resp := ResponseData{Value: r.GetVal(), SendTime: respInfo.sendTime, ReceiveTime: time.Now()}
		// deliver to channels non-blocking, in case of mulitple responses with same seq number
		select {
		case respInfo.gotResponse <- struct{}{}:
		default:
		}

		select {
		case respInfo.respChan <- resp:
		default:
		}

	}
}

var reconnecting bool

func (c *ReplicaConn) reconnect() error {
	if reconnecting {
		return nil
	}
	reconnecting = true
	defer func() { reconnecting = false }()

	log.Println("reconnect: connecting to another replica")
	_, err := c.handshake()
	return err
}

func (c *ReplicaConn) tcpConnect() error {
	maxAttempts := len(c.nodes) * c.conf.GetInt("cycleListMax", config.DefCycleListMax)
	for i := 0; i < maxAttempts; i++ {
		log.Printf("tcpConnect: connecting to node %v", c.nextNodeToConnectTo)
		newConn, err := net.DialTimeout("tcp", c.nodes[c.nextNodeToConnectTo], c.conf.GetDuration("dialTimeout", config.DefDialTimeout))
		if err == nil {
			log.Printf("tcpConnect: connected to node %v", c.nodes[c.nextNodeToConnectTo])
			c.conn = newConn
			c.nextNodeToConnectTo = (c.nextNodeToConnectTo + 1) % len(c.nodes)
			return nil
		}
		log.Println("tcpConnect: error:", err)
		log.Println("tcpConnect: sleeping", c.conf.GetDuration("cycleNodesWait", config.DefCycleNodesWait))
		time.Sleep(c.conf.GetDuration("cycleNodesWait", config.DefCycleNodesWait))
		c.nextNodeToConnectTo = (c.nextNodeToConnectTo + 1) % len(c.nodes)
		continue
	}

	return errors.New("cannot contact any node in cluster")
}

func (c *ReplicaConn) genReq(value []byte) Request {
	seq := c.seq
	req := Request{
		Type: Request_EXEC.Enum(),
		Id:   &c.id,
		Seq:  &seq,
		Val:  value,
	}
	c.seq++
	return req
}
