package batchpaxos

import px "github.com/relab/goxos/paxos"

// Create all the necessary actors given the paxos pack, containing all the communication
// channels and startup information.
func CreateBatchPaxos(pp px.Pack) (*BatchProposer, *BatchAcceptor, *BatchLearner) {
	p := NewBatchProposer(pp)
	a := NewBatchAcceptor(pp)
	l := NewBatchLearner(pp)

	return p, a, l
}
