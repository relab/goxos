package server

import (
	"fmt"
	"strings"
	"time"

	"github.com/relab/goxos/config"

	"github.com/relab/goxos/elog"
	e "github.com/relab/goxos/elog/event"

	"github.com/relab/goxos/Godeps/_workspace/src/github.com/golang/glog"
)

const shutdownTimeout = 500 * time.Millisecond

var ErrShutdownTimeout = fmt.Errorf("stopping goxos submodules timed out (waited %v)", shutdownTimeout)

// Stop all of the submodules.
func (s *Server) Stop() error {
	elog.Log(e.NewEvent(e.ShutdownStart))
	glog.V(1).Info("starting shutdown procedure")
	glog.V(2).Infoln("local aru is", s.localAru)

	s.stopChan <- true
	stopped := make(chan bool)
	go func() {
		glog.V(1).Info("waiting for submodules to report stopped")
		s.subModulesStopSync.Wait()
		stopped <- true
	}()

	select {
	case <-stopped:
		glog.V(1).Info("clean exit")
	case <-time.After(shutdownTimeout):
		glog.Error(ErrShutdownTimeout)
		return ErrShutdownTimeout
	}

	return nil
}

func (s *Server) stop() {
	glog.V(1).Info("signaling stop to submodules")
	s.grpmgr.Stop()
	s.networkStop()
	s.paxosStop()
	s.livenessStop()
	s.clientHandler.Stop()
	s.failureHandlingStop()
}

func (s *Server) networkStop() {
	s.dmx.Stop()
	s.snd.Stop()
}

func (s *Server) livenessStop() {
	protocol := s.config.GetString("protocol", config.DefProtocol)
	switch strings.TrimSpace(strings.ToLower(protocol)) {
	case "authenticatedbc", "reliablebc":
		return
	default:
		s.hbem.Stop()
		s.fd.Stop()
		s.ld.Stop()
	}
}

func (s *Server) paxosStop() {
	s.prop.Stop()
	s.acc.Stop()
	s.lrn.Stop()
}

func (s *Server) failureHandlingStop() {
	switch strings.ToLower(s.config.GetString("failureHandlingType", config.DefFailureHandlingType)) {
	case "none":
		return
	case "livereplacement":
		s.replacementHandler.Stop()
	case "reconfiguration":
		s.reconfigHandler.Stop()
	case "areconfiguration":
		s.aReconfHandler.Stop()
	default:
		glog.Warning("unknown failure handling method specified, nothing to stop")
	}
}
