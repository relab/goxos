package parallelpaxos

import (
	"errors"

	"github.com/relab/goxos/grp"
	px "github.com/relab/goxos/paxos"
)

type ProposerProcessor struct {
	numProcessors uint
	proposerID    grp.ID
	bcast         chan<- interface{}
	reqQueue      <-chan *px.Value
	id            uint
	slots         map[uint]*ProposerSlot
	adu           uint
	next          uint
}

type ProposerSlot struct {
	Decided  bool
	Proposal *px.Accept
}

// Returns pointer to a slot. Initializes the slot if
// necessary. Returns error if it is not our slot.
func (pp *ProposerProcessor) getSlot(slotID uint) (*ProposerSlot, error) {
	if slotID%pp.numProcessors != pp.id {
		return nil, errors.New("not our slot")
	}

	_, ok := pp.slots[slotID]
	if !ok {
		pp.slots[slotID] = &ProposerSlot{
			Decided: false,
		}
	}
	return pp.slots[slotID], nil
}

func (pp *ProposerProcessor) ProcessPromises(msg *PromisesMsg) {
	for _, promise := range msg.promises {
		if promise != nil {
			for _, accSlot := range promise.AccSlots {
				if accSlot.ID <= px.SlotID(pp.adu) {
					continue
				}

				slot, err := pp.getSlot(uint(accSlot.ID))

				if err != nil {
					// Not our slot
					continue
				}

				if slot.Proposal == nil || slot.Proposal.Rnd.Compare(accSlot.VRnd) >= 0 {
					slot.Proposal = &px.Accept{
						Rnd: accSlot.VRnd,
						Val: accSlot.VVal,
					}
				}
			}
		}
	}

	msg.wg.Done()
}

// Set the slot to decided and calculate new adu
func (pp *ProposerProcessor) HandleDecided(slotID uint, advanceCh chan<- *AduMsg) {
	slot, err := pp.getSlot(slotID)

	if err != nil {
		return
	}

	slot.Decided = true

	// We have a edge case. If this is the first desicion and our
	// processor id is not 0, then our initial pp.adu may not be
	// our slot. We just handle this edge case for itself here:
	if slotID == pp.id {
		pp.adu = slotID
		msg := &AduMsg{
			processor: pp.id,
			adu:       pp.adu,
		}
		go func(ch chan<- *AduMsg, m *AduMsg) {
			ch <- msg
		}(advanceCh, msg)
		return
	}

	if slotID == (pp.adu + pp.numProcessors) {
		didAdvance := false

		for i := pp.adu + pp.numProcessors; i <= slotID; i += pp.numProcessors {
			slot, _ := pp.getSlot(slotID)

			if slot.Decided {
				pp.adu = slotID
				didAdvance = true
			} else {
				break
			}
		}

		if didAdvance {
			msg := &AduMsg{
				processor: pp.id,
				adu:       pp.adu,
			}
			go func(ch chan<- *AduMsg, m *AduMsg) {
				ch <- msg
			}(advanceCh, msg)
		}
	}
}

// Sends accepts for slots up to given slot id. Assumes that there are
// enough elements in the queue. Panics otherwise.
func (pp *ProposerProcessor) SendAccepts(msg *SendAcceptsMsg) {
	for pp.next <= msg.upTo {
		select {
		case queuedVal := <-pp.reqQueue:
			accept := &px.Accept{
				ID:   pp.proposerID,
				Rnd:  msg.crnd,
				Slot: px.SlotID(pp.next),
				Val:  *queuedVal,
			}
			pp.bcast <- accept
			pp.next += pp.numProcessors
		default:
			panic("Proposer processor was commanded to send accept, but queue was empty")
		}
	}
}
