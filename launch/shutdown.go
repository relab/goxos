package main

import (
	"encoding/gob"
	"log"

	"github.com/relab/goxos/Godeps/_workspace/src/github.com/codegangsta/cli"
)

const shutdownName = "shutdown"

var shutdownCmd = cli.Command{
	Name:   shutdownName,
	Usage:  "shutdown launch (in serve mode) and all its services",
	Action: shutdown,
}

func shutdown(c *cli.Context) {
	err := ipcFunc(shutdownName, nil)
	if err != nil {
		log.Println(err)
	}
}

type ShutdownError struct{}

func (s ShutdownError) Error() string {
	return "Shutdown"
}

func shutdownHandler(enc *gob.Encoder, dec *gob.Decoder) error {
	log.Println("Shutdown in progress...")
	running.killall()
	return &ShutdownError{}
}
