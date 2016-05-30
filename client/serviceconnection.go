package client

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"

	"github.com/relab/goxos/config"
)

type ServiceConn interface {
	Send(request []byte) <-chan ResponseData
	Close() error
}

func Dial(conf *config.Config) (ServiceConn, error) {
	c := newReplicaConn(conf)
	var err error

	log.SetPrefix("client ")
	c.id, err = generateID()
	if err != nil {
		return nil, err
	}
	log.Println("dial: generated client id:", c.id)

	log.Println("dial: reading configuration")
	nodeMap, err := conf.GetNodeMap("nodes")
	if err != nil {
		return nil, err
	}
	for _, node := range nodeMap.Nodes() {
		c.nodes = append(c.nodes, node.ClientAddr())
	}

	log.Println("dial: number of remote nodes in config is", len(c.nodes))
	return c.handshake()
}

var handShaking bool

func (c *ReplicaConn) handshake() (ServiceConn, error) {
	log.Println("handshaking is: ", handShaking)
	if !handShaking {
		handShaking = true
		c.handshakeLock.Lock()
		defer c.handshakeLock.Unlock()
		defer func() { handShaking = false }()
	}

	if err := c.tcpConnect(); err != nil {
		log.Println("handshake: tcp connect error: ", err)
		return nil, err
	}
	hsResp, err := handshakeReq(c.conn, &c.id)
	if err != nil {
		return nil, err
	}

	return c.handshakeResp(hsResp)
}

func handshakeReq(conn net.Conn, id *string) (*Response, error) {
	hsResp, err := exchangeID(conn, id)
	addr := conn.RemoteAddr()
	if err != nil {
		if err == io.EOF {
			log.Printf("handshake(%s): connection closed", addr)
		} else if neterr, ok := err.(net.Error); ok && neterr.Timeout() {
			log.Printf("handshake(%s): timeout", addr)
		} else {
			log.Printf("handshake(%s): unexpected error: %v", addr, err)
		}
		// Leave it to the caller to retry
		conn.Close()
		return nil, err
	}
	if hsResp.GetType() != Response_HELLO_RESP {
		// Leave it to the caller to retry
		conn.Close()
		return nil, errors.New("unknown response type" + hsResp.GetErrorDetail())
	}

	return hsResp, nil
}

func (c *ReplicaConn) handshakeResp(hsResp *Response) (ServiceConn, error) {
	switch hsResp.GetErrorCode() {
	case Response_NONE:
		log.Println("handshake: id accepted")
		return c.checkPaxosType(hsResp)
	case Response_REDIRECT:
		newNode := hsResp.GetErrorDetail()
		log.Println("handshake: redirected to", newNode)
		c.nextNodeToConnectTo = addNodeIfMissing(c.nodes, newNode)
		c.Close()
		// TODO: WARNING: This is a recursive call; should be avoided
		return c.handshake()
	default:
		c.Close()
		return nil, errors.New("unknown response from replica")
	}
}

func (c *ReplicaConn) checkPaxosType(response *Response) (ServiceConn, error) {
	switch response.GetProtocol() {
	case Response_MULTIPAXOS:
		log.Println("checkPaxosType: multipaxos reported in handshake")
		go c.tryReceive()
		return c, nil
	case Response_FASTPAXOS:
		log.Println("checkPaxosType: fastpaxos reported in handshake")
		return handleFastPaxosRunningOnService(c)
	case Response_BATCHPAXOS:
		log.Println("checkPaxosType: batchpaxos reported in handshake")
		return handleBatchPaxosRunningOnService(c)
	default:
		log.Println("checkPaxosType: unknown paxos type in handshake:", response.GetProtocol())
		return nil, errors.New("unknown paxos type reported in handshake")
	}
}

func handleFastPaxosRunningOnService(c *ReplicaConn) (ServiceConn, error) {
	connections, err := connectToRemaining(c)
	if err != nil {
		return nil, err
	}
	connections = append(connections, c.conn)
	return NewFastConn(c, connections), nil
}

func handleBatchPaxosRunningOnService(c *ReplicaConn) (ServiceConn, error) {
	connections, err := connectToRemaining(c)
	if err != nil {
		return nil, err
	}
	connections = append(connections, c.conn)
	return NewBatchConn(c, connections), nil
}

func connectToRemaining(c *ReplicaConn) ([]net.Conn, error) {
	var connections []net.Conn
	// we already have a connection to this replica
	connectedAdr := c.conn.RemoteAddr().String()
	for _, address := range c.nodes {
		if address == connectedAdr {
			continue
		}
		conn, err := net.DialTimeout("tcp", address, c.conf.GetDuration("dialTimeout", config.DefDialTimeout))
		if err != nil {
			return nil, err
		}
		log.Println("connect: connected to " + address)

		//TODO: We are not looking at the handshake response below.
		if _, err := handshakeReq(conn, &c.id); err == nil {
			connections = append(connections, conn)
		} else {
			return nil, err
		}
	}
	return connections, nil
}

// TODO These read() and write() do not timeout
func exchangeID(conn net.Conn, id *string) (*Response, error) {
	hello := &Request{Type: Request_HELLO.Enum(), Id: id}
	if err := write(conn, hello); err != nil {
		return nil, fmt.Errorf("exhangeID-write %v", err)
	}

	var idresp Response
	if err := read(conn, &idresp); err != nil {
		return nil, fmt.Errorf("exhangeID-read %v", err)
	}

	return &idresp, nil
}

func addNodeIfMissing(nodeList []string, possibleNewNode string) int {
	for i, n := range nodeList {
		if n == possibleNewNode {
			return i
		}
	}
	nodeList = append(nodeList, possibleNewNode)
	return len(nodeList) - 1
}
