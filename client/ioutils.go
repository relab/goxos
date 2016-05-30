package client

import (
	"encoding/binary"
	"io"
	"net"

	"github.com/relab/goxos/Godeps/_workspace/src/code.google.com/p/goprotobuf/proto"
)

// These functions are not thread-safe.

func write(conn net.Conn, msg proto.Message) error {
	buffer, err := proto.Marshal(msg)
	if err != nil {
		return err
	}
	err = binary.Write(conn, binary.LittleEndian, int32(len(buffer)))
	if err != nil {
		return err
	}
	_, err = conn.Write(buffer)
	// if Write was ok, err == nil
	return err
}

func read(conn net.Conn, msg proto.Message) error {
	var size int32
	err := binary.Read(conn, binary.LittleEndian, &size)
	if err != nil {
		return err
	}
	buffer := make([]byte, size)
	_, err = io.ReadFull(conn, buffer)
	if err != nil {
		return err
	}
	err = proto.Unmarshal(buffer, msg)
	// if Unmarshal was ok, err == nil
	return err
}
