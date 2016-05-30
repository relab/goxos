package batchpaxos

import (
	"sync"
	"time"

	"github.com/relab/goxos/client"
	"github.com/relab/goxos/config"
	"github.com/relab/goxos/grp"
	"github.com/relab/goxos/net"
	px "github.com/relab/goxos/paxos"

	"github.com/relab/goxos/Godeps/_workspace/src/github.com/golang/glog"
)

// The main structure storing all the state associated with a batch acceptor.
type BatchAcceptor struct {
	startable    bool
	id           grp.ID // Our id
	leader       grp.ID // Current leader
	batcher      *Batcher
	batchpter    *BatchPointer
	batchqc      *BatchQuorumChecker
	batchTimeout time.Duration    // Period of inactivity we wait before batching
	timeoutChan  <-chan time.Time // Timer channel, we receive a tick on this channel after inactivity
	batchAru     uint             // The batch id for which all batches less than this are executed
	batchMaxReq  uint             // Maximum number of requests that we can fit into a batch
	currNumReq   uint             // Current number of requests in the current batch
	ucast        chan<- net.Packet
	bcast        chan<- interface{}
	trust        <-chan grp.ID
	propChan     <-chan *px.Value
	blChan       <-chan BatchLearnMsg
	bcChan       <-chan BatchCommitMsg
	ureqChan     <-chan UpdateRequestMsg
	urepChan     <-chan UpdateReplyMsg
	dcdChan      chan<- *px.Value
	dmx          net.Demuxer
	grpmgr       grp.GroupManager
	stop         chan bool
	recvtime     map[uint32]time.Time
	exectime     map[uint32]time.Time
	stopCheckIn  *sync.WaitGroup
}

// Construct a new batch acceptor state, given the setup information required in
// the paxos pack.
func NewBatchAcceptor(pp px.Pack) *BatchAcceptor {
	if pp.Config.GetInt("batchMaxSize", config.DefBatchMaxSize) > 1 {
		panic("Cannot run BatchPaxos when regular batching is enabled. Set config `batchMaxSize` to 1.")
	}

	return &BatchAcceptor{
		id:           pp.ID,
		batchTimeout: pp.Config.GetDuration("batchPaxosTimeout", config.DefBatchPaxosTimeout),
		batchMaxReq:  uint(pp.Config.GetInt("batchPaxosMaxSize", config.DefBatchPaxosMaxSize)),
		startable:    pp.RunAcc,
		ucast:        pp.Ucast,
		bcast:        pp.Bcast,
		trust:        pp.Ld.SubscribeToPaxosLdMsgs("acceptor"),
		dmx:          pp.Dmx,
		grpmgr:       pp.Gm,
		dcdChan:      pp.DcdChan,
		propChan:     pp.PropChan,
		batcher:      NewBatcher(),
		batchpter:    NewBatchPointer(),
		batchqc:      NewBatchQuorumChecker(pp.NrOfAcceptors/2 + 1),
		stop:         make(chan bool),
		recvtime:     make(map[uint32]time.Time),
		exectime:     make(map[uint32]time.Time),
		stopCheckIn:  pp.StopCheckIn,
		leader:       pp.Ld.PaxosLeader(),
	}
}

// Spawn a new goroutine handling all of the channels from the demuxer.
func (ba *BatchAcceptor) Start() {
	glog.V(1).Infof("starting (timeout=%v, maxreq=%d)", ba.batchTimeout, ba.batchMaxReq)

	ba.registerChannels()
	ba.resetTimer()

	go func() {
		defer ba.stopCheckIn.Done()
		for {
			select {
			case val := <-ba.propChan:
				// Only can handle client requests
				if val.Vt != px.App {
					glog.Fatal("can't handle non client request type value")
				}
				ba.handleRequest(*val)
			case msg := <-ba.blChan:
				ba.handleLearnMsg(msg)
			case msg := <-ba.bcChan:
				ba.handleCommitMsg(msg)
			case msg := <-ba.ureqChan:
				ba.handleUpdateRequestMsg(msg)
			case msg := <-ba.urepChan:
				ba.handleUpdateReplyMsg(msg)
			case <-ba.timeoutChan:
				ba.handleBatchTimeout()
			case trustID := <-ba.trust:
				ba.leader = trustID
			case <-ba.stop:
				return
			}
		}
	}()
}

// Stop the goroutine, no future messages will be handled after the batch acceptor
// goroutine receives this message.
func (ba *BatchAcceptor) Stop() {
	ba.stop <- true
}

// Reset the timer -- which is usually done when a batch fills up or the timer
// expires due to inactivity.
func (ba *BatchAcceptor) resetTimer() {
	ba.timeoutChan = time.After(ba.batchTimeout)
}

// Register channels with the demuxer so that we can receive messages sent over
// the network.
func (ba *BatchAcceptor) registerChannels() {
	blChan := make(chan BatchLearnMsg, 128)
	ba.dmx.RegisterChannel(blChan)
	ba.blChan = blChan

	bcChan := make(chan BatchCommitMsg, 128)
	ba.dmx.RegisterChannel(bcChan)
	ba.bcChan = bcChan

	ureqChan := make(chan UpdateRequestMsg, 128)
	ba.dmx.RegisterChannel(ureqChan)
	ba.ureqChan = ureqChan

	urepChan := make(chan UpdateReplyMsg, 128)
	ba.dmx.RegisterChannel(urepChan)
	ba.urepChan = urepChan
}

// Handle a timer event -- called (event-based) when a timer expires or if we get
// enough requests to fill a batch (manually).
func (ba *BatchAcceptor) handleBatchTimeout() {
	if glog.V(3) {
		glog.Info("timeout -- batching")
	}

	batcher := ba.batcher
	batch := batcher.GetBatch()

	if batch != nil {
		if glog.V(4) {
			glog.Info("timeout -- batch not nil")
		}

		ba.send(BatchLearnMsg{
			ID:      ba.id,
			BatchID: batch.ID,
			HSS:     batch.HSS,
		}, ba.leader)
	} else {
		if glog.V(4) {
			glog.Info("timeout -- batch is nil")
		}
	}

	ba.resetTimer()
}

// Handle a request from a client. Must check whether or not the batch size is
// exceeding the specified limit.
func (ba *BatchAcceptor) handleRequest(val px.Value) error {
	if glog.V(3) {
		glog.Info("handling request")
	}

	req := val.Cr

	ba.recvtime[req[0].GetSeq()] = time.Now()

	batcher := ba.batcher
	batcher.LogRequest(*req[0])

	ba.currNumReq = (ba.currNumReq + 1) % ba.batchMaxReq

	// When wrap-around occurs, we've received enough commands to batch
	if ba.currNumReq == 0 {
		if glog.V(3) {
			glog.Info("maximum batch size -- batching")
		}

		batch := batcher.GetBatch()

		if batch != nil {
			if glog.V(4) {
				glog.Info("handlerequest -- batch not nil")
			}

			ba.send(BatchLearnMsg{
				ID:      ba.id,
				BatchID: batch.ID,
				HSS:     batch.HSS,
			}, ba.leader)
		} else {
			if glog.V(4) {
				glog.Info("handlerequest -- batch is nil")
			}
		}

		ba.resetTimer()
	}

	return nil
}

// Handle an update request -- part of the update protocol. This occurs when a commit
// message is received at a node and they don't yet have (at least one of) the ranges
// in the commit.
func (ba *BatchAcceptor) handleUpdateRequestMsg(msg UpdateRequestMsg) {
	if glog.V(3) {
		msg.Print()
	}

	br := ba.batcher
	inc := br.GetIncompleteRanges(msg.Ranges)

	if len(inc) == 0 {
		cmds := br.GetClientCommands(msg.Ranges)
		ba.send(UpdateReplyMsg{
			ID:      ba.id,
			BatchID: msg.BatchID,
			Ranges:  msg.Ranges,
			Values:  cmds,
		}, msg.ID)
	} else {
		if glog.V(4) {
			glog.Info("handleUpdateRequestMsg: did not have complete ranges")
		}
	}
}

// Handle an update reply -- part of the update protocol.
func (ba *BatchAcceptor) handleUpdateReplyMsg(msg UpdateReplyMsg) {
	if glog.V(3) {
		msg.Print()
	}

	br := ba.batcher
	bqc := ba.batchqc

	state := bqc.GetState(msg.BatchID)

	if state.Execute {
		return
	}
	br.SetClientCommands(msg.Values)
	inc := br.GetIncompleteRanges(state.ExecuteRanges)

	if len(inc) == 0 {
		state.Execute = true
		ba.tryToAdvance()
	} else {
		glog.Fatal("attempted update didn't bring us up to date")
	}
}

// Handle a learn message sent from one of the replicas to the leader. Quorum checking
// is then done. If an majority exists, then we intersect the ranges and send a commit
// to the quorum.
func (ba *BatchAcceptor) handleLearnMsg(msg BatchLearnMsg) {
	if glog.V(3) {
		msg.Print()
	}

	bp := ba.batchpter

	bqc := ba.batchqc
	bqc.AddLearn(msg)

	if bqc.LearnQuorumExists(msg.BatchID) {
		if glog.V(3) {
			glog.Infoln("quorum exists in batch", msg.BatchID)
		}

		intersect := bqc.Intersect(msg.BatchID)
		done, ranges := bp.TryBatchPoint(intersect)

		if glog.V(3) {
			if done {
				glog.Info("intersect advances to new batchpoint")
			} else {
				glog.Info("intersect does not advance batchpoint")
			}
		}

		ba.broadcast(BatchCommitMsg{
			ID:      ba.id,
			BatchID: msg.BatchID,
			Ranges:  ranges,
		})
	}
}

// Handle a commit message. Checking must be done to see whether we actually have the
// ranges contained in the commit. If not, the update protocol is initiated. Otherwise,
// start execution.
func (ba *BatchAcceptor) handleCommitMsg(msg BatchCommitMsg) {
	if glog.V(3) {
		msg.Print()
	}

	br := ba.batcher
	bqc := ba.batchqc

	state := bqc.GetState(msg.BatchID)
	state.ExecuteRanges = msg.Ranges

	inc := br.GetIncompleteRanges(msg.Ranges)

	if len(inc) == 0 {
		// We have all ranges, complete
		state.Execute = true
		ba.tryToAdvance()
	} else {
		// Need to use update protocol
		ba.broadcast(UpdateRequestMsg{
			ID:      ba.id,
			BatchID: msg.BatchID,
			Ranges:  inc,
		})
	}
}

// Execute a single request -- also record timings for the request.
func (ba *BatchAcceptor) execute(req client.Request) {
	if glog.V(3) {
		glog.Infof("executing command %v from %v", req.GetSeq(), req.GetId())
	}
	ba.dcdChan <- &px.Value{Vt: px.App, Cr: []*client.Request{&req}}
	if glog.V(3) {
		glog.Info("sent to server")
	}
	exectime := time.Now()
	recvtime := ba.recvtime[req.GetSeq()]
	ba.exectime[req.GetSeq()] = exectime
	if glog.V(3) {
		glog.Infof("time to execute for %d: %v", req.GetSeq(), exectime.Sub(recvtime))
	}
}

// Attempt to advance the batch ARU, usually called after receiving a commit
// message from the leader. This starts at the current ARU and executes batches
// until it reaches a batch id that cannot yet be executed.
func (ba *BatchAcceptor) tryToAdvance() {
	br := ba.batcher
	bqc := ba.batchqc

	for {
		state := bqc.GetState(ba.batchAru)

		if state.Execute && !state.Executed {
			if glog.V(3) {
				glog.Infoln("executing batch", ba.batchAru)
			}

			br.ForEach(state.ExecuteRanges, func(cr client.Request) {
				ba.execute(cr)
			})
			state.Executed = true
			ba.batchAru++

			if glog.V(3) {
				glog.Infoln("incrementing batch aru to", ba.batchAru)
			}
		} else {
			break
		}
	}
}

// Helper function to unicast to a single node.
func (ba *BatchAcceptor) send(msg interface{}, id grp.ID) {
	ba.ucast <- net.Packet{DestID: id, Data: msg}
}

// Helper function to broadcast to the whole RSM.
func (ba *BatchAcceptor) broadcast(msg interface{}) {
	ba.bcast <- msg
}

// Not used by BatchPaxos, but needed to fulfill actor interface
func (ba *BatchAcceptor) HandlePrepare(msg px.Prepare) error {
	return nil
}

// Not used by BatchPaxos, but needed to fulfill actor interface
func (ba *BatchAcceptor) HandleAccept(msg px.Accept) error {
	return nil
}

// Not used by BatchPaxos, but needed to fulfill actor interface
func (ba *BatchAcceptor) GetState(afterSlot px.SlotID, release <-chan bool) *px.AcceptorSlotMap {
	return nil
}

func (ba *BatchAcceptor) GetMaxSlot(release <-chan bool) *px.AcceptorSlotMap {
	return nil
}

func (ba *BatchAcceptor) SetLowSlot(slot px.SlotID) {
}

// Not used by BatchPaxos, but needed to fulfill actor interface
func (ba *BatchAcceptor) SetState(slotmap *px.AcceptorSlotMap) {
}
