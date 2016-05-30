package server

import (
	"strings"
	"sync"

	"github.com/relab/goxos/app"
	"github.com/relab/goxos/client"
	"github.com/relab/goxos/config"
	"github.com/relab/goxos/elog"
	e "github.com/relab/goxos/elog/event"
	"github.com/relab/goxos/grp"
	"github.com/relab/goxos/net"
	"github.com/relab/goxos/paxos"
	"github.com/relab/goxos/ringreplacer"

	"github.com/relab/goxos/Godeps/_workspace/src/github.com/golang/glog"
)

// Shared constructor for both a replacer node and a reconfiguration node
func NewStandbyServer(id grp.ID, appID string, nodemap grp.NodeMap,
	conf config.Config, ah app.Handler, slotMarker paxos.SlotID) *Server {
	s := &Server{
		id:                 id,
		appID:              appID,
		nodes:              nodemap,
		config:             conf,
		outUnicast:         make(chan net.Packet, 64),
		outBroadcast:       make(chan interface{}, 64),
		outProposer:        make(chan interface{}, 64),
		outAcceptor:        make(chan interface{}, 64),
		outLearner:         make(chan interface{}, 64),
		heartbeatChan:      make(chan grp.ID, 128),
		propChan:           make(chan *paxos.Value, 32),
		decidedChan:        make(chan *paxos.Value, 32),
		clientReqChan:      make(chan *client.Request, 64),
		propDcdChan:        make(chan bool, 32),
		appStateReqChan:    make(chan app.StateReq),
		reconfigCmdChan:    make(chan paxos.ReconfigCmd),
		recMsgChan:         make(chan ringreplacer.ReconfMsg),
		localAru:           paxos.NewAdu(slotMarker),
		firstSlot:          slotMarker + 1,
		ah:                 ah,
		stopChan:           make(chan bool),
		subModulesStopSync: new(sync.WaitGroup),
	}

	switch strings.ToLower(conf.GetString("failureHandlingType", config.DefFailureHandlingType)) {
	case "livereplacement", "areconfiguration":
		s.replacer = true
	}

	return s
}

func (s *Server) StartReplacer() {
	s.grpmgrStart()
	s.networkStart(false)
	s.startHbEmitter()
	s.waitForActivation()
	s.paxosStart()
	s.clientHandlerStart()
	s.startFdAndLd()
	go s.run()
}

func (s *Server) waitForActivation() {
	s.failureHandlingStart()
	glog.V(1).Info("waiting for replacer activation")
	elog.Log(e.NewEvent(e.LRWaitForActivation))
	s.replacementHandler.WaitForActivation()
	elog.Log(e.NewEvent(e.LRActivated))
}

func (s *Server) StartReconfig() {
	s.grpmgrStart()
	s.networkStart(false)
	s.ringReplacerStart()
	s.failureHandlingStart()
	s.firstSlot = s.waitForFirstSlot()
	elog.Log(e.NewEvent(e.ReconfigFirstSlotReceived))
	s.initPaxos()
	s.paxosStart()
	s.waitForJoin()
	elog.Log(e.NewEvent(e.ReconfigJoined))
	s.startHbEmitter()
	s.clientHandlerStart()
	s.startFdAndLd()
	go s.run()
}

func (s *Server) waitForFirstSlot() paxos.SlotID {
	glog.V(1).Info("waiting for first slot before starting paxos")
	return s.reconfigHandler.WaitForFirstSlot()
}

func (s *Server) waitForJoin() {
	glog.V(1).Info("waiting for join before starting client & liveness modules")
	s.reconfigHandler.WaitForJoin()
}

func (s *Server) StartAReconfig() {
	s.grpmgrStart()
	s.networkStart(false)
	s.startHbEmitter()
	s.failureHandlingStart()
	s.waitForValidActivation()
	s.paxosStart()
	s.clientHandlerStart()
	s.startFdAndLd()
	go s.run()
}

/*
//TODO: implement in arec: WaitForReconfMsg
func (s *Server) waitForReconf() paxos.SlotId {
	glog.V(4).Info("waiting for reconf")
	return s.aReconfHandler.WaitForReconf()
}
*/

//TODO: implement in arec: WaitForValidActivation
func (s *Server) waitForValidActivation() {
	glog.V(1).Info("waiting for valid activation")
	s.aReconfHandler.RunningPaxos()
}
