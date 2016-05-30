package grp

type GrpMgrMock struct {
	nrOfNodes uint
	arEnabled bool
	lrEnabled bool
	epochs    []Epoch
}

func (gmm *GrpMgrMock) Start() {}

func (gmm *GrpMgrMock) Stop() {}

func (gmm *GrpMgrMock) RequestHold(releaseChan chan bool) {}

func (gmm *GrpMgrMock) SubscribeToHold(name string) Subscriber {
	return Subscriber{}
}

func (gmm *GrpMgrMock) NodeMap() *NodeMap {
	return &NodeMap{}
}

func (gmm *GrpMgrMock) NrOfNodes() uint {
	return gmm.nrOfNodes
}

func (gmm *GrpMgrMock) NrOfAcceptors() uint {
	return 0
}

func (gmm *GrpMgrMock) Quorum() uint {
	return (gmm.nrOfNodes / 2) + 1
}

func (gmm *GrpMgrMock) Epochs() []Epoch {
	return gmm.epochs
}

func (gmm *GrpMgrMock) LrEnabled() bool {
	return gmm.lrEnabled
}

func (gmm *GrpMgrMock) ArEnabled() bool {
	return gmm.arEnabled
}

func (gmm *GrpMgrMock) SetNewNodeMap(nm map[ID]Node) {}

func NewGrpMgrMock(nrOfNodes uint) GroupManager {
	return &GrpMgrMock{nrOfNodes: nrOfNodes}
}

func NewGrpMgrMockWithLr(nrOfNodes uint, epochs []Epoch) GroupManager {
	return &GrpMgrMock{
		nrOfNodes: nrOfNodes,
		lrEnabled: true,
		epochs:    epochs,
	}
}

func (gmm *GrpMgrMock) GetID() ID {
	return undefinedID
}

func (gmm *GrpMgrMock) SetID(id ID) error {
	return nil
}
