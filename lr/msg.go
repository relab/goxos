package lr

import (
	"encoding/gob"
	"fmt"

	"github.com/relab/goxos/grp"
	"github.com/relab/goxos/paxos"
)

func init() {
	gob.Register(PrepareEpoch{})
	gob.Register(EpochPromise{})
	gob.Register(ForwardedEpSet{})
}

// Represents a PrepareEpoch message as defined in the Live Replacement protocol.
type PrepareEpoch struct {
	ReplacerID   grp.ID
	ReplacerNode grp.Node
	SlotMarker   paxos.SlotID
}

// Represents an EpochPromise message as defined in the Live Replacement protocol.
type EpochPromise struct {
	ID          grp.ID
	Sender      grp.Node
	EpochVector []grp.Epoch
	SlotMarker  paxos.SlotID
	AccState    paxos.AcceptorSlotMap
}

// ForwardEpSet encapsulates a set of EpochPromise messages.
type ForwardedEpSet struct {
	EpSet []*EpochPromise
}

// String returns a text representation of the PrepareEpoch message.
func (pe PrepareEpoch) String() string {
	return fmt.Sprintf("PrepareEpoch: replace id %v - new epoch %v, "+
		"new %v, slot marker is %v", pe.ReplacerID.PaxosID, pe.ReplacerID.Epoch,
		pe.ReplacerNode, pe.SlotMarker)
}

// String returns a text representation of the EpochPromise message.
func (ep EpochPromise) String() string {
	return fmt.Sprintf("EpochPromise from %v (%v), "+
		"epoch vector %v and slot marker %v", ep.ID,
		ep.Sender, ep.EpochVector, ep.SlotMarker)

}

// String returns a text representation of the ForwaredEpSet message.
func (fes ForwardedEpSet) String() string {
	return fmt.Sprintf("Forwared set of EpochPromises with size %d", len(fes.EpSet))
}
