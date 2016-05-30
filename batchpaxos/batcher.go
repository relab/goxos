package batchpaxos

import (
	"github.com/relab/goxos/client"
)

// A client log entry -- where client requests are actually stored, with a mapping
// from sequence number to request. Also maintained is the highest currently seen
// sequence number for this client.
type ClientLogEntry struct {
	Values      map[uint32]client.Request // Map from seq. no to request
	HighestSeen uint32                    // Highest seen seq. no
}

// A client range represents a range of client request from the start request
// to the stop request -- inclusive.
type ClientRange struct {
	Start uint32
	Stop  uint32
}

// A batch is tied to an id, as well as a mapping from client id to highest seen
// sequence number.
type Batch struct {
	ID  uint
	HSS map[string]uint32
}

// The batcher struct is used by the batch acceptor to easily keep track of the
// client requests as well as other batch information.
type Batcher struct {
	CLog         map[string]*ClientLogEntry
	BatchMap     map[uint]*Batch
	CurrentBatch uint
}

// Create a new batcher containing an initially empty client log, as well as an
// empty batch map.
func NewBatcher() *Batcher {
	return &Batcher{
		CLog:     make(map[string]*ClientLogEntry),
		BatchMap: make(map[uint]*Batch),
	}
}

// Get a client log for a particular client id.
func (br *Batcher) GetLog(cid string) *ClientLogEntry {
	_, exists := br.CLog[cid]

	if !exists {
		br.CLog[cid] = &ClientLogEntry{
			Values: make(map[uint32]client.Request),
		}
	}

	return br.CLog[cid]
}

// LogRequest will add the request to the log, indexed by the id of the client
// sending the request, as well as the sequence number.
func (br *Batcher) LogRequest(req client.Request) {
	id := req.GetId()
	seq := req.GetSeq()

	clog := br.GetLog(id)
	clog.Values[seq] = req

	if seq > clog.HighestSeen {
		clog.HighestSeen = seq
	}
}

// GetRequest will fetch a request from the log. Returns zero value of request if
// request is not found.
func (br *Batcher) GetRequest(id string, seq uint32) (req client.Request) {
	clog, logexists := br.CLog[id]

	if logexists {
		val, vexists := clog.Values[seq]
		if vexists {
			req = val
		}
	}

	return
}

// Check if any ranges in the given range map are incomplete -- i.e. if there are
// any missing requests on this replica. Returns the ids which don't have complete
// ranges.
func (br *Batcher) GetIncompleteRanges(rng RangeMap) RangeMap {
	inc := make(RangeMap)

	for cid, rng := range rng {
		log := br.GetLog(cid)
		for i := rng.Start; i <= rng.Stop; i++ {
			_, exists := log.Values[i]

			if !exists {
				inc[cid] = rng
				break
			}
		}
	}

	return inc
}

// Do a batchpoint -- could be the same as previous batchpoint, so this needs to be
// checked later.
func (br *Batcher) GetCurrentBatchPoint() BatchPoint {
	bp := make(BatchPoint)

	for cid, entry := range br.CLog {
		bp[cid] = entry.HighestSeen
	}

	return bp
}

// Helper function to loop over ranges in a range map in sorted order, passing each
// client request to the callback function passed to us.
func (br *Batcher) ForEach(ranges RangeMap, fn func(client.Request)) {
	// Sort
	ids := SortedKeysRange(ranges)

	for _, id := range ids {
		rng := ranges[id]
		for i := rng.Start; i <= rng.Stop; i++ {
			req := br.GetRequest(id, i)
			fn(req)
		}
	}
}

// Check whether we can batch -- i.e. if the latest batchpoint created is
// greater than the previous.
func (br *Batcher) GetBatch() *Batch {
	hss := br.GetCurrentBatchPoint()

	// Don't batch if no clients
	if len(hss) == 0 {
		return nil
	}

	// Growth in log since last time?
	if br.CurrentBatch == 0 {
		// First batch ==> Yes
		batch := &Batch{
			ID:  br.CurrentBatch,
			HSS: hss,
		}

		br.BatchMap[br.CurrentBatch] = batch
		br.CurrentBatch++
		return batch
	}

	prev := br.BatchMap[br.CurrentBatch-1]
	prevhss := prev.HSS
	batchable := false

	if len(hss) > len(prevhss) {
		// New client(s)
		batchable = true
	} else {
		// Check if clients HSS has grown
		for cid, hs := range hss {
			if hs > prevhss[cid] {
				batchable = true
				break
			}
		}
	}

	if batchable {
		batch := &Batch{
			ID:  br.CurrentBatch,
			HSS: hss,
		}

		br.BatchMap[br.CurrentBatch] = batch
		br.CurrentBatch++
		return batch
	}

	return nil
}

// Helper for the update protocol -- get a range of commands which can be included in
// a update reply message.
func (br *Batcher) GetClientCommands(ranges RangeMap) map[string][]client.Request {
	ret := make(map[string][]client.Request)

	for cid, rng := range ranges {
		ccmds := make([]client.Request, rng.Stop-rng.Start+1)

		for i := rng.Start; i <= rng.Stop; i++ {
			ccmds[i-rng.Start] = br.GetRequest(cid, i)
		}

		ret[cid] = ccmds
	}

	return ret
}

// Helper for the update protocol -- set a range of commands received in an update
// reply message.
func (br *Batcher) SetClientCommands(cmdmap UpdateValueMap) {
	for _, cmds := range cmdmap {
		for _, cmd := range cmds {
			br.LogRequest(cmd)
		}
	}
}
