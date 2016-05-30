package grp

//TODO this interface is too big. Do we need a GroupManager interface? If so,
//what is the generic function of the group manager that could be implemented
//differently for different group managers?

// The RequestHold() and SubscribeToHold() should be removed: Instead Start()
// should return a channel on which requests to hold and corresponding
// releases should be sent. NO: this does not work, since Start() is only called once.

type GroupManager interface {
	Start()
	Stop()
	RequestHold(releaseChan chan bool)
	SubscribeToHold(name string) Subscriber
	NodeMap() *NodeMap
	NrOfNodes() uint
	NrOfAcceptors() uint
	Quorum() uint
	Epochs() []Epoch
	LrEnabled() bool
	ArEnabled() bool
	SetNewNodeMap(nm map[ID]Node)
	GetID() ID
	SetID(ID) error
}
