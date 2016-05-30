package authenticatedbc

import (
	px "github.com/relab/goxos/paxos"
)

// Create the three authenticated broadcast actors from the PaxosPack.
func CreateAuthenticatedBC(pp *px.Pack) (*AuthAcceptor, *AuthPublisher, *AuthLearner) {
	a := NewAuthAcceptor(pp)
	p := NewAuthPublisher(pp)
	l := NewAuthLearner(pp)
	return a, p, l
}
