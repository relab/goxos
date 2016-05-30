package net

type MockDemuxer struct{}

func NewMockDemuxer() *MockDemuxer {
	return &MockDemuxer{}
}

func (dmx *MockDemuxer) Start() {}

func (dmx *MockDemuxer) Stop() {}

func (dmx *MockDemuxer) RegisterChannel(ch interface{}) {}

func (dmx *MockDemuxer) RegisterFD() <-chan interface{} {
	return nil
}

func (dmx *MockDemuxer) HandleMessage(msg interface{}) {}
