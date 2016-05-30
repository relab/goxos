package arec

import (
	"fmt"

	"github.com/relab/goxos/grp"
)

// ReconfCmd encapsualtes a replacement command.
type ReconfCmd struct {
	repl []grp.ID
	add  int
}

// NewReconfCmdFromID returns a replacement command for
// the replica with identifier id.
func NewReconfCmdFromID(id grp.ID) *ReconfCmd {
	return &ReconfCmd{[]grp.ID{id}, 0}
}

// NewReconfCmdFromSlice returns a replacement command for
// the replicas specified.
func NewReconfCmdFromSlice(ids []grp.ID) ReconfCmd {
	return ReconfCmd{ids, 0}
}

// String returns a text representation of the
// replacement command.
func (rc ReconfCmd) String() string {
	return fmt.Sprintf("Replace nodes %v, add %v nodes", rc.repl, rc.add)
}
