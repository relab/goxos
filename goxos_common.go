package goxos

import (
	"errors"
)

var (
	ErrNodeNotInitialized            = errors.New("goxos node must be initialized before started")
	ErrCanNotStartAlreadyRunningNode = errors.New("can't start already running Goxos node")
	ErrCanNotStopNonRunningNode      = errors.New("can't stop non-runnning Goxos node")
	ErrMethodUnavailable             = errors.New("method unavailable for a Goxos replacer/reconfig node")
)
