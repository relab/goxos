package server

import (
	"sync"
	"time"

	"github.com/relab/goxos/app"
	"github.com/relab/goxos/arec"
	"github.com/relab/goxos/client"
	"github.com/relab/goxos/config"
	"github.com/relab/goxos/grp"
	"github.com/relab/goxos/liveness"
	"github.com/relab/goxos/lr"
	"github.com/relab/goxos/net"
	"github.com/relab/goxos/paxos"
	"github.com/relab/goxos/reconfig"
	"github.com/relab/goxos/ringreplacer"
)

// A Server is the main module that maintains communication with all of the
// submodules. It is also responsible for initializing and launching all of
// the submodules and their goroutines.
type Server struct {
	id                 grp.ID
	appID              string
	config             config.Config
	replacer           bool
	nodes              grp.NodeMap
	pxLeader           grp.ID
	dmx                net.Demuxer
	snd                *net.Sender
	outUnicast         chan net.Packet
	outBroadcast       chan interface{}
	outProposer        chan interface{}
	outAcceptor        chan interface{}
	outLearner         chan interface{}
	hbem               *liveness.HbEm
	fd                 *liveness.Fd
	ld                 liveness.LeaderDetector
	heartbeatChan      chan grp.ID
	prop               paxos.Proposer
	acc                paxos.Acceptor
	lrn                paxos.Learner
	pxLeaderChan       <-chan grp.ID
	propChan           chan *paxos.Value
	decidedChan        chan *paxos.Value
	propDcdChan        chan bool
	appStateReqChan    chan app.StateReq
	clientHandler      client.ClientHandler
	clientReqChan      chan *client.Request
	grpmgr             grp.GroupManager
	replacementHandler *lr.ReplacementHandler
	reconfigHandler    *reconfig.ReconfigHandler
	aReconfHandler     *arec.AReconfHandler
	ringReplacer       *ringreplacer.RingReplacer
	reconfigCmdChan    chan paxos.ReconfigCmd
	recMsgChan         chan ringreplacer.ReconfMsg
	localAru           *paxos.Adu
	firstSlot          paxos.SlotID
	ah                 app.Handler
	stopChan           chan bool
	subModulesStopSync *sync.WaitGroup
	batchTimeout       time.Duration
	batchMaxSize       uint
	batchBuffer        []*client.Request
	batchNextIndex     uint
	batchTimer         time.Timer
}

// Create a new Server for an application.
func NewServer(id grp.ID, appID string, conf config.Config,
	ah app.Handler) *Server {
	s := &Server{
		id:                 id,
		appID:              appID,
		config:             conf,
		pxLeader:           grp.UndefinedID(),
		outUnicast:         make(chan net.Packet, 64),
		outBroadcast:       make(chan interface{}, 64),
		outProposer:        make(chan interface{}, 64),
		outAcceptor:        make(chan interface{}, 64),
		outLearner:         make(chan interface{}, 64),
		heartbeatChan:      make(chan grp.ID, 128),
		propChan:           make(chan *paxos.Value, 512),
		decidedChan:        make(chan *paxos.Value, 512),
		propDcdChan:        make(chan bool, 32),
		clientReqChan:      make(chan *client.Request, 512),
		appStateReqChan:    make(chan app.StateReq),
		reconfigCmdChan:    make(chan paxos.ReconfigCmd),
		recMsgChan:         make(chan ringreplacer.ReconfMsg),
		localAru:           &paxos.Adu{},
		firstSlot:          1,
		ah:                 ah,
		stopChan:           make(chan bool),
		subModulesStopSync: new(sync.WaitGroup),
		batchTimeout:       conf.GetDuration("batchTimeout", config.DefBatchTimeout),
		batchMaxSize:       uint(conf.GetInt("batchMaxSize", config.DefBatchMaxSize)),
		batchTimer:         *time.NewTimer(0),
	}
	s.initNodeMap()
	return s
}
