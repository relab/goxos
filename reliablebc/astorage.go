package reliablebc

import (
	"sync"

	"github.com/relab/goxos/grp"
	"github.com/relab/goxos/paxos"
)

//This is the data a reliable broadcast acceptor need for each round.
type AcceptorRound struct {
	RndID       uint
	Prepares    map[grp.ID]Prepare
	Commits     map[grp.ID]CommitA
	SentPrepare bool
	SentCommit  bool
	Value       paxos.Value
}

//Slice that stores acceptor rounds
type AcceptorStorage struct {
	m      sync.Mutex
	rounds []AcceptorRound
	min    uint
	length uint
}

func NewAcceptorStorage(length uint) *AcceptorStorage {
	return &AcceptorStorage{
		rounds: make([]AcceptorRound, length),
		min:    0,
		length: length,
	}
}

//Retrieves the correct value from the slice given a proposerround
func (as *AcceptorStorage) Get(rnd uint) *AcceptorRound {
	i := rnd
	if i >= as.min+as.length {
		as.reSlice()
	}
	return &as.rounds[i-as.min]
}

//Reslices the slice to make room for new values
func (as *AcceptorStorage) reSlice() {
	as.m.Lock()
	defer as.m.Unlock()
	i := as.length / 3
	as.rounds = append(as.rounds[i:], make([]AcceptorRound, i)...)
	as.min += i
}

//Checks if the round is still valid
func (as *AcceptorStorage) IsValid(rnd uint) bool {
	i := rnd
	return i >= as.min
}
