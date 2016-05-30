package reliablebc

import (
	px "github.com/relab/goxos/paxos"
)

// Create the three reliable broadcast actors from the PaxosPack.
func CreateReliableBC(pp *px.Pack) (*RelPublisher, *RelAcceptor, *RelLearner) {
	p := NewRelPublisher(pp)
	a := NewRelAcceptor(pp)
	l := NewRelLearner(pp)
	return p, a, l
}
