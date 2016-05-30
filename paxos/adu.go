package paxos

import (
	"fmt"
	"sync"
)

type Adu struct {
	mu  sync.Mutex
	val SlotID
}

func NewAdu(sid SlotID) *Adu {
	return &Adu{val: sid}
}

func (a *Adu) Increment() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.val++
}

func (a *Adu) Value() SlotID {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.val
}

func (a *Adu) String() string {
	return fmt.Sprintf("%v", a.val)
}
