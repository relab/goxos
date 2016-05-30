package client

import (
	"log"
	"sync"
	"time"
)

type ResponseData struct {
	Value                 []byte
	SendTime, ReceiveTime time.Time
	Err                   error
}

type responseInfo struct {
	respChan    chan ResponseData
	resend      chan struct{}
	sendTime    time.Time
	gotResponse chan struct{}
}

func newResponseInfo() *responseInfo {
	return &responseInfo{
		respChan:    make(chan ResponseData, 1),
		resend:      make(chan struct{}, 1),
		sendTime:    time.Now(),
		gotResponse: make(chan struct{}, 1),
	}
}

type responseMap struct {
	respMap map[uint32]*responseInfo
	mapLock *sync.Mutex
}

func newResponseMap() responseMap {
	return responseMap{
		respMap: make(map[uint32]*responseInfo),
		mapLock: new(sync.Mutex),
	}
}

//TODO remove: only for debugging
func (m *responseMap) print() {
	for k := range m.respMap {
		log.Printf("key: %d\n", k)
	}
}

func (m *responseMap) delete(seqNum uint32) {
	m.mapLock.Lock()
	defer m.mapLock.Unlock()
	delete(m.respMap, seqNum)
}

func (m *responseMap) add(seqNum uint32, resp *responseInfo) {
	m.mapLock.Lock()
	defer m.mapLock.Unlock()
	m.respMap[seqNum] = resp
}

func (m *responseMap) get(seqNum uint32) (resp *responseInfo, exists bool) {
	m.mapLock.Lock()
	defer m.mapLock.Unlock()
	resp, exists = m.respMap[seqNum]
	return
}
