package client

import (
	"log"
	"net"
	"sync"
	"time"

	"github.com/relab/goxos/config"
)

const (
	batchReadTimeout = 10000 * time.Millisecond
)

type BatchConnection struct {
	id                 string
	replicaConnections []net.Conn
	seq                uint32
	writeLock          *sync.Mutex
	respMap            responseMap
	conf               *config.Config
}

func NewBatchConn(conn *ReplicaConn, connections []net.Conn) ServiceConn {
	batchConn := BatchConnection{
		id:                 conn.id,
		replicaConnections: connections,
		respMap:            newResponseMap(),
		writeLock:          new(sync.Mutex),
		conf:               conn.conf,
	}
	batchConn.startReadWorkers()
	return &batchConn
}

func (batchConn *BatchConnection) Send(request []byte) <-chan ResponseData {
	req := batchConn.genReq(request)
	respInfo := newResponseInfo()
	batchConn.respMap.add(req.GetSeq(), respInfo)

	go batchConn.trySend(req, respInfo)
	return respInfo.respChan
}

func (batchConn *BatchConnection) trySend(request Request, respInfo *responseInfo) {

	for {
		err := batchConn.writeToAllReplicas(request)
		if err != nil {
			log.Println("trySend: error on write: ", err)
			//Should we return error here? Should we just continue if error is timeout?

			//send err to response channel for this request
			// batchConn.mapLock.Lock()
			// responseChan, exist := batchConn.futureResponses[request.GetSeq()]
			// batchConn.mapLock.Unlock()
			// if exist {
			// 	responseChan <- Response{Value: request.GetVal(), Err: err}
			// }
			break
		}

		time.Sleep(batchConn.conf.GetDuration("awaitResponseTimeout", config.DefAwaitResponseTimeout))

		// if respInfo.isReceived() {
		// 	batchConn.respMap.delete(request.GetSeq())
		// 	return
		// }
	}
}

func (batchConn *BatchConnection) writeToAllReplicas(request Request) error {
	batchConn.writeLock.Lock()
	defer batchConn.writeLock.Unlock()
	for _, conn := range batchConn.replicaConnections {
		err := write(conn, &request)
		if err != nil {
			return err
		}
	}
	return nil
}

func (batchConn *BatchConnection) startReadWorkers() {
	for _, conn := range batchConn.replicaConnections {
		go func(conn net.Conn) {
			for {
				var response Response
				//TODO Do we need this deadline?
				conn.SetReadDeadline(time.Now().Add(batchReadTimeout))
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

				respInfo, exist := batchConn.respMap.get(response.GetSeq())
				if !exist {
					log.Println("worker: no sequence information found for: ", response.GetSeq())
					return
				}
				// respInfo.setReceived(true)
				resp := ResponseData{Value: response.GetVal(), SendTime: respInfo.sendTime, ReceiveTime: time.Now()}
				respInfo.respChan <- resp
			}
		}(conn)
	}
}

func (batchConn *BatchConnection) Close() error {
	var err error
	for _, conn := range batchConn.replicaConnections {
		log.Println("close: closing", conn.RemoteAddr())
		err = conn.Close()
	}
	return err
}

func (batchConn *BatchConnection) genReq(value []byte) Request {
	seq := batchConn.seq
	req := Request{
		Type: Request_EXEC.Enum(),
		Id:   &batchConn.id,
		Seq:  &seq,
		Val:  value,
	}
	batchConn.seq++
	return req
}
