package reconfig

import (
	"github.com/relab/goxos/app"
	"github.com/relab/goxos/config"
	"github.com/relab/goxos/grp"
	"github.com/relab/goxos/nodeinit"

	"github.com/relab/goxos/Godeps/_workspace/src/github.com/golang/glog"
)

func (rh *ReconfigHandler) prepareReconfig(newNodeID grp.ID) (
	map[grp.ID]grp.Node, error) {
	glog.V(2).Infoln("preparing reconfiguration for node", newNodeID)

	glog.V(2).Infoln("trying to obtain a new node through node provider")
	newNode, err := rh.replicaProvider.GetReplica(rh.appID, "reconfig")
	if err != nil {
		return nil, err
	}

	// Clone the existing node map
	newNodeMap := rh.grpmgr.NodeMap().CloneMap()

	// Replace old node with new
	newNodeMap[newNodeID] = newNode

	// Generate config
	glog.V(2).Infoln("generating config")
	configTransferWrapper := config.TransferWrapper{
		ID:        newNodeID,
		NodeMap:   newNodeMap,
		ConfigMap: rh.config.CloneToKeyValueMap(),
	}

	// Request application state
	appState := rh.getAppState()

	// Contact and initalize node
	if err = nodeinit.InitNode(
		nodeinit.ReconfigNode,
		newNode,
		newNodeID,
		configTransferWrapper,
		appState,
	); err != nil {
		return nil, err
	}

	return newNodeMap, nil
}

func (rh *ReconfigHandler) getAppState() app.State {
	glog.V(2).Info("requesting application state from server module")
	respChan := make(chan app.State)
	req := app.NewStateReq(respChan)
	rh.appStateReqChan <- req
	appState := <-respChan
	return appState
}
