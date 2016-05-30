package net

import (
	"errors"
	"testing"

	"github.com/relab/goxos/grp"
)

var (
	dmx = NewMockDemuxer()
	id1 = grp.NewIDFromInt(0, 0)
	id2 = grp.NewIDFromInt(1, 0)
	id3 = grp.NewIDFromInt(2, 0)
	id4 = grp.NewIDFromInt(1, 1)
	gc1 = NewGxConnection(nil, id1, dmx)
	gc2 = NewGxConnection(nil, id2, dmx)
)

// Add a GxConnection to the connection map.
func MockAddToConnections(gc *GxConnection, lrarEnabled bool) error {
	if !lrarEnabled {
		connections[gc.id.PaxosID] = gc
	} else {
		if existingConn, found := connections[gc.id.PaxosID]; !found {
			connections[gc.id.PaxosID] = gc
		} else {
			comparison := gc.id.CompareTo(existingConn.id)
			if comparison == 1 || comparison == 0 {
				connections[gc.id.PaxosID] = gc
			} else {
				return errors.New("id for connection is lower than already present")
			}
		}
	}

	return nil
}

func TestCheckConnections(t *testing.T) {
	MockAddToConnections(gc1, false)
	MockAddToConnections(gc2, false)
	conf1 := []grp.ID{id1, id3, id4}
	notin1 := CheckConnections(conf1)
	if len(notin1) != 2 {
		t.Errorf("Did not check correct conf1: %v", notin1)
	}
	conf2 := []grp.ID{id1, id2}
	notin2 := CheckConnections(conf2)
	if len(notin2) != 0 {
		t.Errorf("Did not check correct conf1: %v", notin2)
	}
}

func TestUpdateId(t *testing.T) {
	MockAddToConnections(gc1, false)
	MockAddToConnections(gc2, false)
	ok := UpdateConnID(id2, grp.Epoch(1))
	if !ok {
		t.Error("Did not update Epoch")
	}
	id := connections[id2.PaxosID].id
	if id != id4 {
		t.Errorf("Wrong Id after update: %v", id)
	}
	ok2 := UpdateConnID(id3, grp.Epoch(1))
	if ok2 {
		t.Error("Update id not present")
	}
	ok3 := UpdateConnID(id2, grp.Epoch(2))
	if ok3 {
		t.Error("Update id not present")
	}
}
