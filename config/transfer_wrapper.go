package config

import (
	"encoding/gob"
	"errors"
	"fmt"
	"strings"

	"github.com/relab/goxos/grp"
)

// TODO: We should evaluate if there are good reasons for the
// functionality in this file, especially after the changes in commit
// 508c9c2477ae83c1
//
// Update 06.08.14, Tormod: Refactored as a temporary solution, but we
// should try to do this config transfer in a more clean way.
func init() {
	gob.Register(TransferWrapper{})
}

type TransferWrapper struct {
	ID                       grp.ID
	NodeMap                  map[grp.ID]grp.Node
	ConfigMap                map[string]string
	NrOfNodes, NrOfAcceptors uint
}

func (tw *TransferWrapper) GenerateNodeMapAndConfig() (grp.NodeMap, *Config, error) {
	var nodeMap *grp.NodeMap
	conf := newConfigFromValues(tw.ConfigMap)
	fhType := conf.GetString("failureHandlingType", DefFailureHandlingType)
	switch strings.ToLower(fhType) {
	case "reconfiguration", "areconfiguration":
		nodeMap = grp.NewNodeMap(tw.NodeMap)
	case "livereplacement":
		node, found := tw.NodeMap[tw.ID]
		if !found {
			return grp.NodeMap{},
				nil,
				errors.New("cannot genereate node map and" +
					"config, provided node map did not" +
					"contain replacer node",
				)
		}
		//nodeMap = grp.NewNodeMap(tw.NodeMap)
		nodeMap = grp.NewNodeMapFromSingleNode(
			tw.ID,
			node,
			uint(len(tw.NodeMap)),
			uint(len(tw.NodeMap)),
		)
	default:
		return grp.NodeMap{},
			nil,
			fmt.Errorf("cannot genereate node map and config for"+
				"failurehandling %v",
				conf.GetString(
					"failureHandlingType",
					DefFailureHandlingType,
				),
			)
	}

	return *nodeMap, conf, nil
}
