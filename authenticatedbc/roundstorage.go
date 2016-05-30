package authenticatedbc

import (
	"sync"

	"github.com/relab/goxos/grp"
	"github.com/relab/goxos/paxos"
)

//This is the data a reliable broadcast learner require for each round.
type Round struct {
	RndID    uint
	Forwards map[grp.ID]Forward
	Decided  bool
	SntReq   bool
	Learned  bool
	Val      paxos.Value
}

//Slice that stores the learner rounds
type RoundStorage struct {
	m      sync.Mutex
	rounds []Round
	min    uint
	length uint
}

func NewRoundStorage(length uint) *RoundStorage {
	return &RoundStorage{
		rounds: make([]Round, length),
		min:    0,
		length: length,
	}
}

//Retrieves the correct value from the slice
func (ls *RoundStorage) Get(rnd uint) *Round {
	i := rnd
	if i >= ls.min+ls.length {
		ls.reSlice()
	}
	return &ls.rounds[i-ls.min]
}

//Reslices the slice to make room for new values
func (ls *RoundStorage) reSlice() {
	ls.m.Lock()
	defer ls.m.Unlock()
	i := ls.length / 3
	ls.rounds = append(ls.rounds[i:], make([]Round, i)...)
	ls.min += i
}

//Checks if the round is still valid
func (ls *RoundStorage) IsValid(rnd uint) bool {
	i := rnd
	return i >= ls.min
}
