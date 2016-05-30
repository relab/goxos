package client

import (
	"log"
	"net"
	"sync"
	"time"

	"github.com/relab/goxos/config"
)

type FastConnection struct {
	id                 string
	replicaConnections []net.Conn
	seq                uint32
	receivedFrom       map[uint32][]net.Conn
	writeLock          *sync.Mutex
	respMap            responseMap
	conf               *config.Config
}

func NewFastConn(conn *ReplicaConn, connections []net.Conn) ServiceConn {
	c := FastConnection{
		id:                 conn.id,
		replicaConnections: connections,
		receivedFrom:       make(map[uint32][]net.Conn),
		respMap:            newResponseMap(),
		conf:               conn.conf,
	}
	c.startReadWorkers()
	return &c
}

func (c *FastConnection) Send(request []byte) <-chan ResponseData {
	req := c.genReq(request)
	respInfo := newResponseInfo()
	c.respMap.add(req.GetSeq(), respInfo)

	go c.trySend(req, respInfo)
	return respInfo.respChan
}

func (c *FastConnection) trySend(request Request, respInfo *responseInfo) {

	for retries := 0; retries < 30; retries++ {
		err := c.writeToAllReplicas(request)
		if err != nil {
			//send err to response channel for this request
			// c.mapLock.Lock()
			// responseChan, exist := c.futureResponses[request.GetSeq()]
			// c.mapLock.Unlock()

			if respInfo != nil {
				respInfo.respChan <- ResponseData{Value: request.GetVal(), Err: err}
			}
			break
		}

		time.Sleep(c.conf.GetDuration("awaitResponseTimeout", config.DefAwaitResponseTimeout))

		// Maybe check here if we have received answer from some replicas (but not all) and in that case send error?
		// This is because answer from some (but not all) replicas might point to a read error,
		// if it happens several times in row. The previous version of fastConnection returned error if read gave
		// an error
		// if respInfo.isReceived() {
		// 	c.respMap.delete(request.GetSeq())
		// 	return
		// }
	}
}

func (c *FastConnection) writeToAllReplicas(request Request) error {
	c.writeLock.Lock()
	defer c.writeLock.Unlock()
	for _, conn := range c.replicaConnections {
		err := write(conn, &request)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *FastConnection) startReadWorkers() {
	for _, conn := range c.replicaConnections {
		go func(conn net.Conn) {
			for {
				var response Response
				err := read(conn, &response)
				if err != nil {
					if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
						//ignore if timeout
						continue
					}
					log.Printf("worker: read error, %v\n", err)
					continue
				}

				if response.HasError() {
					log.Printf("worker: error code nonzero, %v - %v\n",
						response.GetErrorCode(), response.GetErrorDetail())
					continue
				}
				//TODO: This code needs the mapLock, but has temporarily been disabled (while refactoring)
				// c.mapLock.Lock()
				receivedFrom, exist := c.receivedFrom[response.GetSeq()]
				if !exist {
					receivedFrom = make([]net.Conn, 0)
				}
				receivedFrom = append(receivedFrom, conn)
				c.receivedFrom[response.GetSeq()] = receivedFrom
				// c.mapLock.Unlock()

				//TODO: It seems problematic to require all replicaConnections
				// to respond. Are we assuming that replicaConnections refer to
				// only live connections?
				if len(receivedFrom) != len(c.replicaConnections) {
					continue
				}

				respInfo, exist := c.respMap.get(response.GetSeq())
				if !exist {
					log.Println("worker: no sequence information found for: ", response.GetSeq())
					return
				}
				// respInfo.setReceived(true)
				resp := ResponseData{Value: response.GetVal(), SendTime: respInfo.sendTime, ReceiveTime: time.Now()}
				respInfo.respChan <- resp
				//go deliverResponseToChannel(responseChan, resp)

			}
		}(conn)
	}
}

func (c *FastConnection) Close() error {
	var err error
	for _, conn := range c.replicaConnections {
		log.Println("close: closing", conn.RemoteAddr())
		err = conn.Close()
	}
	return err
}

func (c *FastConnection) genReq(value []byte) Request {
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
