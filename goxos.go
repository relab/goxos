package goxos

import (
	"github.com/relab/goxos/app"
	"github.com/relab/goxos/config"
	"github.com/relab/goxos/elog"
	e "github.com/relab/goxos/elog/event"
	"github.com/relab/goxos/grp"
	"github.com/relab/goxos/server"

	"github.com/relab/goxos/Godeps/_workspace/src/github.com/golang/glog"
)

// Replica holds the state of a replicated service.
type Replica struct {
	started     bool
	id          grp.ID
	ah          app.Handler
	appID       string
	config      config.Config
	initialized bool
	server      *server.Server
}

// Create a new Goxos replica. Arguments required are an integer id, a string application id,
// a path to a configuration file, and an application that fulfills the app.Handler interface.
func NewReplica(id uint, appID string, config config.Config, ah app.Handler) *Replica {
	return &Replica{
		id:          grp.NewIDFromInt(int8(id), 0),
		appID:       appID,
		config:      config,
		ah:          ah,
		initialized: false,
	}
}

// Initialize the state.
func (r *Replica) Init() {
	glog.V(1).Info("initializing goxos node")

	r.server = server.NewServer(r.id, r.appID, r.config, r.ah)

	r.initialized = true
}

// Launch the server and all modules that make up Goxos.
func (r *Replica) Start() error {
	if !r.initialized {
		return ErrNodeNotInitialized
	}

	if r.started {
		return ErrCanNotStartAlreadyRunningNode
	}

	elog.Log(e.NewEvent(e.Start))
	glog.V(1).Info("starting node")

	r.server.InitModules()
	r.server.Start()
	r.started = true

	return nil
}

// Halt the server and all Goxos modules.
func (r *Replica) Stop() error {
	if !r.started {
		return ErrCanNotStopNonRunningNode
	}
	r.started = false
	glog.V(1).Info("stopping node")
	err := r.server.Stop()
	elog.Log(e.NewEvent(e.Exit))
	elog.Flush()
	return err
}
