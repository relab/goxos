package paxos

import (
	"github.com/relab/goxos/grp"
)

// For now only allowing replacement through reconfiguration. Nodes is
// the new NodeMap to be used. New is the id of the replaced node in the node
// map. Should be expanded to allow different types of reconfiguration
// commands: add, remove, replace etc.
type ReconfigCmd struct {
	Nodes map[grp.ID]grp.Node
	New   grp.ID
}
