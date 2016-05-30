package authenticatedbc

import (
	"encoding/gob"

	"github.com/relab/goxos/grp"
	"github.com/relab/goxos/paxos"
)

func init() {
	gob.Register(Publish{})
	gob.Register(Forward{})
	gob.Register(DcdRnd{})
}

type Publish struct {
	ID   grp.ID
	Rnd  uint
	Val  paxos.Value
	Hmac []byte
}

type Forward struct {
	ID   grp.ID
	Hash uint32
	Rnd  uint
	Hmac []byte
}

type DcdRnd struct {
	Rnd uint
}

type ValReq struct {
	Rnd uint
}

type ValResp struct {
	Rnd uint
	Val paxos.Value
}
