package multipaxos

import px "github.com/relab/goxos/paxos"

// Construct all of the actors for MultiPaxos. Returns a MultiProposer, MultiAcceptor, and
// MultiLearner.
func CreateMultiPaxos(pp *px.Pack) (*MultiProposer, *MultiAcceptor, *MultiLearner) {
	p := NewMultiProposer(pp)
	a := NewMultiAcceptor(pp)
	l := NewMultiLearner(pp)

	return p, a, l
}
