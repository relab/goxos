package main

import (
	"github.com/relab/goxos/Godeps/_workspace/src/github.com/golang/glog"
)

type GoxosHandler struct {
}

func (gh *GoxosHandler) Execute(req []byte) (resp []byte) {
	glog.V(2).Infof("Request received from execute: %s\n", req)
	return req
}

func (gh *GoxosHandler) GetState(slotMarker uint) (sm uint, state []byte) {
	return 0, nil
}

func (gh *GoxosHandler) SetState(state []byte) error {
	return nil
}
