package multipaxos

import (
	"testing"

	"github.com/relab/goxos/client"
	"github.com/relab/goxos/config"
	"github.com/relab/goxos/grp"
	"github.com/relab/goxos/liveness"
	px "github.com/relab/goxos/paxos"

	gc "github.com/relab/goxos/Godeps/_workspace/src/gopkg.in/check.v1"
)

// -----------------------------------------------------------------------
// Hook up gocheck into the "go test" runner
func TestMultiPaxos(t *testing.T) {
	gc.TestingT(t)
}

// -----------------------------------------------------------------------
// Common test data used across MultiPaxos actors

func genClientReq(cid, value string, seq uint32) client.Request {
	return client.Request{
		Type: client.Request_EXEC.Enum(),
		Id:   &cid,
		Seq:  &seq,
		Val:  []byte(value),
	}
}

// Client requests
var (
	reqFoo = genClientReq("clientx", "foo", 0)
	reqBar = genClientReq("clienty", "bar", 0)
)

// Paxos Slots
var (
	sid0 = px.SlotID(0)
	sid1 = px.SlotID(1)
	sid2 = px.SlotID(2)
	sid3 = px.SlotID(3)
	sid4 = px.SlotID(4)
)

// Paxos values
var (
	valFoo = px.Value{
		Vt: px.App,
		Cr: []*client.Request{&reqFoo},
	}

	valBar = px.Value{
		Vt: px.App,
		Cr: []*client.Request{&reqBar},
	}
)

// Replica IDs
var (
	r0id = grp.NewPxIDFromInt(0)
	r1id = grp.NewPxIDFromInt(1)
	r2id = grp.NewPxIDFromInt(2)
)

// Proposer rounds
var (
	rnd00 = px.ProposerRound{ID: r0id, Rnd: 0}
	rnd01 = px.ProposerRound{ID: r0id, Rnd: 1}
	rnd02 = px.ProposerRound{ID: r0id, Rnd: 2}
	rnd03 = px.ProposerRound{ID: r0id, Rnd: 3}
	rnd10 = px.ProposerRound{ID: r1id, Rnd: 0}
	rnd11 = px.ProposerRound{ID: r1id, Rnd: 1}
	rnd12 = px.ProposerRound{ID: r1id, Rnd: 2}
	rnd20 = px.ProposerRound{ID: r2id, Rnd: 0}
	rnd21 = px.ProposerRound{ID: r2id, Rnd: 1}
	rnd22 = px.ProposerRound{ID: r2id, Rnd: 2}
)

// Paxos Packs
var (
	ppThreeNodesNonLr = &px.Pack{
		ID:              r0id,
		Gm:              grp.NewGrpMgrMock(3),
		Ld:              liveness.NewMockLD(),
		NextExpectedDcd: 1,
		FirstSlot:       1,
		Config:          config.NewConfig(),
	}

	ppThreeNodesLr = &px.Pack{
		ID:              r0id,
		Gm:              grp.NewGrpMgrMockWithLr(3, []grp.Epoch{0, 0, 0}),
		Ld:              liveness.NewMockLD(),
		NextExpectedDcd: 1,
		FirstSlot:       1,
		Config:          config.NewConfig(),
	}
)
