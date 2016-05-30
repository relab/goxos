package config

import (
	"time"
)

// Default configuration settings for Goxos
const (
	// REQUIRED!
	// nodes: [id:hostname:paxosPort:clientPort], ...
	// Defines all nodes that should run. Comma separated list.
	DefNodes = ""

	// protocol: MultiPaxos | ParallelPaxos | BatchPaxos | AuthenticatedBC | ReliableBC
	DefProtocol = "MultiPaxos"

	// alpha: int
	// Alpha value used for pipelining
	DefAlpha = 3

	// batchMaxSize: int
	// Regular batching of requests before they are sent through paxos
	DefBatchMaxSize = 1

	// batchTimeout: duration
	// Regular batching of requests before they are sent through paxos
	DefBatchTimeout = 3000 * time.Microsecond

	// failureHandlingType: None | AReconfiguration | Reconfiguration | LiveReplacement
	DefFailureHandlingType = "None"

	// hbEmitterInterval: duration
	// How frequently do we emit heartbeats?
	DefHbEmitterInterval = 500 * time.Millisecond

	// fdTimeoutInterval: duration
	// How frequently does the FD module check for liveness?
	DefFdTimeoutInterval = 1000 * time.Millisecond

	// fdDeltaIncrease: duration
	// If node is falsely detected as crashed, how much do we increase
	// timeout by?
	DefFdDeltaIncrease = 250 * time.Millisecond

	// parallelPaxosProposers: int
	// Number of parallel proposers to run
	DefParallelPaxosProposers = 2

	// parallelPaxosAcceptors: int
	// Number of parallel acceptors to run
	DefParallelPaxosAcceptors = 2

	// parallelPaxosLearners: int
	// Number of parallel learners to run
	DefParallelPaxosLearners = 2

	// parallelPaxosLearners: duration
	// Maximum period of inactivity before batching occurs anyway
	DefBatchPaxosTimeout = 1000 * time.Millisecond

	// parallelPaxosLearners: int
	// Maximum number of requests in a batch
	DefBatchPaxosMaxSize = 100

	// nodeInitReplicaProvider: Disabled | Mock | Kvs
	// NodeInit specific settings
	DefNodeInitReplicaProvider = "Disabled"

	// nodeInitStandbys: [hostname:paxosPort:clientPort], ...
	// NodeInit specific settings
	DefNodeInitStandbys = ""

	// activationTimeout: int (milliseconds)
	DefActivationTimeout = 3000

	// LRStrategy: int (1 | 2 | 3)
	// Live Replacement strategy
	DefLRStrategy = 2

	// LRExpRndSleep: duration
	// If != 0, enable 0.5 prob for sleeping this duration before connecting to replacer
	DefLRExpRndSleep = 0

	// throughputSamplingInterval: duration
	// How often should the server log its throughput.
	// 0 turns off throughput logging.
	DefThroughputSamplingInterval = time.Duration(0)

	DefLivenessValues = "100,50,250"

	// Dunno if this is used:
	MinNrNodes = 3

	// AuthenticatedBC and ReliableBC specific:

	// configKeysPath: string
	// Path to keys used by authentication manager
	DefConfigKeysPath = "config-keys.json"

	// nrOfPublications: int
	// Number of publications to be published
	DefNrOfPublications = 100

	// publicationPayloadSize: int
	// Size of each publication in bytes
	DefPublicationPayloadSize = 32
)

// Default configuration settings for the Goxos client library
const (
	// cycleListMax: int
	// How many times should we loop through the node list when trying to tcp connect?
	DefCycleListMax = 3

	// cycleNodesWait: int (milliseconds)
	// How long should we wait between each connection attempt in tcp connect?
	DefCycleNodesWait = 1000 * time.Millisecond

	// dialTimeout: int (milliseconds)
	// What should the timeout be when dialing a replica in tcp connect?
	DefDialTimeout = 500 * time.Millisecond

	// writeTimeout: int (milliseconds)
	// What should the timeout be when writing/sending requests?
	DefWriteTimeout = 20000 * time.Millisecond

	// readTimeout: int (seconds)
	// What should the timeout be when reading responses?
	DefReadTimeout = 20 * time.Second

	// awaitResponseTimeout: int (seconds)
	// How long should we wait before timing out (wihtout a response) when sending requests?
	DefAwaitResponseTimeout = 60 * time.Second
)
