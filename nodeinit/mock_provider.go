package nodeinit

import (
	"github.com/relab/goxos/grp"
)

type MockReplicaProvider struct {
}

func NewMockReplicaProvider() *MockReplicaProvider {
	return &MockReplicaProvider{}
}

func (mrp *MockReplicaProvider) GetReplica(appID, failureHandlingType string) (grp.Node, error) {
	node := grp.NewNode("127.0.0.1", "9020", "9021", true, true, true)
	return node, nil
}
