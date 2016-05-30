package arec

import (
	"encoding/gob"
	"fmt"

	"github.com/relab/goxos/grp"
	"github.com/relab/goxos/paxos"
)

func init() {
	gob.Register(ReconfMsg{})
	gob.Register(CPromise{})
	gob.Register(Activation{})
}

type ReconfMsg struct {
	NewEpoch grp.Epoch
	NodeMap  map[grp.ID]grp.Node
	OldIds   map[grp.ID]bool
	AduSlot  paxos.SlotID
}

type CPromise struct {
	ID         grp.ID
	QuorumSize uint
	Sender     grp.Node
	NewEpoch   grp.Epoch
	Confs      map[grp.Epoch]map[grp.ID]grp.Node
	AduSlot    paxos.SlotID
	AccState   paxos.AcceptorSlotMap
}

type Activation struct {
	Epoch    grp.Epoch
	AduSlot  paxos.SlotID
	AccState *paxos.AcceptorSlotMap
	Confs    map[grp.Epoch]map[grp.ID]grp.Node
}

func (rm ReconfMsg) String() string {
	return fmt.Sprintf("Reconf Msg with epoch %v, nodes:"+
		"%v and Slot %v", rm.NewEpoch, rm.NodeMap, rm.AduSlot)
}

func (cp CPromise) String() string {
	keys := make([]grp.Epoch, 0, len(cp.Confs))
	for k := range cp.Confs {
		keys = append(keys, k)
	}
	return fmt.Sprintf("CPromise from %v, including confs %v:", cp.ID, keys)
}

func (ac Activation) String() string {
	keys := make([]grp.Epoch, 0, len(ac.Confs))
	for k := range ac.Confs {
		keys = append(keys, k)
	}
	return fmt.Sprintf("Activation for epoch %v, including confs %v:", ac.Epoch, keys)
}
