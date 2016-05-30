package server

import (
	"strings"
	"time"

	"github.com/relab/goxos/app"
	"github.com/relab/goxos/client"
	"github.com/relab/goxos/config"
	"github.com/relab/goxos/grp"
	"github.com/relab/goxos/paxos"

	"github.com/relab/goxos/elog"
	e "github.com/relab/goxos/elog/event"

	"github.com/relab/goxos/Godeps/_workspace/src/github.com/golang/glog"
)

func (s *Server) run() {
	elog.Log(e.NewEvent(e.Running))

	tschan := make(chan bool)
	s.startLogThroughput(tschan)
	defer s.stopLogThroughput(tschan)

	protocol := s.config.GetString("protocol", config.DefProtocol)
	switch strings.TrimSpace(strings.ToLower(protocol)) {
	case "authenticatedbc", "reliablebc":
		s.pxLeader = grp.UndefinedID()
	default:
		s.pxLeader = s.ld.PaxosLeader()
	}

	for {
		select {
		case pxLeaderID := <-s.pxLeaderChan:
			s.pxLeader = pxLeaderID
		case req := <-s.clientReqChan:
			// Shortcut if batching turned off:
			if s.batchMaxSize == 1 {
				s.propChan <- &paxos.Value{Vt: paxos.App, Cr: []*client.Request{req}}
				continue
			}

			s.appendToBatch(req)
			if s.batchNextIndex == s.batchMaxSize {
				s.sendBatch()
			}
		case <-s.batchTimer.C:
			s.sendBatch()
		case reconfigCmd := <-s.reconfigCmdChan:
			s.propChan <- &paxos.Value{Vt: paxos.Reconfig, Rc: &reconfigCmd}
		case val := <-s.decidedChan:
			s.handleDecidedVal(val, true)
		case asreq := <-s.appStateReqChan:
			s.handleAppStateReq(asreq)
		case <-s.stopChan:
			s.stop()
			return
		}
	}
}

func (s *Server) appendToBatch(req *client.Request) {
	if s.batchNextIndex == 0 {
		s.batchBuffer = make([]*client.Request, s.batchMaxSize)
		s.batchTimer.Reset(s.batchTimeout)
	}
	s.batchBuffer[s.batchNextIndex] = req
	s.batchNextIndex++
}

func (s *Server) sendBatch() {
	s.batchTimer.Stop()

	if s.batchNextIndex == 0 {
		return
	}

	s.propChan <- &paxos.Value{Vt: paxos.App, Cr: s.batchBuffer[0:s.batchNextIndex]}
	s.batchNextIndex = 0
}

func (s *Server) handleDecidedVal(val *paxos.Value, informProp bool) {
	if glog.V(3) {
		glog.Infof("executing decided value of type %v from learner", val.Vt)
	}

	switch val.Vt {
	case paxos.Noop:
		s.localAru.Increment()
		if informProp {
			s.propDcdChan <- true
		}
	case paxos.App:
		for i := range val.Cr {
			appresp := s.ah.Execute(val.Cr[i].GetVal())
			if glog.V(3) {
				glog.Info("application generated response")
			}
			cresp := genRespForReq(val.Cr[i], appresp)
			s.clientHandler.ForwardResponse(cresp)
			s.localAru.Increment()
		}
		if informProp {
			s.propDcdChan <- true
		}
	case paxos.Reconfig:
		// A node joining as a new node after a reconfiguration will do
		// catch-up to get the slots decided in the time between it got
		// its state and it joined the cluster. But, it should not execute
		// the reconfiguration command that included it. Therefore a
		// node only execute a reconfiguration command if
		// the slot id is larger or equal to its firstSlot.
		//s.localAru.Increment()
		if s.localAru.Value() >= s.firstSlot-1 {
			s.handleReconfigCmd(val.Rc)
			elog.Log(e.NewEvent(e.ReconfigDone))
		}
		s.localAru.Increment()
	default:
		glog.Fatal("received decided value of unkown type")
	}

	if glog.V(3) {
		glog.Infoln("localaru incremented to", s.localAru)
	}
}

func genRespForReq(req *client.Request, appresp []byte) *client.Response {
	var resp client.Response
	id := req.GetId()
	seq := req.GetSeq()
	resp.Type = client.Response_EXEC_RESP.Enum()
	resp.Id = &id
	resp.Seq = &seq
	resp.Val = appresp
	return &resp
}

func (s *Server) handleAppStateReq(asreq app.StateReq) {
	glog.V(2).Infoln("requesting state from application",
		"with slot marker:", s.localAru)
	slotMarker, state := s.ah.GetState(uint(s.localAru.Value()))
	glog.V(2).Infoln("received state from application,",
		"size was", len(state), "bytes and slot marker", slotMarker)
	appState := app.NewState(paxos.SlotID(slotMarker), state)
	asreq.RespChan() <- appState
}

func (s *Server) startLogThroughput(stop <-chan bool) {
	interval := s.config.GetDuration("throughputSamplingInterval", config.DefThroughputSamplingInterval)
	if interval == 0 {
		return
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		var currentAdu paxos.SlotID
		prevAdu := s.localAru.Value()
		for {
			select {
			case <-ticker.C:
				currentAdu = s.localAru.Value()
				elog.Log(
					e.NewEventWithMetric(
						e.ThroughputSample,
						uint64(currentAdu-prevAdu),
					),
				)
				prevAdu = currentAdu
			case <-stop:
				return
			}
		}
	}()
}

func (s *Server) stopLogThroughput(stop chan<- bool) {
	if s.config.GetDuration("throughputSamplingInterval", config.DefThroughputSamplingInterval) == 0 {
		return
	}
	stop <- true
}
