package main

import (
	"bytes"

	kc "github.com/relab/goxos/kvs/common"

	"github.com/relab/goxos/Godeps/_workspace/src/github.com/golang/glog"
)

type GoxosHandler struct {
	kvmap map[string][]byte
}

var (
	buffer = new(bytes.Buffer)
	kvreq  kc.MapRequest
	kvresp kc.MapResponse
)

func (gh *GoxosHandler) Execute(req []byte) (resp []byte) {
	buffer.Reset()
	buffer.Write(req)

	if err := kvreq.Unmarshal(buffer); err != nil {
		glog.Errorln("Execute: Unmarshal error:", err)
		buffer.Reset()
		kvresp = kc.MapResponse{
			Err: []byte("I can't decode you request"),
		}
		kvresp.Marshal(buffer)
		return buffer.Bytes()
	}

	if glog.V(3) {
		glog.Info(kvreq)
	}

	switch kvreq.Ct {

	case kc.Read:
		val, found := gh.kvmap[string(kvreq.Key)]
		kvresp = kc.MapResponse{
			Value:  val,
			ToType: kc.Read,
		}
		if found {
			kvresp.Found = 1
		} else {
			kvresp.Found = 0
		}
	case kc.Write:
		gh.kvmap[string(kvreq.Key)] = kvreq.Value
		kvresp = kc.MapResponse{
			Value:  kvreq.Value,
			ToType: kc.Write,
		}
	case kc.Delete:
		delete(gh.kvmap, string(kvreq.Key))
		kvresp = kc.MapResponse{
			ToType: kc.Delete,
		}
	default:
		kvresp = kc.MapResponse{Err: []byte("Unkown map command")}

	}

	buffer.Reset()
	kvresp.Marshal(buffer)

	if glog.V(3) {
		glog.Info(kvresp)
	}

	resp = make([]byte, buffer.Len())
	if _, err := buffer.Read(resp); err != nil {
		glog.Errorln("Execute:", err)
	}

	return resp
}
