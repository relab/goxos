package server

import (
	"errors"

	"github.com/relab/goxos/client"
	"github.com/relab/goxos/config"
	"github.com/relab/goxos/net"
	"github.com/relab/goxos/paxos"
	"github.com/relab/goxos/reconfig"

	"github.com/relab/goxos/elog"
	e "github.com/relab/goxos/elog/event"

	"github.com/relab/goxos/Godeps/_workspace/src/github.com/golang/glog"
)

var (
	ErrRecvReconfCmdDuringReconf = errors.New("received reconfig command during reconfiguration")
)

func (s *Server) handleReconfigCmd(reconfCmd *paxos.ReconfigCmd) {
	elog.Log(e.NewEvent(e.ReconfigExecReconfCmd))
	defer s.reconfigHandler.SetReconfigInProgress(false)
	glog.V(2).Info("handling reconfiguration command")

	firstSlot := s.localAru.Value() + paxos.SlotID(s.config.GetInt("alpha", config.DefAlpha))
	glog.V(2).Info("first slot in new configuration is", firstSlot)

	glog.V(2).Info("connecting to new node")
	newNode := reconfCmd.Nodes[reconfCmd.New]
	newNodeConn, err := net.GxConnectTo(newNode, s.id, reconfCmd.New, s.dmx)
	if err != nil {
		glog.Errorln("reconfiguration error:", err)
		return
	}

	glog.V(2).Info("sending first slot to new node")
	fs := reconfig.FirstSlot{Slot: firstSlot}
	if err = newNodeConn.Write(fs); err != nil {
		glog.Errorln("reconfiguration error:", err)
		return
	}

	if s.id == s.pxLeader {
		err = s.handlePreReconfigAsLeader(firstSlot)
	} else {
		err = s.handlePreReconfigAsNonLeader(firstSlot)
	}
	if err != nil {
		glog.Errorln("reconfinguration error:", err)
		return
	}

	glog.V(2).Info("reached last slot, reconfiguring")

	// Are we beeing excluded?
	if reconfCmd.New == s.id {
		glog.Warning("we are being excluded, stopping...")
		go s.Stop()
		return
	}

	glog.V(2).Info("holding sub-modules")
	relaseChan := make(chan bool)
	s.grpmgr.RequestHold(relaseChan)

	glog.V(2).Info("updating node map")
	s.grpmgr.SetNewNodeMap(reconfCmd.Nodes)

	glog.V(2).Info("sending join to new node")
	if err = newNodeConn.Write(reconfig.Join{}); err != nil {
		glog.Errorln("reconfiguration error:", err)
		return
	}

	glog.V(2).Info("adding new connection to connections map")
	if err = net.AddToConnections(newNodeConn, s.grpmgr.LrEnabled()); err != nil {
		glog.Errorln("reconfiguration error:", err)
		return
	}

	glog.V(2).Info("releasing sub-modules")
	close(relaseChan)

	glog.V(2).Info("setting reconfig in progress false")
	s.reconfigHandler.SignalReconfCompleted()
	glog.V(2).Info("resuming normal operation")

	glog.V(2).Info("sending decided to proposer")
	for i := 0; i < int(s.config.GetInt("alpha", config.DefAlpha)); i++ {
		s.propDcdChan <- true
	}
}

// TODO(tormod):
// Handle leader change for the pre-reconfiguration phase below
// Handle stop request for the pre-reconfiguration phase below
func (s *Server) handlePreReconfigAsLeader(firstSlot paxos.SlotID) error {
	cmdsToFill := int(s.config.GetInt("alpha", config.DefAlpha) - 1)
	nrProposedToFill := 0

	glog.V(2).Infof("we are paxos leader, waiting for %v client cmds"+
		"for proposal and for them to be decided", cmdsToFill)

	var reqChan <-chan *client.Request
	decided := 0

	for {
		// If we must fill more slots, set the reqChan to the client
		// request channel. If we have proposed for the number of slots
		// needed before first slot, set the request chan to nil so we
		// don't propose any more client requests and only listen for
		// decided slots.
		if nrProposedToFill < cmdsToFill {
			reqChan = s.clientReqChan
		} else {
			reqChan = nil
		}

		select {
		case val := <-s.decidedChan:
			if val.Vt == paxos.Reconfig {
				return ErrRecvReconfCmdDuringReconf
			}
			s.handleDecidedVal(val, false)
			decided++
			if decided == cmdsToFill {
				return nil
			}
			//if s.localAru.Value() == firstSlot-1 {
			//	return nil
			//}
		case req := <-reqChan:
			s.propChan <- &paxos.Value{Vt: paxos.App, Cr: []*client.Request{req}}
			nrProposedToFill++
			glog.V(2).Infof("have proposed %d of %d needed cmds before first slot",
				nrProposedToFill, cmdsToFill)
		default:
			if nrProposedToFill >= cmdsToFill {
				continue
			}
			glog.V(2).Info("no client request in queue, proposing no-op")
			s.propChan <- &paxos.Value{Vt: paxos.Noop}
			nrProposedToFill++
		}
	}
}

// TODO(tormod):
// Handle leader change for the pre-reconfiguration phase below
// Handle stop request for the pre-reconfiguration phase below
func (s *Server) handlePreReconfigAsNonLeader(firstSlot paxos.SlotID) error {
	glog.V(2).Infof("waiting for last slot (%v) in current configuration", firstSlot-1)
	cmdsToFill := int(s.config.GetInt("alpha", config.DefAlpha) - 1)
	decided := 0
	for {
		val := <-s.decidedChan

		if val.Vt == paxos.Reconfig {
			return ErrRecvReconfCmdDuringReconf
		}

		s.handleDecidedVal(val, false)
		decided++
		if decided == cmdsToFill {
			return nil
		}
	}
}
