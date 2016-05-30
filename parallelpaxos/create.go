package parallelpaxos

import px "github.com/relab/goxos/paxos"

// Construct all of the actors for ParallelPaxos. Returns a ParallelProposer, ParallelAcceptor, and
// ParallelLearner.
func CreateParallelPaxos(pp *px.Pack) (*ParallelProposer, *ParallelAcceptor, *ParallelLearner) {
	if pp.Gm.LrEnabled() || pp.Gm.ArEnabled() {
		panic("Lr or Ar not supported in ParallelPaxos")
	}

	p := NewParallelProposer(pp)
	a := NewParallelAcceptor(pp)
	l := NewParallelLearner(pp)

	return p, a, l
}
