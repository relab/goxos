package net

import (
	"github.com/relab/goxos/grp"
)

// A Packet contains a message, as well as the destination id of the replica. Used
// to send a unicast message to a replica.
type Packet struct {
	DestID grp.ID
	Data   interface{}
}

// The IdExchange message is used for verifying a replica in the Connection phase.
type IDExchange struct {
	ID grp.ID
}

// The IdResponse message is used for verifying a replica in the Connection phase.
type IDResponse struct {
	Accepted bool
	Error    string
}
