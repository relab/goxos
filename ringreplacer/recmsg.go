package ringreplacer

import (
	"encoding/gob"
	"fmt"

	"github.com/relab/goxos/grp"
	"github.com/relab/goxos/paxos"
)

func init() {
	gob.Register(ReconfMsg{})
}

type ReconfMsg struct {
	NodeMap map[grp.ID]grp.Node
	NewIds  []grp.ID
	AduSlot paxos.SlotID
}

func (rm ReconfMsg) String() string {
	return fmt.Sprintf("Reconf Msg with new nodes:"+
		"%v and Slot %v", rm.NewIds, rm.AduSlot)
}
