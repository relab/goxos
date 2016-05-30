package grp

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
)

var (
	ErrNodeAlreadyPresent         = errors.New("node with given id is already present in NodeMap")
	ErrOldNodeNotFound            = errors.New("node with given id not already present in Nodemap")
	ErrEqualPaxosIDAlreadyPresent = errors.New("node with given paxos id is already present in NodeMap")
	ErrNodeIDOutOfBounds          = errors.New("node with given id is out of bounds for current cluster size")
)

const (
	MinPaxosID = PaxosID(-1)
	MinEpoch   = Epoch(0)
	MaxPaxosID = PaxosID(math.MaxInt8)
	MaxEpoch   = Epoch(math.MaxUint64)
)

type ID struct {
	PaxosID PaxosID
	Epoch   Epoch
}

type PaxosID int8
type Epoch uint64

func NewID(id PaxosID, epoch Epoch) ID {
	return ID{id, epoch}
}

func NewIDFromInt(paxosID int8, epoch uint64) ID {
	return ID{
		PaxosID(paxosID),
		Epoch(epoch),
	}
}

func NewPxIDFromInt(paxosID int8) ID {
	return ID{PaxosID: PaxosID(paxosID)}
}

var minID = ID{MinPaxosID, MinEpoch}
var maxID = ID{MaxPaxosID, MinEpoch}
var undefinedID = NewIDFromInt(-1, 0)

func MinID() ID {
	return minID
}

func MaxID() ID {
	return maxID
}

func UndefinedID() ID {
	return undefinedID
}

func (id ID) CompareTo(otherID ID) int {
	if id.PaxosID > otherID.PaxosID {
		return 1
	} else if id.PaxosID < otherID.PaxosID {
		return -1
	} else {
		if id.Epoch > otherID.Epoch {
			return 1
		} else if id.Epoch < otherID.Epoch {
			return -1
		}
	}

	return 0
}

func (id ID) PxInt() int {
	return int(id.PaxosID)
}

func (id ID) String() string {
	return strconv.Itoa(int(id.PaxosID)) + "-" + strconv.Itoa(int(id.Epoch))
}

type Node struct {
	IP         string
	PaxosPort  string
	ClientPort string
	Proposer   bool
	Acceptor   bool
	Learner    bool
}

func NewNode(ip, pp, cp string, prop, acc, lrn bool) Node {
	return Node{ip, pp, cp, prop, acc, lrn}
}

func (n Node) PaxosAddr() string {
	return n.IP + ":" + n.PaxosPort
}

func (n Node) ClientAddr() string {
	return n.IP + ":" + n.ClientPort
}

func (n Node) ActorString() string {
	var actors []string
	if n.Proposer {
		actors = append(actors, "P")
	}
	if n.Acceptor {
		actors = append(actors, "A")
	}
	if n.Learner {
		actors = append(actors, "L")
	}
	return strings.Join(actors, ":")
}

func (n Node) String() string {
	return fmt.Sprintf("node (%v) with address: %v:%v/%v",
		n.ActorString(), n.IP, n.PaxosPort, n.ClientPort)
}

func (n Node) IsSame(m Node) bool {
	return m.IP == n.IP && m.PaxosPort == n.PaxosPort
}

type NodeMap struct {
	nodes               map[ID]Node
	epochs              []Epoch
	nrOfNodes           uint
	nrOfAcceptors       uint
	quorum              uint
	idsMemoized         []ID
	proposerIdsMemoized []ID
	acceptorIdsMemoized []ID
	learnerIdsMemoized  []ID
}

func NewNodeMap(nodes map[ID]Node) *NodeMap {
	var nrOfNodes, nrOfAcc uint
	for _, node := range nodes {
		if node.Acceptor {
			nrOfAcc++
		}
		nrOfNodes++
	}

	nm := NodeMap{
		nodes:  nodes,
		epochs: make([]Epoch, nrOfNodes),
	}

	for id := range nodes {
		nm.epochs[id.PaxosID] = id.Epoch
	}

	nm.nrOfNodes = nrOfNodes
	nm.nrOfAcceptors = nrOfAcc
	nm.quorum = nrOfAcc/2 + 1

	nm.memoize()

	return &nm
}

func NewNodeMapWithCount(nodes map[ID]Node, nrOfNodes, nrOfAcceptors uint) *NodeMap {
	nm := NodeMap{
		nodes:  nodes,
		epochs: make([]Epoch, nrOfNodes),
	}

	for id := range nodes {
		nm.epochs[id.PaxosID] = id.Epoch
	}

	nm.nrOfNodes = nrOfNodes
	nm.nrOfAcceptors = nrOfAcceptors
	nm.quorum = nrOfAcceptors/2 + 1

	nm.memoize()

	return &nm
}

func NewNodeMapFromSingleNode(id ID, node Node, nrOfNodes, nrOfAcceptors uint) *NodeMap {
	nodes := make(map[ID]Node)
	nodes[id] = node
	return NewNodeMapWithCount(nodes, nrOfNodes, nrOfAcceptors)
}

func (nm *NodeMap) memoize() {
	nm.memoizeIDs()
	nm.memoizeActorIDs()
}

func (nm *NodeMap) memoizeIDs() {
	ids := make([]ID, len(nm.nodes))
	i := 0
	for id := range nm.nodes {
		ids[i] = id
		i++
	}
	nm.idsMemoized = ids
}

func (nm *NodeMap) memoizeActorIDs() {
	var pids, aids, lids []ID
	for id, n := range nm.nodes {
		if n.Proposer {
			pids = append(pids, id)
		}
		if n.Acceptor {
			aids = append(aids, id)
		}
		if n.Learner {
			lids = append(lids, id)
		}
	}
	nm.proposerIdsMemoized = pids
	nm.acceptorIdsMemoized = aids
	nm.learnerIdsMemoized = lids
}

func (nm *NodeMap) CloneMap() map[ID]Node {
	clonedMap := make(map[ID]Node)
	for id, node := range nm.nodes {
		clonedMap[id] = node
	}

	return clonedMap
}

func (nm *NodeMap) IDs() []ID {
	return nm.idsMemoized
}

func (nm *NodeMap) ProposerIDs() []ID {
	return nm.proposerIdsMemoized
}

func (nm *NodeMap) AcceptorIDs() []ID {
	return nm.acceptorIdsMemoized
}

func (nm *NodeMap) LearnerIDs() []ID {
	return nm.learnerIdsMemoized
}

func (nm *NodeMap) NrOfNodes() uint {
	return nm.nrOfNodes
}

func (nm *NodeMap) NrOfAcceptors() uint {
	return nm.nrOfAcceptors
}

func (nm *NodeMap) Quorum() uint {
	return nm.quorum
}

func (nm *NodeMap) LookupNode(id ID) (Node, bool) {
	n, found := nm.nodes[id]
	return n, found
}

func (nm *NodeMap) LookupNodeWithPaxosID(pxID PaxosID) (ID, Node, bool) {
	for id, node := range nm.nodes {
		if id.PaxosID == pxID {
			return id, node, true
		}
	}

	return ID{}, Node{}, false
}

func (nm *NodeMap) IsNew(id ID, n Node) bool {
	oldid, on, found := nm.LookupNodeWithPaxosID(id.PaxosID)
	if !found {
		return true
	}
	if id.CompareTo(oldid) < 1 {
		return false
	}
	return !on.IsSame(n)
}

func (nm *NodeMap) Len() uint {
	return uint(len(nm.nodes))
}

func (nm *NodeMap) Add(id ID, node Node) error {
	if id.PaxosID < 0 || int(id.PaxosID) > int(nm.nrOfNodes-1) {
		return ErrNodeIDOutOfBounds
	}

	if _, found := nm.nodes[id]; found {
		return ErrNodeAlreadyPresent
	}

	if _, _, found := nm.LookupNodeWithPaxosID(id.PaxosID); found {
		return ErrEqualPaxosIDAlreadyPresent
	}

	nm.nodes[id] = node
	nm.epochs[id.PaxosID] = id.Epoch
	nm.memoize()

	return nil
}

func (nm *NodeMap) Replace(newID ID, newNode Node) error {
	oldID, _, found := nm.LookupNodeWithPaxosID(newID.PaxosID)
	if !found {
		return ErrOldNodeNotFound
	}

	delete(nm.nodes, oldID)
	nm.nodes[newID] = newNode
	nm.epochs[newID.PaxosID] = newID.Epoch
	nm.memoize()

	return nil
}

func (nm *NodeMap) Epochs() []Epoch {
	return nm.epochs
}

func (nm *NodeMap) Nodes() []Node {
	var nodes []Node
	for _, node := range nm.nodes {
		nodes = append(nodes, node)
	}
	return nodes
}

func EpochSlicesEqual(a, b []Epoch) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

func (nm *NodeMap) GetNext(id ID) ID {
	intid := uint(id.PaxosID)
	intid++
	intid %= nm.nrOfNodes
	return NewID(PaxosID(intid), nm.epochs[intid])
}
