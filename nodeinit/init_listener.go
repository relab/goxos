package nodeinit

import (
	"fmt"
	"net"
	"sync"

	"github.com/relab/goxos/app"
	"github.com/relab/goxos/config"
	"github.com/relab/goxos/grp"
	gnet "github.com/relab/goxos/net"

	"github.com/relab/goxos/elog"
	e "github.com/relab/goxos/elog/event"

	"github.com/relab/goxos/Godeps/_workspace/src/github.com/golang/glog"
)

const defaultActivationPort = "10000"

const (
	Preinit = iota
	Listening
	Responding
	Transfering
	Initialized
	Activated
	Stopped
)

type state int

var states = [...]string{
	"Pre-init",
	"Listening",
	"Responding",
	"Transfering",
	"Initialized",
	"Activated",
	"Stopped",
}

func (s state) String() string {
	return states[s]
}

type initState struct {
	s state
	m sync.Mutex
}

type InitData struct {
	ID               grp.ID
	Config           config.TransferWrapper
	AppState         app.State
	ReplacedNode     grp.Node
	applyStateResult chan error
}

func (id *InitData) ApplyStateResult(err error) {
	id.applyStateResult <- err
}

type InitListener struct {
	initAddr           string
	listener           net.Listener
	initState          initState
	configAndStateChan chan InitData
}

func NewInitListener(ip string) *InitListener {
	return &InitListener{
		initAddr:           ip + ":" + defaultActivationPort,
		initState:          initState{s: Preinit},
		configAndStateChan: make(chan InitData),
	}
}

func (sl *InitListener) Start() (err error) {
	glog.V(1).Infof("listening to %v", sl.initAddr)
	sl.listener, err = net.Listen("tcp", sl.initAddr)
	if err != nil {
		glog.Fatalln("couldn't listen on provided address:", err)
		return err
	}
	sl.listenForConnections()
	return nil
}

func (sl *InitListener) listenForConnections() {
	sl.setState(Listening)
	go func() {
		defer sl.Stop()
		for {
			conn, err := sl.listener.Accept()
			if err != nil {
				if gnet.IsSocketClosed(err) {
					glog.Warningln("init listener closed; exiting")
					break
				}
				glog.Error(err)
				continue
			}
			glog.V(2).Infoln("received connection from", conn.RemoteAddr())
			go sl.handleConnection(conn)
		}
	}()
}

func (sl *InitListener) Stop() {
	sl.setState(Stopped)
	sl.listener.Close()
}

func (sl *InitListener) handleConnection(conn net.Conn) {
	connection := gnet.NewConnection(conn)
	defer connection.Close()

	var err error

	glog.V(2).Info("waiting for init request")
	var initRequest InitRequest
	if err = connection.Dec.Decode(&initRequest); err != nil {
		if err != nil {
			logError(err, "decoding", connection)
			return
		}
	}

	glog.V(2).Infoln("received InitRequest from", initRequest.SenderID)
	elog.Log(e.NewEvent(e.InitTransferStart))
	defer elog.Log(e.NewEvent(e.InitTransferDone))

	sl.initState.m.Lock()
	currentState := sl.initState.s
	if currentState == Listening {
		glog.V(2).Info("state is listening, responding with ack")
		sl.initState.s = Responding
		sresp := InitResponse{true, sl.initState.s}
		if err = connection.Enc.Encode(sresp); err != nil {
			sl.initState.s = Listening
			sl.initState.m.Unlock()
			logError(err, "encoding", connection)
			return
		}
	} else {
		sl.initState.m.Unlock()
		glog.V(2).Infoln("responding with nack due to state:", currentState)
		sresp := InitResponse{false, sl.initState.s}
		if err = connection.Enc.Encode(sresp); err != nil {
			logError(err, "encoding", connection)
		}
		return
	}

	glog.V(2).Info("waiting for transfer request")
	var initData InitData
	sl.initState.s = Transfering
	sl.initState.m.Unlock()
	if err = connection.Dec.Decode(&initData); err != nil {
		if err != nil {
			logError(err, "decoding", connection)
			sl.setState(Listening)
			return
		}
	}

	glog.V(2).Info("sending init data to goxos")
	initData.applyStateResult = make(chan error)
	sl.configAndStateChan <- initData
	err = <-initData.applyStateResult
	if err != nil {
		glog.Errorln("applying init data failed, reason:", err, "responding with error")
		tresp := TransferResponse{false, err.Error()}
		if err = connection.Enc.Encode(tresp); err != nil {
			logError(err, "encoding", connection)
		}
		glog.V(2).Info("will continue to listen for init requests")
		sl.setState(Listening)
		return
	}

	glog.V(2).Info("applying initialization data was successful, responding with ack")
	tresp := TransferResponse{true, ""}
	if err = connection.Enc.Encode(tresp); err != nil {
		logError(err, "encoding", connection)
		glog.Warning("response with initialization success failed to be sent")
		sl.setState(Initialized)
		return
	}
	glog.V(2).Info("response with init ack sent")
	sl.setState(Initialized)
}

func (sl *InitListener) WaitForInitData() (InitData, error) {
	sl.initState.m.Lock()
	currentState := sl.initState.s
	sl.initState.m.Unlock()
	if !canWaitForInitData(currentState) {
		return InitData{}, fmt.Errorf("can't wait for init data due to current state (%v)", currentState)
	}
	glog.V(2).Infoln("waiting for state and config")
	return <-sl.configAndStateChan, nil
}

func (sl *InitListener) SetActivated() error {
	sl.initState.m.Lock()
	defer sl.initState.m.Unlock()
	if sl.initState.s != Initialized {
		return fmt.Errorf("could not set activated due to current state (%v)", sl.initState.s)
	}
	sl.initState.s = Activated
	return nil
}

func (sl *InitListener) setState(state state) {
	sl.initState.m.Lock()
	sl.initState.s = state
	sl.initState.m.Unlock()
}

func canWaitForInitData(state state) bool {
	if state == Initialized || state == Activated || state == Stopped {
		return false
	}
	return true
}

func logError(err error, op string, connection *gnet.Connection) {
	glog.Errorln("error when", op, "message from", connection, ":", err)
}
