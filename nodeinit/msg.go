package nodeinit

import (
	"github.com/relab/goxos/app"
	"github.com/relab/goxos/config"
	"github.com/relab/goxos/grp"
)

type InitRequest struct {
	SenderID grp.ID
}

type InitResponse struct {
	Ack   bool
	State state
}

type TransferRequest struct {
	ID       grp.ID
	Config   config.TransferWrapper
	AppState app.State
}

type TransferResponse struct {
	Success     bool
	ErrorString string
}
