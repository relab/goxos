package server

import (
	"strings"

	"github.com/relab/goxos/arec"
	"github.com/relab/goxos/authenticatedbc"
	"github.com/relab/goxos/batchpaxos"
	"github.com/relab/goxos/client"
	"github.com/relab/goxos/config"
	"github.com/relab/goxos/grp"
	"github.com/relab/goxos/liveness"
	"github.com/relab/goxos/lr"
	"github.com/relab/goxos/multipaxos"
	"github.com/relab/goxos/net"
	"github.com/relab/goxos/parallelpaxos"
	"github.com/relab/goxos/paxos"
	"github.com/relab/goxos/reconfig"
	"github.com/relab/goxos/reliablebc"
	"github.com/relab/goxos/ringreplacer"

	"github.com/relab/goxos/Godeps/_workspace/src/github.com/golang/glog"
)

// Initialize all of the modules.
func (s *Server) InitModules() {
	glog.V(1).Info("initializing submodules")
	if glog.V(2) {
		s.logInitInfo()
	}
	s.initGroupManager()
	s.initNetwork()
	s.initLiveness()
	s.initPaxos()
	s.initRingReplacer()
	s.initFailureHandling()
	s.initClientHandler()
}

func (s *Server) InitModulesReconfig() {
	glog.V(1).Info("initializing submodules")
	if glog.V(2) {
		s.logInitInfo()
	}
	s.initGroupManager()
	s.initNetwork()
	s.initLiveness()
	s.initRingReplacer()
	s.initFailureHandling()
	s.initClientHandler()
}

func (s *Server) logInitInfo() {
	glog.Infoln("\n----------------------------------------------",
		"\ninitalization values summary",
		"\napplication identifier:", s.appID,
		"\nid is", s.id,
		"\nThere are", s.nodes.NrOfNodes(), "nodes and",
		s.nodes.NrOfAcceptors(),
		"acceptors in cluster",
		"\npaxos type is",
		s.config.GetString("protocol", config.DefProtocol),
		"with alpha",
		s.config.GetInt("alpha", config.DefAlpha),
		"\nfailure handling type is", s.config.GetString(
			"failureHandlingType",
			config.DefFailureHandlingType,
		),
		"\nreplacer mode:", s.replacer,
		"\nliveness values:",
		s.config.GetDuration(
			"hbEmitterInterval",
			config.DefHbEmitterInterval,
		),
		s.config.GetDuration(
			"fdTimeoutInterval", config.DefFdTimeoutInterval,
		),
		s.config.GetDuration(
			"fdDeltaIncrease",
			config.DefFdDeltaIncrease,
		),
		"\nbatch max size:", s.config.GetInt(
			"batchMaxSize",
			config.DefBatchMaxSize,
		),
		"\nbatch timeout:", s.config.GetDuration(
			"batchTimeout",
			config.DefBatchTimeout,
		),
		"\nthroughput sampling interval:", s.config.GetDuration(
			"throughputSamplingInterval",
			config.DefThroughputSamplingInterval,
		),
		"\n----------------------------------------------")
}

// Parses the config value `nodes`
func (s *Server) initNodeMap() {
	nodeMap, err := s.config.GetNodeMap("nodes")
	if err != nil {
		panic(err.Error())
	}
	s.nodes = nodeMap
}

func (s *Server) initGroupManager() {
	lrEnabled := strings.ToLower(s.config.GetString("failureHandlingType", config.DefFailureHandlingType)) == "livereplacement"
	arEnabled := strings.ToLower(s.config.GetString("failureHandlingType", config.DefFailureHandlingType)) == "areconfiguration"
	s.grpmgr = grp.NewGrpMgr(s.id, &s.nodes, lrEnabled, arEnabled, s.subModulesStopSync)
}

func (s *Server) initNetwork() {
	net.SetHeartbeatChan(s.heartbeatChan)
	s.dmx = net.NewTcpDemuxer(s.id, s.grpmgr, s.subModulesStopSync)
	s.snd = net.NewSender(s.id, s.grpmgr, s.outUnicast, s.outBroadcast,
		s.outProposer, s.outAcceptor, s.outLearner, s.dmx, s.subModulesStopSync)
}

func (s *Server) initLiveness() {
	protocol := s.config.GetString("protocol", config.DefProtocol)
	switch strings.TrimSpace(strings.ToLower(protocol)) {
	case "authenticatedbc", "reliablebc":
		return
	default:
		s.hbem = liveness.NewHbEm(s.config, s.id, s.snd.ResetChan(),
			s.outBroadcast, s.subModulesStopSync)
		s.fd = liveness.NewFd(s.id, s.grpmgr, s.config,
			s.heartbeatChan, s.subModulesStopSync)
		s.ld = liveness.NewMonarchicalLD(s.grpmgr, s.fd, s.subModulesStopSync)
		s.pxLeaderChan = s.ld.SubscribeToPaxosLdMsgs("server")
	}
}

func (s *Server) initPaxos() {
	node, exists := s.nodes.LookupNode(s.id)
	if !exists {
		glog.Fatal("initPaxos: can't find self in nodemap")
	}

	pp := &paxos.Pack{
		ID:              s.id,
		NrOfNodes:       s.nodes.NrOfNodes(),
		NrOfAcceptors:   s.nodes.NrOfAcceptors(),
		Config:          &s.config,
		Dmx:             s.dmx,
		Gm:              s.grpmgr,
		Ld:              s.ld,
		StopCheckIn:     s.subModulesStopSync,
		RunProp:         node.Proposer,
		RunAcc:          node.Acceptor,
		RunLrn:          node.Learner,
		Ucast:           s.outUnicast,
		Bcast:           s.outBroadcast,
		BcastP:          s.outProposer,
		BcastA:          s.outAcceptor,
		BcastL:          s.outLearner,
		PropChan:        s.propChan,
		DcdChan:         s.decidedChan,
		NewDcdChan:      s.propDcdChan,
		DcdSlotIDToProp: make(chan paxos.SlotID, 32),
		LocalAdu:        s.localAru,
		FirstSlot:       s.firstSlot,
		NextExpectedDcd: paxos.SlotID(s.localAru.Value() + 1),
	}

	protocol := s.config.GetString("protocol", config.DefProtocol)
	switch strings.TrimSpace(strings.ToLower(protocol)) {
	case "multipaxos":
		s.prop, s.acc, s.lrn = multipaxos.CreateMultiPaxos(pp)
	case "parallelpaxos":
		s.prop, s.acc, s.lrn = parallelpaxos.CreateParallelPaxos(pp)
	case "fastpaxos":
		panic("FastPaxos needs to be updated due to API changes")
	case "batchpaxos":
		s.prop, s.acc, s.lrn = batchpaxos.CreateBatchPaxos(*pp)
	case "authenticatedbc":
		s.acc, s.prop, s.lrn = authenticatedbc.CreateAuthenticatedBC(pp)
	case "reliablebc":
		s.prop, s.acc, s.lrn = reliablebc.CreateReliableBC(pp)

	default:
		panic("Unknown paxos variant: " + protocol + " given as config value for `protocol`.")
	}
}

func (s *Server) initFailureHandling() {
	fhType := s.config.GetString("failureHandlingType", config.DefFailureHandlingType)

	switch strings.TrimSpace(strings.ToLower(fhType)) {
	case "none":
		return
	case "livereplacement":
		s.replacementHandler = lr.NewReplacementHandler(s.id, &s.config, s.replacer,
			s.localAru.Value(), s.grpmgr, s.recMsgChan, s.outBroadcast, s.outUnicast,
			s.dmx, s.acc, s.localAru, s.subModulesStopSync)
	case "reconfiguration":
		s.reconfigHandler = reconfig.NewReconfigHandler(s.id, &s.config, s.appID,
			s.grpmgr, s.ld, s.recMsgChan, s.outBroadcast, s.outUnicast,
			s.dmx, s.appStateReqChan, s.reconfigCmdChan, s.subModulesStopSync)
	case "areconfiguration":
		s.aReconfHandler = arec.NewAReconfHandler(s.id, &s.config, s.appID,
			s.grpmgr, s.fd, s.ld, s.outBroadcast, s.outUnicast, s.dmx,
			s.appStateReqChan, s.acc, s.prop, s.localAru, s.subModulesStopSync)
	default:
		glog.Infoln("Unknown FailureHandlingType: `" + fhType +
			"`. Ignoring and running without failure handling")
		return
	}
}

func (s *Server) initClientHandler() {
	protocol := s.config.GetString("protocol", config.DefProtocol)
	switch strings.TrimSpace(strings.ToLower(protocol)) {
	case "authenticatedbc", "reliablebc":
		s.clientHandler = &client.ClientHandlerMock{}
	default:
		s.clientHandler = client.NewClientHandlerTCP(
			s.id,
			protocol,
			s.grpmgr,
			s.ld,
			s.clientReqChan,
			s.subModulesStopSync,
		)
	}
}

func (s *Server) initRingReplacer() {
	s.ringReplacer = ringreplacer.NewRingReplacer(s.id, &s.config, s.appID, s.grpmgr, s.fd,
		s.appStateReqChan, s.recMsgChan, s.subModulesStopSync)
}
