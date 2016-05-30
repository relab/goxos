package nodeinit

import (
	"strings"

	"github.com/relab/goxos/Godeps/_workspace/src/github.com/golang/glog"
	"github.com/relab/goxos/config"
)

var rp ReplicaProvider

func GetReplicaProvider(pid int, cfg *config.Config) ReplicaProvider {
	if rp != nil {
		return rp
	}

	provider := cfg.GetString("nodeInitReplicaProvider", config.DefNodeInitReplicaProvider)
	switch strings.TrimSpace(strings.ToLower(provider)) {
	default:
		glog.Warningf("Could not understand config value for \"%s\": \"%s\". Using Disabled.",
			"nodeInitReplicaProvider", provider)
		fallthrough
	case "disabled":
		rp = NewNoopReplicaProvider()
	case "mock":
		rp = NewMockReplicaProvider()
	case "kvs":
		standbys, err := cfg.GetNodeList("nodeInitStandbys")
		if err != nil {
			glog.Warningln("No standby nodes given in configuration")
		}
		rp = NewKvsReplicaProvider(pid, standbys)
	}

	return rp
}
