package nodeinit

import (
	"errors"
	"fmt"

	"github.com/relab/goxos/app"
	"github.com/relab/goxos/config"
	"github.com/relab/goxos/grp"
	"github.com/relab/goxos/net"

	"github.com/relab/goxos/elog"
	e "github.com/relab/goxos/elog/event"

	"github.com/relab/goxos/Godeps/_workspace/src/github.com/golang/glog"
)

type NodeType uint

const (
	ReplacerNode NodeType = iota
	ReconfigNode
	AReconfNode
)

func InitNode(nt NodeType, rn grp.Node, id grp.ID,
	config config.TransferWrapper, appState app.State) error {

	elog.Log(e.NewEvent(e.FailureHandlingInitStart))

	conn, err := net.ConnectToAddr(rn.IP + ":" + defaultActivationPort)
	if err != nil {
		return err
	}

	defer conn.Close()

	glog.V(2).Info("init request")
	if err = conn.Enc.Encode(InitRequest{}); err != nil {
		return err
	}

	glog.V(2).Info("reading init response")
	var iresp InitResponse
	if err := conn.Dec.Decode(&iresp); err != nil {
		return err
	}

	if iresp.Ack {
		glog.V(2).Info("replacer node responded available for init")
	} else {
		return fmt.Errorf("node could not be started due to state (%v)", iresp.State)
	}

	glog.V(2).Info("generating transfer request")
	treq := TransferRequest{ID: id, Config: config, AppState: appState}

	glog.V(2).Info("sending transfer request")
	if err = conn.Enc.Encode(treq); err != nil {
		return err
	}

	glog.V(2).Info("reading transfer response")
	var tresp TransferResponse
	if err := conn.Dec.Decode(&tresp); err != nil {
		return err
	}

	if !tresp.Success {
		return errors.New(tresp.ErrorString)
	}

	glog.V(2).Info("node responded with init success")
	elog.Log(e.NewEvent(e.FailureHandlingInitDone))

	return nil
}
