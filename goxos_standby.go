package goxos

import (
	"fmt"
	"strings"

	"github.com/relab/goxos/app"
	"github.com/relab/goxos/config"
	"github.com/relab/goxos/elog"
	e "github.com/relab/goxos/elog/event"
	"github.com/relab/goxos/nodeinit"
	"github.com/relab/goxos/server"

	"github.com/relab/goxos/Godeps/_workspace/src/github.com/golang/glog"
)

type StandbyReplica struct {
	Replica
	initListener *nodeinit.InitListener
}

func NewStandbyReplica(ah app.Handler,
	appID, standbyIP string) (s *StandbyReplica) {
	r := *NewReplica(0, appID, *config.NewConfig(), ah)
	return &StandbyReplica{
		Replica:      r,
		initListener: nodeinit.NewInitListener(standbyIP),
	}
}

func (s *StandbyReplica) Standby() error {
	glog.V(1).Infoln("attempting to start node in standby mode")

	elog.Log(e.NewEvent(e.InitListening))

	err := s.initListener.Start()
	if err != nil {
		err = logAbortAndGenError("starting initialization listener failed", err)
		return err
	}
	glog.V(2).Info("initilization listener started")

	initData, err := s.initListener.WaitForInitData()
	glog.V(2).Info("received init data from listener")
	if err != nil {
		err = logAbortAndGenError("receiving initialization data failed", err)
		return err
	}

	glog.V(2).Info("attempting to set application state")
	err = s.ah.SetState(initData.AppState.State)
	if err != nil {
		err = logAbortAndGenError("setting application state failed", err)
		initData.ApplyStateResult(err)
		return err
	}

	glog.V(2).Info("converting config")
	nodes, conf, err := initData.Config.GenerateNodeMapAndConfig()
	if err != nil {
		glog.Error(err)
		panic(err)
	}
	s.server = server.NewStandbyServer(
		initData.ID,
		s.appID,
		nodes,
		*conf,
		s.ah,
		initData.AppState.SlotMarker,
	)

	fhType := conf.GetString("failureHandlingType", config.DefFailureHandlingType)
	switch strings.ToLower(fhType) {
	case "livereplacement":
		s.server.InitModules()
	case "reconfiguration":
		s.server.InitModulesReconfig()
	case "areconfiguration":
		s.server.InitModules()
	default:
		return fmt.Errorf("cannot init new node with failurehandling %v",
			conf.GetString("failureHandlingType", config.DefFailureHandlingType))
	}

	glog.V(2).Info("init modules done and application state set, responding to init listener")
	initData.ApplyStateResult(nil)
	s.initialized = true
	glog.V(1).Infoln("initialization success, creating and starting server for handlingtype", fhType)
	elog.Log(e.NewEvent(e.InitInitialized))

	switch strings.ToLower(fhType) {
	case "livereplacement":
		s.server.StartReplacer()
	case "reconfiguration":
		s.server.StartReconfig()
	case "areconfiguration":
		s.server.StartAReconfig()
	}

	s.started = true

	return nil
}

func (s *StandbyReplica) Init() error {
	return ErrMethodUnavailable
}

func (s *StandbyReplica) Start() error {
	return ErrMethodUnavailable
}

func logAbortAndGenError(event string, reason error) error {
	err := InitializationAbort{event, reason}
	glog.Error(err)
	return err
}

type InitializationAbort struct {
	event  string
	reason error
}

func (ia InitializationAbort) Error() string {
	return fmt.Sprintf("Initialization abort on event: %v, reason: %v",
		ia.event, ia.reason)
}
