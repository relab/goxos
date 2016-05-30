package grp

import (
	"testing"
)

var (
	id1        = NewID(0, 0)
	id2        = NewID(0, 1)
	id3        = NewID(1, 1)
	id4        = NewID(1, 0)
	id5        = NewID(2, 0)
	id6        = NewID(3, 0)
	dummyNode1 = NewNode("127.0.0.1", "8080", "8081", true, true, true)
	dummyNode2 = NewNode("127.0.0.1", "8080", "8081", true, false, false)
	dummyNode3 = NewNode("127.0.0.1", "8082", "8083", true, true, true)
)

func GenerateBasicMap() (map[ID]Node, uint, uint) {
	m := map[ID]Node{
		id1: dummyNode1,
		id4: dummyNode1,
		id5: dummyNode1,
	}

	return m, uint(len(m)), 3
}

func GenerateBasicMapWithDifferentActors() (map[ID]Node, uint, uint) {
	m := map[ID]Node{
		id1: dummyNode1,
		id4: dummyNode1,
		id5: dummyNode2,
	}

	return m, uint(len(m)), 2
}

func GenerateIncompleteMap() (map[ID]Node, uint, uint) {
	m := map[ID]Node{
		id1: dummyNode1,
		id4: dummyNode1,
	}

	return m, 3, 3
}

var idCompareToTests = []struct {
	a, b     ID
	expected int
}{
	{id1, id2, -1},
	{id1, id1, 0},
	{id2, id1, 1},
	{id3, id2, 1},
	{id3, id1, 1},
	{id1, id3, -1},
}

func TestIdCompareTo(t *testing.T) {
	var actual int
	for i, ictt := range idCompareToTests {
		actual = ictt.a.CompareTo(ictt.b)
		if actual != ictt.expected {
			t.Errorf("%d. %d != %d", i, actual, ictt.expected)
		}
	}
}

var idStringTests = []struct {
	id       ID
	expected string
}{
	{id1, "0-0"},
	{id2, "0-1"},
	{id3, "1-1"},
}

func TestIdString(t *testing.T) {
	var actual string
	for i, tist := range idStringTests {
		actual = tist.id.String()
		if actual != tist.expected {
			t.Errorf("%d. %q != %q", i, actual, tist.expected)
		}
	}
}

func TestBasicNodeMapAndIdsLength(t *testing.T) {
	nm, nrOfNodes, nrOfAcc := GenerateBasicMap()
	nodeMap := NewNodeMap(nm)
	testNodeMapLength(t, nodeMap, nrOfNodes, nrOfNodes, nrOfAcc, nrOfNodes)
}

func TestNodeMapAdd(t *testing.T) {
	nm, nrOfNodes, nrOfAcc := GenerateIncompleteMap()

	nodeMap := NewNodeMapWithCount(nm, nrOfNodes, nrOfAcc)
	initial := nodeMap.Len()

	err := nodeMap.Add(id5, dummyNode1)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	testNodeMapLength(t, nodeMap, initial+1, initial+1, initial+1, initial+1)
}

func TestNodeMapAddExsistingId(t *testing.T) {
	nm, _, _ := GenerateBasicMap()
	nodeMap := NewNodeMap(nm)
	initial := nodeMap.Len()

	err := nodeMap.Add(id1, dummyNode1)
	if err != ErrNodeAlreadyPresent {
		t.Errorf("Expected %v", ErrNodeAlreadyPresent)
	}

	testNodeMapLength(t, nodeMap, initial, initial, initial, initial)
}

func TestNodeMapAddExsistingPaxosId(t *testing.T) {
	nm, _, _ := GenerateBasicMap()
	nodeMap := NewNodeMap(nm)
	initial := nodeMap.Len()

	err := nodeMap.Add(id3, dummyNode1)
	if err != ErrEqualPaxosIDAlreadyPresent {
		t.Errorf("Expected %v", ErrEqualPaxosIDAlreadyPresent)
	}

	testNodeMapLength(t, nodeMap, initial, initial, initial, initial)
}

func TestNodeMapAndIdsLenghtWithDifferentActorTypes(t *testing.T) {
	nm, nrOfNodes, nrOfAcc := GenerateBasicMapWithDifferentActors()
	nodeMap := NewNodeMap(nm)
	testNodeMapLength(t, nodeMap, nrOfNodes, nrOfNodes, nrOfAcc, nrOfNodes-1)
}

func TestNodeMapReplace(t *testing.T) {
	nm, _, _ := GenerateBasicMap()
	nodeMap := NewNodeMap(nm)
	initial := nodeMap.Len()

	err := nodeMap.Replace(id2, dummyNode1)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	testNodeMapLength(t, nodeMap, initial, initial, initial, initial)
}

func TestNodeMapReplaceNonExisting(t *testing.T) {
	nm, _, _ := GenerateBasicMap()
	nodeMap := NewNodeMap(nm)
	initial := nodeMap.Len()

	err := nodeMap.Replace(id6, dummyNode1)
	if err != ErrOldNodeNotFound {
		t.Errorf("Expected %v", ErrOldNodeNotFound)
	}

	testNodeMapLength(t, nodeMap, initial, initial, initial, initial)
}

func testNodeMapLength(t *testing.T, nodeMap *NodeMap, nmlen, plen, alen, llen uint) {
	if nodeMap.Len() != nmlen {
		t.Errorf("%d != %d", nodeMap.Len(), nmlen)
	}

	if uint(len(nodeMap.IDs())) != nmlen {
		t.Errorf("%d != %d", len(nodeMap.IDs()), nmlen)
	}

	if uint(len(nodeMap.ProposerIDs())) != plen {
		t.Errorf("%d != %d", len(nodeMap.ProposerIDs()), plen)
	}

	if uint(len(nodeMap.AcceptorIDs())) != alen {
		t.Errorf("%d != %d", len(nodeMap.AcceptorIDs()), alen)
	}

	if uint(len(nodeMap.LearnerIDs())) != llen {
		t.Errorf("%d != %d", len(nodeMap.LearnerIDs()), llen)
	}
}

var (
	epa = []Epoch{Epoch(0), Epoch(1), Epoch(2)}
	epb = []Epoch{Epoch(1), Epoch(1), Epoch(2)}
	epc = []Epoch{Epoch(1), Epoch(1)}
)

var epochSlicesEqualTests = []struct {
	a, b     []Epoch
	expected bool
}{
	{epa, epa, true},
	{epa, epb, false},
	{epa, epc, false},
	{epb, epb, true},
	{epb, epc, false},
	{epc, epc, true},
}

func TestEpochSlicesEqual(t *testing.T) {
	var actual bool
	for i, esqt := range epochSlicesEqualTests {
		actual = EpochSlicesEqual(esqt.a, esqt.b)
		if actual != esqt.expected {
			t.Errorf("%d. %t != %t", i, actual, esqt.expected)
		}
	}
}

func TestIsNew(t *testing.T) {
	mp, _, _ := GenerateBasicMap()
	nm := NewNodeMap(mp)
	if nm.IsNew(id1, dummyNode3) {
		t.Errorf("%v is reported new for %v", id1, nm)
	}

	if nm.IsNew(id2, dummyNode1) {
		t.Errorf("%v is reported new for %v", dummyNode1, nm)
	}

	if !nm.IsNew(id2, dummyNode3) {
		t.Errorf("%v with id %v is not reported new in %v", dummyNode3, id2, nm)
	}
}

func TestGetNext(t *testing.T) {
	var result ID
	nm, _, _ := GenerateBasicMap()
	nodeMap := NewNodeMap(nm)
	result = nodeMap.GetNext(id4)
	if result != id5 {
		t.Error("getNext could not increment")
	}
	result = nodeMap.GetNext(id5)
	if result != id1 {
		t.Error("getNext could not wrap around")
	}
	err := nodeMap.Replace(id3, dummyNode2)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	result = nodeMap.GetNext(id1)
	if result != id3 {
		t.Errorf("getNext did not return correct id with higher Epoch")
	}
}
