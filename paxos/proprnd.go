package paxos

import (
	"github.com/relab/goxos/grp"
)

// The zero-value of a ProposerRound
var ZeroRound = ProposerRound{0, grp.NewIDFromInt(0, 0)}

// A ProposerRound, which is guaranteed to be unique among all the Proposers
// due to the fact that it is based on the Proposer's Id as well as the round
// number.
type ProposerRound struct {
	Rnd uint
	ID  grp.ID
}

func NewProposerRound(id grp.ID) *ProposerRound {
	return &ProposerRound{Rnd: 1, ID: id}
}

// Compare two ProposerRounds. Returns -1 if pr is less tha opr, 0 if they are
// equal, or 1 if pr is greater than opr.
func (pr *ProposerRound) Compare(opr ProposerRound) int {
	if pr.Rnd == opr.Rnd {
		return pr.ID.CompareTo(opr.ID)
	} else if pr.Rnd < opr.Rnd {
		return -1
	}
	return 1
}

// For Arec, can we replace the above Compare with the  following?
/*
func (pr *ProposerRound) Compare(opr ProposerRound) int {
	if pr.Id.Epoch != opr.Id.Epoch {
		return -1
	} else if pr.Rnd == opr.Rnd {
		if pr.Id.CompareTo(opr.Id) == -1 {
			return -1
		} else if pr.Id.CompareTo(opr.Id) == 1 {
			return 1
		} else {
			return 0
		}
	} else if pr.Rnd < opr.Rnd {
		return -1
	}
	return 1
}
*/

// Incremenent the ProposerRound.
func (pr *ProposerRound) Next() {
	pr.Rnd++
}

// Change the ownership of the ProposerRound.
func (pr *ProposerRound) SetOwner(id grp.ID) {
	pr.ID = id
}
