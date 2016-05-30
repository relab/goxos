package batchpaxos

import (
	"github.com/relab/goxos/client"
)

// A batch point is a mapping from client id (string) to sequence number
type BatchPoint map[string]uint32

// A range map is a mapping from client id (string) to client range
type RangeMap map[string]ClientRange

// An update value map is a mapping from client id (string) to slice of client requests
type UpdateValueMap map[string][]client.Request
