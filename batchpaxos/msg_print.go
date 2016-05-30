package batchpaxos

import (
	"github.com/relab/goxos/Godeps/_workspace/src/github.com/golang/glog"
)

func (msg *BatchLearnMsg) Print() {
	glog.Infof("\tLEARN %d", msg.BatchID)

	ids := SortedKeysHSS(msg.HSS)

	for _, id := range ids {
		glog.Infof("\t\t%s: %d", id, msg.HSS[id])
	}
}

func (msg *BatchCommitMsg) Print() {
	glog.Infof("\tCOMMIT %d", msg.BatchID)

	ids := SortedKeysRange(msg.Ranges)

	for _, id := range ids {
		rng := msg.Ranges[id]
		glog.Infof("\t\t%s: [%d, %d]", id, rng.Start, rng.Stop)
	}
}

func (msg *UpdateRequestMsg) Print() {
	glog.Infof("\tUPDATE-REQUEST %d", msg.BatchID)

	ids := SortedKeysRange(msg.Ranges)

	for _, id := range ids {
		rng := msg.Ranges[id]
		glog.Infof("\t\t%s: [%d, %d]", id, rng.Start, rng.Stop)
	}
}

func (msg *UpdateReplyMsg) Print() {
	glog.Infof("\tUPDATE-REPLY %d", msg.BatchID)

	ids := SortedKeysRange(msg.Ranges)

	for _, id := range ids {
		rng := msg.Ranges[id]
		glog.Infof("\t\t%s: [%d, %d] %d %d", id, rng.Start, rng.Stop,
			rng.Stop-rng.Start+1, len(msg.Values[id]))
	}
}
