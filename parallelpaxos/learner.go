package parallelpaxos

import (
	"sync"

	"github.com/relab/goxos/config"
	"github.com/relab/goxos/grp"
	"github.com/relab/goxos/net"
	px "github.com/relab/goxos/paxos"

	"github.com/relab/goxos/Godeps/_workspace/src/github.com/golang/glog"
)

type LearnerSlotMap struct {
	stopCheckIn       *sync.WaitGroup
	numProcessors     uint
	dcdChan           chan<- *px.Value
	dcdSlotIDToProp   chan<- px.SlotID
	id                int
	slots             map[uint]*LearnerSlot
	grpmgr            grp.GroupManager
	nextToDecideToken bool
	tokenChanNext     chan bool
	tokenChan         chan bool
	nextToDecide      uint
	learnChan         chan *px.Learn
	stop              chan bool
}

type LearnerSlot struct {
	ID         px.SlotID
	Rnd        px.ProposerRound
	Learns     []*px.Learn
	Votes      uint
	Learned    bool
	Decided    bool
	LearnedVal px.Value
}

func (lsm *LearnerSlotMap) getSlot(id uint) *LearnerSlot {
	_, ok := lsm.slots[id]
	if !ok {
		lsm.slots[id] = &LearnerSlot{ID: px.SlotID(id)}
	}
	return lsm.slots[id]
}

func (lsm *LearnerSlotMap) receiveNextToDecideToken() {
	lsm.nextToDecideToken = true

	// Do we have anything to decide?
	slot := lsm.getSlot(lsm.nextToDecide)
	if slot.Learned {
		lsm.dcdChan <- &slot.LearnedVal
		lsm.dcdSlotIDToProp <- slot.ID
		lsm.nextToDecide += lsm.numProcessors
		lsm.nextToDecideToken = false
		lsm.tokenChanNext <- true
	}
}

func (lsm *LearnerSlotMap) run() {
	glog.V(1).Info("starting one learner processor")

	lsm.stopCheckIn.Add(1)
	defer lsm.stopCheckIn.Done()

	for {
		select {
		case learn := <-lsm.learnChan:
			lsm.handleLearn(learn)
		case <-lsm.tokenChan:
			lsm.receiveNextToDecideToken()
		case <-lsm.stop:
			return
		}
	}
}

func (lsm *LearnerSlotMap) handleLearn(learn *px.Learn) {
	glog.V(3).Infof("got learn from %v for slot %d", learn.ID, learn.Slot)
	slot := lsm.getSlot(uint(learn.Slot))

	if slot.Learned {
		return
	}

	switch slot.Rnd.Compare(learn.Rnd) {
	case 1:
		// Round in learn is lower, ignore
		return
	case -1:
		// Round in learn is bigger, reset and initialize,
		// then fallthrough. This will always be true for
		// first learn in new slot.
		slot.Rnd = learn.Rnd
		slot.Learns = make([]*px.Learn, lsm.grpmgr.NrOfNodes())
		slot.Votes = 0
		fallthrough
	case 0:
		if slot.Learns[learn.ID.PxInt()] != nil {
			// If we already have a learn with some round
			// from the replica: ignore.
			return
		}
		slot.Votes++
		slot.Learns[learn.ID.PxInt()] = learn

		// Return if no quorum:
		if slot.Votes < lsm.grpmgr.Quorum() {
			return
		}
	}

	// We got a quorum. Continue to learn the value:
	glog.V(3).Infof("Learning slot %d", slot.ID)
	slot.Learned = true
	slot.LearnedVal = learn.Val

	if lsm.nextToDecideToken && uint(slot.ID) == lsm.nextToDecide {
		lsm.dcdChan <- &slot.LearnedVal
		lsm.dcdSlotIDToProp <- slot.ID
		lsm.nextToDecide += lsm.numProcessors
		lsm.nextToDecideToken = false
		lsm.tokenChanNext <- true
	} else if uint(slot.ID) > lsm.nextToDecide {
		// Could send catch up here
	}
}

type ParallelLearner struct {
	id              grp.ID
	started         bool
	startable       bool
	processors      []LearnerSlotMap
	numProcessors   int
	learnChan       <-chan px.Learn
	learnChans      []chan *px.Learn
	dcdChan         chan<- *px.Value
	dcdSlotIDToProp chan<- px.SlotID
	dmx             net.Demuxer
	grpmgr          grp.GroupManager
	stop            chan bool
	stopCheckIn     *sync.WaitGroup
}

// Create a new ParallelLearner with the help of a PaxosPack.
func NewParallelLearner(pp *px.Pack) *ParallelLearner {
	ma := &ParallelLearner{
		id:              pp.ID,
		started:         false,
		startable:       (pp.RunAcc && pp.RunLrn),
		dmx:             pp.Dmx,
		grpmgr:          pp.Gm,
		dcdChan:         pp.DcdChan,
		dcdSlotIDToProp: pp.DcdSlotIDToProp,
		stopCheckIn:     pp.StopCheckIn,
		numProcessors:   pp.Config.GetInt("parallelPaxosLearners", config.DefParallelPaxosLearners),
		stop:            make(chan bool),
	}

	ma.processors = make([]LearnerSlotMap, ma.numProcessors)
	tokenChans := make([]chan bool, ma.numProcessors)

	for i := 0; i < ma.numProcessors; i++ {
		tokenChans[i] = make(chan bool, 2)
	}

	for i := 0; i < ma.numProcessors; i++ {
		p := LearnerSlotMap{
			stopCheckIn:       ma.stopCheckIn,
			numProcessors:     uint(ma.numProcessors),
			dcdChan:           ma.dcdChan,
			dcdSlotIDToProp:   pp.DcdSlotIDToProp,
			id:                i,
			slots:             make(map[uint]*LearnerSlot),
			grpmgr:            ma.grpmgr,
			nextToDecideToken: false,
			learnChan:         make(chan *px.Learn, 10),
			tokenChanNext:     tokenChans[(i+1)%ma.numProcessors],
			tokenChan:         tokenChans[i],
			nextToDecide:      uint(i),
			stop:              make(chan bool, 1),
		}

		if i == 0 {
			p.nextToDecide = uint(ma.numProcessors)
		}

		ma.processors[i] = p
	}

	return ma
}

// Start listening for ParallelPaxos messages from other
// replicas. Registers interest for PrepareMsg and AcceptMsg.
func (l *ParallelLearner) Start() {
	if !l.startable || l.started {
		glog.Warning("ignoring start request")
		return
	}

	glog.V(1).Info("starting")
	l.started = true
	l.registerChannels()

	for i := range l.processors {
		go l.processors[i].run()
	}

	go func() {
		firstLearn := true

		for {
			select {
			case learn := <-l.learnChan:
				if firstLearn {
					l.processors[int(learn.Slot)%l.numProcessors].tokenChan <- true
					firstLearn = false
				}
				l.processors[int(learn.Slot)%l.numProcessors].learnChan <- &learn
			case <-l.stop:
				return
			}
		}
	}()
}

// Stops all Learner processors.
func (l *ParallelLearner) Stop() {
	if l.started {
		l.started = false

		l.stop <- true
		for i := range l.processors {
			l.processors[i].stop <- true
		}
		l.stopCheckIn.Done()
	}
}

func (l *ParallelLearner) registerChannels() {
	learnChan := make(chan px.Learn, 256)
	l.learnChan = learnChan
	l.dmx.RegisterChannel(learnChan)
}
