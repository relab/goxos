package net

type Demuxer interface {
	Start()
	Stop()
	RegisterChannel(ch interface{})
	HandleMessage(msg interface{})
}
