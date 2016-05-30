package reliablebc

import (
	"encoding/gob"

	"github.com/relab/goxos/grp"
	"github.com/relab/goxos/paxos"
)

func init() {
	gob.Register(Publish{})
	gob.Register(Prepare{})
	gob.Register(Commit{})
	gob.Register(CommitA{})
	gob.Register(DcdRnd{})
}

type Publish struct {
	ID   grp.ID
	Rnd  uint
	Val  paxos.Value
	Hmac []byte
}

type Prepare struct {
	ID   grp.ID
	Rnd  uint
	Hash uint32
	Hmac []byte
}

type CommitA struct {
	ID   grp.ID
	Rnd  uint
	Hash uint32
	Hmac []byte
}

type Commit struct {
	ID   grp.ID
	Rnd  uint
	Hash uint32
	Hmac []byte
}

type DcdRnd struct {
	Rnd uint
}

//These will only be sent internally in the brokers
type ValReq struct {
	Rnd uint
}

type ValResp struct {
	Rnd uint
	Val paxos.Value
}
