package paxos

import (
	"hash/fnv"

	"github.com/relab/goxos/client"
)

type ValueType int

const (
	Noop ValueType = iota
	App
	Reconfig
)

var valueTypes = [...]string{
	"No-op command",
	"Application command",
	"Reconfiguration command",
}

func (vt ValueType) String() string {
	return valueTypes[vt]
}

type Value struct {
	Vt ValueType
	Cr []*client.Request
	Rc *ReconfigCmd
}

func (v *Value) Equal(o Value) bool {
	if v.Vt != o.Vt {
		return false
	}

	switch v.Vt {
	case Noop:
		return true
	case App:
		if len(v.Cr) != len(o.Cr) {
			return false
		}

		for i := range v.Cr {
			if !v.Cr[i].Equal(*o.Cr[i]) {
				return false
			}
		}
		return true
	case Reconfig:
		// TODO: Implement equality for reconfig commands
	}

	return false
}

func (v *Value) Hash() uint32 {
	if v.Vt != App {
		panic("cannot hash non-app value")
	}

	h := fnv.New32a()
	for i := range v.Cr {
		h.Write(v.Cr[i].GetVal())
	}
	return h.Sum32()
}
