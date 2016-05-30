package common

import (
	"fmt"
)

const (
	Read = iota
	Write
	Delete
)

var Commandtypes = [...]string{
	"Read",
	"Write",
	"Delete",
}

type CommandType int8

type MapRequest struct {
	Ct    CommandType
	Key   []byte
	Value []byte
}

type MapResponse struct {
	ToType CommandType
	Value  []byte
	Found  byte
	Err    []byte
}

func (ct CommandType) String() string {
	return Commandtypes[ct]
}

func (req MapRequest) String() string {
	switch req.Ct {
	case Read:
		return fmt.Sprintf("Read request for key %s", req.Key)
	case Write:
		return fmt.Sprintf("Write request for key %s with value %s", req.Key, req.Value)
	case Delete:
		return fmt.Sprintf("Delete request for key %s", req.Key)
	}

	return fmt.Sprintf("Unkown command type for map request (code %d)", req.Ct)
}

func (resp MapResponse) String() string {
	if len(resp.Err) != 0 {
		return fmt.Sprintf("Response to %v had error: %s", resp.ToType, resp.Err)
	}

	switch resp.ToType {
	case Read:
		if resp.Found != 0 {
			return fmt.Sprintf("Value for key: %s", resp.Value)
		}
		return fmt.Sprintf("Value for key was not found in map")
	case Write:
		return fmt.Sprintf("Write request for %q OK", resp.Value)
	case Delete:
		return fmt.Sprintf("Delete request OK")
	}

	return fmt.Sprintf("Unkown command type for map response (code %d)", resp.ToType)
}
