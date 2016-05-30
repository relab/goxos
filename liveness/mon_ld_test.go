package liveness

import (
	"testing"

	"github.com/relab/goxos/grp"
)

var (
	mockNode = grp.NewNode("", "", "", true, true, true)

	nodeMapForPaxosRank = map[grp.ID]grp.Node{
		grp.NewIDFromInt(0, 0): mockNode,
		grp.NewIDFromInt(1, 0): mockNode,
		grp.NewIDFromInt(2, 0): mockNode,
		grp.NewIDFromInt(3, 0): mockNode,
		grp.NewIDFromInt(4, 0): mockNode,
	}

	nodeMapForLrRank = map[grp.ID]grp.Node{
		grp.NewIDFromInt(0, 0): mockNode,
		grp.NewIDFromInt(1, 0): mockNode,
		grp.NewIDFromInt(2, 0): mockNode,
		grp.NewIDFromInt(3, 0): mockNode,
		grp.NewIDFromInt(4, 0): mockNode,
	}

	id          = grp.NewIDFromInt(0, 0)
	nrOfNodes   = len(nodeMapForPaxosRank)
	nrOfNodesLr = len(nodeMapForLrRank)

	nmPaxos       = grp.NewNodeMap(nodeMapForPaxosRank)
	nmPaxosWithLr = grp.NewNodeMap(nodeMapForLrRank)
)

func TestMaxRank(t *testing.T) {
	grpmgr := grp.NewGrpMgr(id, nmPaxos, false, false, nil)
	ld := NewMonarchicalLD(grpmgr, nil, nil)

	actualID, expectedID := ld.maxRank(), grp.NewIDFromInt(4, 0)
	if actualID != expectedID {
		t.Errorf("maxRank: want %v, got %v", expectedID, actualID)
	}

	// Suspect {4,0}
	ld.suspected[grp.NewIDFromInt(4, 0)] = true

	actualID, expectedID = ld.maxRank(), grp.NewIDFromInt(3, 0)
	if actualID != expectedID {
		t.Errorf("maxRank: want %v, got %v", expectedID, actualID)
	}
}
