package nodeinit

import (
	"errors"

	"github.com/relab/goxos/grp"
)

var (
	ErrNoReplicaFound = errors.New("no available new replica found")
)

type ReplicaProvider interface {
	GetReplica(appID, failureHandlingType string) (grp.Node, error)
}
