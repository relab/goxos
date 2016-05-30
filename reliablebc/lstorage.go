package reliablebc

import (
	"sync"

	"github.com/relab/goxos/grp"
	"github.com/relab/goxos/paxos"
)

//This is the data a reliable broadcast learner require for each round.
type LearnerRound struct {
	RndID   uint
	Commits map[grp.ID]Commit
	Decided bool
	SntReq  bool
	Learned bool
	Val     paxos.Value
}

//Slice that stores the learner rounds
type LearnerStorage struct {
	m      sync.Mutex
	rounds []LearnerRound
	min    uint
	length uint
}

func NewLearnerStorage(length uint) *LearnerStorage {
	return &LearnerStorage{
		rounds: make([]LearnerRound, length),
		min:    0,
		length: length,
	}
}

//Retrieves the correct value from the slice
func (ls *LearnerStorage) Get(rnd uint) *LearnerRound {
	i := rnd
	if i >= ls.min+ls.length {
		ls.reSlice()
	}
	return &ls.rounds[i-ls.min]
}

//Reslices the slice to make room for new values
func (ls *LearnerStorage) reSlice() {
	ls.m.Lock()
	defer ls.m.Unlock()
	i := ls.length / 3
	ls.rounds = append(ls.rounds[i:], make([]LearnerRound, i)...)
	ls.min += i
}

//Checks if the round is still valid
func (ls *LearnerStorage) IsValid(rnd uint) bool {
	i := rnd
	return i >= ls.min
}
