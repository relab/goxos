package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/relab/goxos/Godeps/_workspace/src/github.com/codegangsta/cli"
)

// save -l label [-o path] [-e path] -p executable [args]
//
//               -o path  Where to send the stdout of the program.
//
//               -e path  Where to send the stderr of the program.

const saveName = "save"

//TODO: How to document the optional arguments portion?
var saveCmd = cli.Command{
	Name:      saveName,
	ShortName: "s",
	Usage:     "save a program to a configuration file",
	Action:    save,
	Flags:     saveFlags,
}

var saveFlags = []cli.Flag{
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

func save(c *cli.Context) {
	label := c.String("label")
	prg := c.String("program")
	if label == "" || prg == "" {
		fmt.Println("Required flags not provided.")
		cli.ShowCommandHelp(c, "save")
		os.Exit(-1)
	}
	srvInfo := &ServiceInfo{
		Label:       label,
		Program:     prg,
		ProgramArgs: c.Args(),
		Disabled:    false,
	}
	log.Printf("saving %s to %s.%s\n", srvInfo.Program, prg, fileType)
	saveConfigFile(prg+"."+fileType, srvInfo)
}

func saveConfigFile(path string, srvInfo *ServiceInfo) {
	fd, err := os.Create(path)
	if err != nil {
		panic(err)
	}
	enc := json.NewEncoder(fd)
	if err = enc.Encode(srvInfo); err != nil {
		panic(err)
	}
}
