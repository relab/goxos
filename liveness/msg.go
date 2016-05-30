package liveness

import (
	"encoding/gob"

	"github.com/relab/goxos/grp"
)

func init() {
	gob.Register(Heartbeat{})
}

// A Heartbeat message is used by replicas to indicate they are alive in
// periods when no other messages are sent.
type Heartbeat struct {
	ID grp.ID
}
