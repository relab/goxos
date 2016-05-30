package net

import (
	"sync"
	"testing"
	"time"

	"github.com/relab/goxos/grp"
	"github.com/relab/goxos/liveness"
)

var (
	testingID        = grp.NewIDFromInt(0, 0)
	testingNode      = grp.NewNode("127.0.0.1", "8090", "8081", true, true, true)
	testingNrOfNodes = uint(3)
	testingNodeMap   = grp.NewNodeMapFromSingleNode(testingID, testingNode, testingNrOfNodes, testingNrOfNodes)
	testingGrpMgr    = grp.NewGrpMgr(testingID, testingNodeMap, false, false, nil)
	testingWg        = new(sync.WaitGroup)
)

// Deprecated test.
// Reason: glog.Fatal is called instead of panic. Method should be changed to return error.
/*
func TestPassingNonChannelParamToRegisterChannel(t *testing.T) {
	testingWg.Add(1)
	dmx := NewDemuxer(testingId, testingGrpMgr, testingWg)
	defer dmx.Stop()
	defer func() {
		if x := recover(); x == nil {
			t.Error("Expected a run time panic but got none")
		}
	}()
	hb := Heartbeat{}
	dmx.RegisterChannel(hb)
}
*/

// Deprecated test.
// Reason: Multiple registration is now allowed.
/*
func TestMultipleRegistrationOfSameChannelType(t *testing.T) {
	dmx = NewDemuxer("localhost:8090")
	defer dmx.Stop()
	defer func() {
		if x := recover(); x == nil {
			t.Error("Expected a run time panic but got none")
		}
	}()
	dmx.RegisterChannel(make(chan PrepareMsg))
	dmx.RegisterChannel(make(chan PrepareMsg))
}
*/

// Deprecated test.
// Reason: Start with no registration is currently needed for LR.
/*
func TestStartWithNoPriorRegistration(t *testing.T) {
	defer func() {
		if x := recover(); x == nil {
			t.Error("Expected a run time panic but got none")
		}
	}()
	dmx = NewDemuxer("localhost:8090")
	defer dmx.Stop()
	dmx.Start()
}
*/

func TestStartWithPriorRegistration(t *testing.T) {
	testingWg.Add(1)
	defer func() {
		if x := recover(); x != nil {
			t.Error("Run time panic: ", x)
		}
	}()
	dmx := NewTcpDemuxer(testingID, testingGrpMgr, testingWg)
	defer dmx.Stop()
	dmx.RegisterChannel(make(chan liveness.Heartbeat))
	dmx.Start()
}

func TestHandleMessage(t *testing.T) {
	testingWg.Add(1)
	dmx := NewTcpDemuxer(testingID, testingGrpMgr, testingWg)
	defer dmx.Stop()
	hbCh := make(chan liveness.Heartbeat)
	dmx.RegisterChannel(hbCh)
	dmx.Start()
	go func() {
		select {
		case <-hbCh:
			// Expected
		case <-time.After(500 * time.Millisecond):
			t.Error("Timeout: Did not received expected message on registered channel")

		}
	}()
	dmx.HandleMessage(liveness.Heartbeat{})
}
