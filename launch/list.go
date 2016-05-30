package main

import (
	"encoding/gob"
	"log"

	"github.com/relab/goxos/Godeps/_workspace/src/github.com/codegangsta/cli"
)

// list [label]

const listName = "list"

//TODO: How to document the optional [label] argument?
var listCmd = cli.Command{
	Name:      listName,
	ShortName: "l",
	Usage:     "list all running services if no label is provided",
	Action:    list,
}

func list(c *cli.Context) {
	args := c.Args()
	label := ""
	if len(args) == 1 {
		label = args[0]
	}
	listServices(label)
}

func listServices(label string) {
	err := ipcFunc(listName, func(enc *gob.Encoder, dec *gob.Decoder) error {
		if e := enc.Encode(label); e != nil {
			return e
		}
		var srvList []string
		if e := dec.Decode(&srvList); e != nil {
			return e
		}
		for _, srv := range srvList {
			log.Println(srv)
		}
		return nil
	})
	if err != nil {
		log.Println(err)
	}
}

func listHandler(enc *gob.Encoder, dec *gob.Decoder) error {
	var label string
	err := dec.Decode(&label)
	if err != nil {
		log.Printf("Failed to decode: %v\n", err)
		return err
	}
	var r []string
	if label == "" {
		r = running.services()
	} else {
		if running.contains(label) {
			r = []string{label}
		} else {
			r = []string{"Service not running"}
		}
	}
	err = enc.Encode(r)
	if err != nil {
		log.Printf("Failed to encode: %v\n", err)
		return err
	}
	return nil
}
