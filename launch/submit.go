package main

import (
	"encoding/gob"
	"fmt"
	"log"
	"os"

	"github.com/relab/goxos/Godeps/_workspace/src/github.com/codegangsta/cli"
)

// submit -l label [-o path] [-e path] -p executable [args]
//
//               -o path  Where to send the stdout of the program.
//
//               -e path  Where to send the stderr of the program.

const submitName = "submit"

//TODO: How to document the optional arguments portion?
var submitCmd = cli.Command{
	Name:      submitName,
	ShortName: "s",
	Usage:     "submit a program to launch without configuration file",
	Action:    submit,
	Flags:     submitFlags,
}

var submitFlags = []cli.Flag{
	cli.StringFlag{
		Name:  "label, l",
		Value: "",
		Usage: "Unique label assigned to this service",
	},
	cli.StringFlag{
		Name:  "program, p",
		Value: "",
		Usage: "Program to execute",
	},
}

func submit(c *cli.Context) {
	label := c.String("label")
	prg := c.String("program")
	if label == "" || prg == "" {
		fmt.Println("Required flags not provided.")
		cli.ShowCommandHelp(c, submitName)
		os.Exit(-1)
	}
	srvInfo := &ServiceInfo{
		Label:       label,
		Program:     prg,
		ProgramArgs: c.Args(),
		Disabled:    false,
	}
	log.Printf("submitting %s for execution on ...\n", srvInfo.Program)
	services := make(map[string]*ServiceInfo)
	services[label] = srvInfo
	submitServices(services)
}

// Submit services in the provided map to launch for execution on the local
// machine. This function is also used for submitting from a configuration
// file, which may actually contain multiple map entries.
func submitServices(services map[string]*ServiceInfo) {
	err := ipcFunc(submitName, func(enc *gob.Encoder, dec *gob.Decoder) error {
		return enc.Encode(services)
	})
	if err != nil {
		log.Println(err)
	}
}

func submitHandler(enc *gob.Encoder, dec *gob.Decoder) error {
	var services map[string]*ServiceInfo
	err := dec.Decode(&services)
	if err != nil {
		log.Printf("Failed to decode: %v\n", err)
		return err
	}
	for _, srv := range services {
		go launch(srv)
	}
	return nil
}
