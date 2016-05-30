package lr

import (
	"fmt"

	"github.com/relab/goxos/grp"
)

// ReplaceCmd encapsualtes a replacement command.
type ReplaceCmd struct {
	id  grp.ID
	err chan error
}

// NewReplaceCmd returns a replacement command for
// the replica with identifier id.
func NewReplaceCmd(id grp.ID) ReplaceCmd {
	return ReplaceCmd{id, make(chan error)}
}

// String returns a text representation of the
// replacement command.
func (rc ReplaceCmd) String() string {
	return fmt.Sprintf("Replace node %v", rc.id)
}
