package reconfig

import (
	"encoding/gob"

	"github.com/relab/goxos/paxos"
)

func init() {
	gob.Register(FirstSlot{})
	gob.Register(Join{})
}

// FirstSlot is a message indicating the first slot in a new configuration.
type FirstSlot struct {
	Slot paxos.SlotID
}

// Join is a message indicating that a standby replica started due to a
// reconfiguration should join the new configuration.
type Join struct{}
