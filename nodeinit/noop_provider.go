package nodeinit

import (
	"errors"

	"github.com/relab/goxos/grp"
)

type NoopReplicaProvider struct {
}

func NewNoopReplicaProvider() *NoopReplicaProvider {
	return &NoopReplicaProvider{}
}

func (nrp *NoopReplicaProvider) GetReplica(appID, failureHandlingType string) (grp.Node, error) {
	return grp.Node{}, errors.New("peplica provider disabled, no available replica")
}
