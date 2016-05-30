package main

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"io/ioutil"
	"log"

	"github.com/relab/goxos/kvs/bgen"
)

const MB = 1024 * 1024

func main() {
	if err := generate(1*MB, 16, 16); err != nil {
		log.Fatal(err)
	}
	if err := generate(10*MB, 16, 16); err != nil {
		log.Fatal(err)
	}
	if err := generate(1*MB, 16, 1008); err != nil {
		log.Fatal(err)
	}
	if err := generate(10*MB, 16, 1008); err != nil {
		log.Fatal(err)
	}
	if err := generate(3*32, 16, 16); err != nil {
		log.Fatal(err)
	}
}

func generate(stateSizeB int, keySize, valSize int) error {
	entries := stateSizeB / (keySize + valSize)
	keys := make([][]byte, entries)
	state := make(map[string][]byte, entries)
	for i := 0; i < entries; i++ {
		key := make([]byte, keySize)
		bgen.GetBytes(key)
		if _, found := state[string(key)]; found {
			return errors.New("duplicate")
		}
		keys[i] = key
		value := make([]byte, valSize)
		bgen.GetBytes(value)
		state[string(key)] = value
	}

	buffer := new(bytes.Buffer)
	encoder := gob.NewEncoder(buffer)
	var keyname, statename string
	if stateSizeB >= MB {
		stateSizeMB := stateSizeB / MB
		keyname = fmt.Sprintf("keys-%dMB-%dB+%dB.gob", stateSizeMB, keySize, valSize)
		statename = fmt.Sprintf("state-%dMB-%dB+%dB.gob", stateSizeMB, keySize, valSize)
	} else {
		keyname = fmt.Sprintf("keys-%dB-%dB+%dB.gob", stateSizeB, keySize, valSize)
		statename = fmt.Sprintf("state-%dB-%dB+%dB.gob", stateSizeB, keySize, valSize)
	}

	if err := encoder.Encode(keys); err != nil {
		return err
	}
	if err := ioutil.WriteFile(keyname, buffer.Bytes(), 0644); err != nil {
		return err
	}

	buffer.Reset()
	encoder = gob.NewEncoder(buffer)
	if err := encoder.Encode(state); err != nil {
		return err
	}
	if err := ioutil.WriteFile(statename, buffer.Bytes(), 0644); err != nil {
		return err
	}

	return nil
}
