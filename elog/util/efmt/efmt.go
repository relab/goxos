package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/relab/goxos/elog/event"
)

func main() {
	var file = flag.String("file", "", "elog file to parse")
	var filter = flag.Bool("filter", true, "filter out throughput samples")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [OPTIONS]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nOptions:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	if *file == "" {
		flag.Usage()
		os.Exit(1)
	}

	events, err := event.Parse(*file)
	if err != nil {
		fmt.Println("Error parsing events:", err)
		return
	}

	if *filter {
		events, _ = event.ExtractThroughput(events)
	}

	for i, event := range events {
		fmt.Printf("%2d: %v\n", i, event)
	}
}
