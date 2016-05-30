package grp

import (
	"sync"
)

type Subscriber struct {
	prepareChan chan *sync.WaitGroup
	releaseChan chan bool
}

func (s *Subscriber) PrepareChan() <-chan *sync.WaitGroup {
	return s.prepareChan
}

func (s *Subscriber) ReleaseChan() <-chan bool {
	return s.releaseChan
}
