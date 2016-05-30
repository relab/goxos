package event

import (
	"bytes"
	"encoding/gob"
	"io"
	"io/ioutil"
)

func Parse(filename string) ([]Event, error) {
	file, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	buf := bytes.NewBuffer(file)
	dec := gob.NewDecoder(buf)
	var events []Event

	for {
		var event Event
		err = dec.Decode(&event)
		if err != nil {
			if err == io.EOF {
				break
			}
			return events, err
		}
		events = append(events, event)
	}

	return events, nil
}

func ExtractThroughput(events []Event) (regular, throughput []Event) {
	regular = make([]Event, 0)
	throughput = make([]Event, 0)
	for _, event := range events {
		if event.Type == ThroughputSample {
			throughput = append(throughput, event)
		} else {
			regular = append(regular, event)
		}
	}

	return
}
