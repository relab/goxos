package paxos

import (
	"sync"

	"github.com/relab/goxos/config"
	"github.com/relab/goxos/grp"
	"github.com/relab/goxos/liveness"
	"github.com/relab/goxos/net"
)

// A Pack is used during the construction of the Paxos actors. This Pack is
// passed to each of the actor constructors upon instantiation. The actors can then pull
// the information they need from this pack to function.
//
// For example, each actor will need to pull the Bcast and Ucast channels from this pack
// and store them locally so that they can send messages to the other replicas:
//
//    func NewMultiProposer(pp px.Pack) *MultiProposer {
//      return &MultiProposer{
//        ...
//        ucast:          pp.Ucast,
//        bcast:          pp.Bcast,
//        ...
//      }
//    }
//
type Pack struct {
	ID          grp.ID
	Gm          grp.GroupManager
	StopCheckIn *sync.WaitGroup

	Config        *config.Config
	NrOfNodes     uint
	NrOfAcceptors uint

	RunProp bool
	RunAcc  bool
	RunLrn  bool

	Dmx net.Demuxer
	Ld  liveness.LeaderDetector

	Ucast  chan<- net.Packet
	Bcast  chan<- interface{}
	BcastP chan<- interface{}
	BcastA chan<- interface{}
	BcastL chan<- interface{}

	PropChan        <-chan *Value // Proposal values
	DcdChan         chan<- *Value // Decided values
	NewDcdChan      chan bool
	DcdSlotIDToProp chan SlotID

	LocalAdu        *Adu
	FirstSlot       SlotID
	NextExpectedDcd SlotID
}
