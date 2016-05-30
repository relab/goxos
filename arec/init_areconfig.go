package arec

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

func (rh *AReconfHandler) reconfSetupLocal(rcmd *ReconfCmd) (
	rMsg *ReconfMsg, newNodes map[grp.Node]config.TransferWrapper, err error) {

	//Generate new epoch
	glog.V(2).Info("calculating new epoch")
	newEpoch := getNewEpoch(rh.id)
	glog.V(2).Infoln("new epoch was", newEpoch)

	//New and Old NodeMap
	oldMap := rh.grpmgr.NodeMap().CloneMap()
	newMap := make(map[grp.ID]grp.Node)
	newids := make([]grp.ID, 0, len(rcmd.repl)+rcmd.add)
	oldids := make(map[grp.ID]bool, len(oldMap))
	newNodes = make(map[grp.Node]config.TransferWrapper)

	//Fill new map with old nodes.
	for id, node := range oldMap {
		newMap[grp.NewID(id.PaxosID, newEpoch)] = node
		oldids[id] = true
	}

	//For every replacement:
	for _, repID := range rcmd.repl {
		if _, ok := oldMap[repID]; ok {
			// Try to get a replacer through our ReplacerProvider
			node, err2 := rh.replicaProvider.GetReplica(rh.appID, "lr")
			if err != nil {
				return nil, newNodes, err2
			}
			newid := grp.NewID(repID.PaxosID, newEpoch)
			delete(oldids, repID)
			newids = append(newids, newid)
			newMap[newid] = node
		} else {
			err = ErrInvalidID
			return nil, newNodes, err
		}
	}

	//Add new nodes:
	if rcmd.add > 0 {
		size := len(newMap)
		for i := 0; i < rcmd.add; i++ {
			node, err2 := rh.replicaProvider.GetReplica(rh.appID, "lr")
			if err2 != nil {
				return nil, newNodes, err2
			}
			newID := grp.NewID(grp.PaxosID(int8(size)), newEpoch)
			newMap[newID] = node
			newids = append(newids, newID)
			size++
		}
	}
	for _, id := range newids {
		replacedID := grp.NewID(id.PaxosID, rh.id.Epoch)
		delete(oldMap, replacedID)
		oldMap[id] = newMap[id]
		newNodes[newMap[id]] = config.TransferWrapper{
			ID:        id,
			NodeMap:   oldMap,
			ConfigMap: rh.config.CloneToKeyValueMap(),
		}
	}
	rMsg = &ReconfMsg{newEpoch, newMap, oldids, rh.adu.Value()}

	return rMsg, newNodes, nil

}

func (rh *AReconfHandler) getAppState() *app.State {
	glog.V(2).Info("requesting application state from server module")
	respChan := make(chan app.State)
	req := app.NewStateReq(respChan)
	rh.appStateReqChan <- req
	appState := <-respChan
	return &appState
}
