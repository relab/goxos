package main

import (
	"encoding/gob"
	"log"

	"github.com/relab/goxos/Godeps/_workspace/src/github.com/codegangsta/cli"
)

const stopName = "stop"

var stopCmd = cli.Command{
	Name:   stopName,
	Usage:  "stop services corresponding to the provide labels",
	Action: stop,
}

func stop(c *cli.Context) {
	labels := c.Args()
	if len(labels) == 0 {
		log.Fatalln("No services to stop provided.")
	}
	stopServices(labels)
}

func stopServices(labels []string) {
	err := ipcFunc(stopName, func(enc *gob.Encoder, dec *gob.Decoder) error {
		return enc.Encode(labels)
	})
	if err != nil {
		log.Println(err)
	}
}

func stopHandler(enc *gob.Encoder, dec *gob.Decoder) error {
	var labels []string
	err := dec.Decode(&labels)
	if err != nil {
		log.Printf("Failed to decode: %v\n", err)
		return err
	}
	for _, label := range labels {
		if running.contains(label) {
			err = running.entry(label).Process.Kill()
			if err != nil {
				log.Printf("%s could not be stopped: %v\n", label, err)
				return err
			}
		} else {
			log.Printf("%s is not running.\n", label)
		}
	}
	return nil
}
