package server

import (
	"strings"

	"github.com/relab/goxos/config"

	"github.com/relab/goxos/Godeps/_workspace/src/github.com/golang/glog"
)

// Start all of the submodules. This must occur in the correct order, so that
// the network is launched before Paxos, etc.
func (s *Server) Start() {
	glog.V(1).Info("starting submodules")
	s.grpmgrStart()
	s.networkStart(true)
	s.paxosStart()
	s.clientHandlerStart()
	s.ringReplacerStart()
	s.failureHandlingStart()
	s.livenessStart()
	go s.run()
}

func (s *Server) grpmgrStart() {
	s.subModulesStopSync.Add(1)
	s.grpmgr.Start()
}

func (s *Server) networkStart(doInitialConnect bool) {
	s.subModulesStopSync.Add(1)
	s.dmx.Start()
	if doInitialConnect {
		s.snd.InitialConnect()
	}
	s.subModulesStopSync.Add(1)
	s.snd.Start()
}

func (s *Server) paxosStart() {
	s.subModulesStopSync.Add(1)
	s.prop.Start()
	s.subModulesStopSync.Add(1)
	s.acc.Start()
	s.subModulesStopSync.Add(1)
	s.lrn.Start()
}

func (s *Server) livenessStart() {
	protocol := s.config.GetString("protocol", config.DefProtocol)
	switch strings.TrimSpace(strings.ToLower(protocol)) {
	case "authenticatedbc", "reliablebc":
		return
	default:
		s.startHbEmitter()
		s.startFdAndLd()
	}
}

func (s *Server) startHbEmitter() {
	s.subModulesStopSync.Add(1)
	s.hbem.Start()
}

func (s *Server) startFdAndLd() {
	s.subModulesStopSync.Add(1)
	s.fd.Start()
	s.subModulesStopSync.Add(1)
	s.ld.Start()
}

func (s *Server) clientHandlerStart() {
	s.subModulesStopSync.Add(1)
	s.clientHandler.Start()
}

func (s *Server) failureHandlingStart() {
	switch strings.ToLower(s.config.GetString("failureHandlingType", config.DefFailureHandlingType)) {
	case "none":
		return
	case "livereplacement":
		s.subModulesStopSync.Add(1)
		s.replacementHandler.Start()
	case "reconfiguration":
		s.subModulesStopSync.Add(1)
		s.reconfigHandler.Start()
	case "areconfiguration":
		s.subModulesStopSync.Add(1)
		s.aReconfHandler.Start()
	default:
		glog.Warning("unknown failure handling method specified, using none")
	}

}

func (s *Server) ringReplacerStart() {
	s.subModulesStopSync.Add(1)
	s.ringReplacer.Start()
}
