package nodeinit

import (
	"errors"

	"github.com/relab/goxos/grp"
)

type KvsReplicaProvider struct {
	counter  int
	standbys []grp.Node
}

func NewKvsReplicaProvider(startc int, standbys []grp.Node) *KvsReplicaProvider {
	return &KvsReplicaProvider{counter: (startc / 2) % len(standbys), standbys: standbys}
}

func (krp *KvsReplicaProvider) GetReplica(appID, failureHandlingType string) (grp.Node, error) {
	if len(krp.standbys) == 0 {
		return grp.Node{}, errors.New("no replica available")
	}
	replica := krp.standbys[krp.counter]
	if krp.counter == len(krp.standbys)-1 {
		krp.standbys = krp.standbys[:krp.counter]
		krp.counter = 0
	} else {
		krp.standbys = append(krp.standbys[:krp.counter], krp.standbys[krp.counter+1:]...)
	}
	return replica, nil
}
