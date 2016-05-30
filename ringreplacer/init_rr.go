package ringreplacer

import (
	"github.com/relab/goxos/app"
	"github.com/relab/goxos/config"
	"github.com/relab/goxos/grp"
	ni "github.com/relab/goxos/nodeinit"

	"github.com/relab/goxos/Godeps/_workspace/src/github.com/golang/glog"
)

func reconfSetupGlobal(newNodes map[grp.Node]config.TransferWrapper,
	appState app.State) error {
	glog.V(2).Infoln("Initializing new Nodes")

	//TODO: Do this in a goroutine?
	for nd, conf := range newNodes {
		if err := ni.InitNode(ni.AReconfNode, nd,
			conf.ID, conf, appState); err != nil {
			glog.V(2).Infoln("Error while initializing new Node", nd)
			return err
		}
	}
	return nil
}

func (rr *RingReplacer) getNewConfigs(replace *[]grp.ID) (
	rMsg *ReconfMsg, newNodes map[grp.Node]config.TransferWrapper, err error) {

	//New and Old NodeMap
	newMap := rr.grpmgr.NodeMap().CloneMap()
	newids := make([]grp.ID, 0, len(*replace))
	newNodes = make(map[grp.Node]config.TransferWrapper)

	//For every replacement:
	for _, oldID := range *replace {
		// Try to get a replacer through our ReplacerProvider
		node, err2 := rr.replicaProvider.GetReplica(rr.appID, "lr")
		if err2 != nil {
			return nil, newNodes, err2
		}
		newEpoch := rr.getReplacerEpoch(oldID.Epoch)
		newid := grp.NewID(oldID.PaxosID, newEpoch)
		delete(newMap, oldID)
		newids = append(newids, newid)
		newMap[newid] = node
	}

	//Make Configurations for new Nodes
	for _, id := range newids {
		newNodes[newMap[id]] = config.TransferWrapper{
			ID:        id,
			NodeMap:   newMap,
			ConfigMap: rr.config.CloneToKeyValueMap(),
		}
	}
	rMsg = &ReconfMsg{newMap, newids, 0}

	return rMsg, newNodes, nil

}

func (rr *RingReplacer) getAppState() *app.State {
	glog.V(2).Info("requesting application state from server module")
	respChan := make(chan app.State)
	req := app.NewStateReq(respChan)
	rr.appStateReqChan <- req
	appState := <-respChan
	return &appState
}
